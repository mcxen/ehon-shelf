package main

import (
	"bytes"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Base is the site; tests point it at a local server.
var Base = "https://lapis.nichibun.ac.jp"

const (
	UA       = "NichibunEnpArchiver/1.0 (EhonShelf/0.2; offline mirror; polite 1 req/s)"
	MinDelay = 1.0
)

var bibKeys = []string{"通番", "絵師名", "よみがな１", "作品名", "よみがな２", "技法", "版型", "点数", "板行年", "西暦", "作品ﾌｧｲﾙNo.", "備考", "請求記号", "図書ID"}

var (
	errPaused  = errors.New("paused")
	errRefused = errors.New("服务器拒绝访问，可能已被限流，请稍后再继续")
)

// Status is a snapshot of the crawler, shown by the settings view.
type Status struct {
	State      string // idle running pausing paused done error
	Current    *Item
	ItemDone   int
	ItemTotal  int
	DoneItems  int
	TotalItems int
	QueueLen   int
	Message    string
	Errors     []string
}

// Crawler downloads works one request at a time, politely and resumably.
type Crawler struct {
	DataDir  string
	Delay    time.Duration
	OnChange func() // called from the crawler's goroutine after each change

	client   *http.Client
	reqMu    sync.Mutex
	mu       sync.Mutex
	last     time.Time
	disallow []string
	robotsOK bool
	pause    bool
	running  bool
	queue    []Item
	st       Status
}

func NewCrawler(dataDir string, delay float64) *Crawler {
	if delay < MinDelay {
		delay = MinDelay
	}
	c := &Crawler{DataDir: dataDir, Delay: time.Duration(delay * float64(time.Second)),
		client: newClient(), st: Status{State: "idle"}}
	c.queue = c.loadQueue()
	if len(c.queue) > 0 {
		c.st.State, c.st.Message, c.st.QueueLen = "paused", "上次未完成的下载可继续", len(c.queue)
		c.st.TotalItems = len(c.queue)
	}
	return c
}

func (c *Crawler) queuePath() string { return filepath.Join(c.DataDir, "queue.json") }

func (c *Crawler) loadQueue() []Item {
	var q []Item
	if b, err := os.ReadFile(c.queuePath()); err == nil {
		_ = jsonUnmarshal(b, &q)
	}
	return q
}

func (c *Crawler) saveQueue() {
	if len(c.queue) == 0 {
		os.Remove(c.queuePath())
		return
	}
	_ = os.MkdirAll(c.DataDir, 0o755)
	_ = writeJSON(c.queuePath(), c.queue)
}

func (c *Crawler) Snapshot() Status {
	c.mu.Lock()
	defer c.mu.Unlock()
	s := c.st
	s.Errors = append([]string(nil), c.st.Errors...)
	s.QueueLen = len(c.queue)
	return s
}

func (c *Crawler) Running() bool { c.mu.Lock(); defer c.mu.Unlock(); return c.running }

func (c *Crawler) update(fn func(s *Status)) {
	c.mu.Lock()
	fn(&c.st)
	c.mu.Unlock()
	if c.OnChange != nil {
		c.OnChange()
	}
}

func (c *Crawler) paused() bool { c.mu.Lock(); defer c.mu.Unlock(); return c.pause }

// --- HTTP ---

func (c *Crawler) robots() {
	if c.robotsOK {
		return
	}
	c.robotsOK = true
	c.last = time.Now()
	req, _ := http.NewRequest("GET", Base+"/robots.txt", nil)
	setHeaders(req)
	resp, err := c.client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return
	}
	b, _ := io.ReadAll(resp.Body)
	c.disallow = parseRobots(string(b), "NichibunEnpArchiver")
}

// parseRobots returns the Disallow prefixes that apply to agent (or to *).
func parseRobots(txt, agent string) []string {
	groups := map[string][]string{}
	var agents []string
	inRules := false
	for _, line := range strings.Split(txt, "\n") {
		if i := strings.IndexByte(line, '#'); i >= 0 {
			line = line[:i]
		}
		k, v, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		k, v = strings.ToLower(strings.TrimSpace(k)), strings.TrimSpace(v)
		switch k {
		case "user-agent":
			if inRules {
				agents, inRules = nil, false
			}
			agents = append(agents, strings.ToLower(v))
		case "disallow", "allow":
			inRules = true
			if k == "disallow" && v != "" {
				for _, a := range agents {
					groups[a] = append(groups[a], v)
				}
			}
		}
	}
	for a, rules := range groups {
		if a != "*" && strings.Contains(strings.ToLower(agent), a) {
			return rules
		}
	}
	return groups["*"]
}

func (c *Crawler) get(path string) ([]byte, error) {
	c.reqMu.Lock()
	defer c.reqMu.Unlock()
	c.robots()
	for _, d := range c.disallow {
		if strings.HasPrefix(path, d) {
			return nil, fmt.Errorf("robots.txt 禁止访问 %s", path)
		}
	}
	refused := 0
	for attempt := 0; attempt < 6; attempt++ {
		if c.paused() {
			return nil, errPaused
		}
		if wait := c.Delay - time.Since(c.last); wait > 0 {
			time.Sleep(wait)
		}
		c.last = time.Now()
		req, _ := http.NewRequest("GET", Base+path, nil)
		setHeaders(req)
		resp, err := c.client.Do(req)
		if err == nil {
			b, rerr := io.ReadAll(resp.Body)
			resp.Body.Close()
			switch {
			case resp.StatusCode == 200 && rerr == nil:
				return b, nil
			case resp.StatusCode == 404:
				return nil, nil
			}
			log.Printf("HTTP %d %s", resp.StatusCode, path)
			if resp.StatusCode == 431 {
				// Header too large: usually cookies piling up. Start a fresh jar and retry.
				c.client.Jar, _ = cookiejar.New(nil)
				continue
			}
			if resp.StatusCode == 403 || resp.StatusCode == 429 {
				// The site's gateway refuses us: slow down a lot, then stop.
				if refused++; refused >= 3 {
					return nil, fmt.Errorf("HTTP %d: %w", resp.StatusCode, errRefused)
				}
				attempt += 2
			}
		} else {
			log.Printf("error %v %s", err, path)
		}
		backoff := time.Duration(min(120, 5<<attempt)) * time.Second
		for t := time.Duration(0); t < backoff; t += time.Second {
			if c.paused() {
				return nil, errPaused
			}
			time.Sleep(time.Second)
		}
	}
	return nil, fmt.Errorf("放弃 %s", path)
}

func (c *Crawler) getText(path string) (string, error) {
	b, err := c.get(path)
	return string(b), err
}

// --- parsing ---

var (
	reTag   = regexp.MustCompile(`<[^>]+>`)
	reSpace = regexp.MustCompile(`\s+`)
	reList  = regexp.MustCompile(`(?s)href="/enp/Cover/List/(\d+)".*?class="s">(.*?)</span>.*?class="ss">(.*?)</span>.*?class="sblue">(.*?)</span>`)
	reTD    = regexp.MustCompile(`(?s)<td[^>]*>(.*?)</td>`)
	reDCT   = regexp.MustCompile(`(?s)/DCTDecode.*?stream\r?\n`)
)

func txt(s string) string {
	return strings.TrimSpace(reSpace.ReplaceAllString(html.UnescapeString(reTag.ReplaceAllString(s, " ")), " "))
}

func ParseList(h string) []Item {
	var items []Item
	for _, m := range reList.FindAllStringSubmatch(h, -1) {
		id, _ := strconv.Atoi(m[1])
		items = append(items, Item{ID: id, Title: txt(m[2]), TitleKana: txt(m[3]), Artist: txt(m[4])})
	}
	return items
}

func (c *Crawler) ListItems() ([]Item, error) {
	h, err := c.getText("/enp/SearchBooks/BySakuContents")
	if err != nil {
		return nil, err
	}
	items := ParseList(h)
	if len(items) == 0 {
		return nil, errors.New("目录为空，网站结构可能已变化")
	}
	return items, nil
}

func ParseShoshi(h string) map[string]string {
	var cells []string
	for _, m := range reTD.FindAllStringSubmatch(h, -1) {
		cells = append(cells, txt(m[1]))
	}
	isKey := func(s string) bool {
		for _, k := range bibKeys {
			if k == s {
				return true
			}
		}
		return false
	}
	d := map[string]string{}
	for i := 0; i+1 < len(cells); i++ {
		k, v := cells[i], cells[i+1]
		if _, seen := d[k]; isKey(k) && !seen {
			if isKey(v) {
				v = ""
			}
			d[k] = v
		}
	}
	return d
}

type pageRef struct {
	n    int
	name string
}

func ParseVolumePage(h string, iid, vol int) []pageRef {
	re1 := regexp.MustCompile(fmt.Sprintf(`href="/enp/Picture/View/%d/%d/(\d+)"\s*>?\s*<img src="/enp/image/2/([^"]+)\.jpg"`, iid, vol))
	ms := re1.FindAllStringSubmatch(h, -1)
	if len(ms) == 0 {
		re2 := regexp.MustCompile(fmt.Sprintf(`(?s)View/%d/%d/(\d+)".{0,200}?image/2/([^"]+)\.jpg`, iid, vol))
		ms = re2.FindAllStringSubmatch(h, -1)
	}
	var out []pageRef
	for _, m := range ms {
		n, _ := strconv.Atoi(m[1])
		out = append(out, pageRef{n, m[2]})
	}
	return out
}

func (c *Crawler) volumePages(iid, vol int) ([]pageRef, error) {
	var names []pageRef
	seen := map[pageRef]bool{}
	for pg := 1; ; pg++ {
		h, err := c.getText(fmt.Sprintf("/enp/Picture/ListContents/%d/%d/%d", iid, vol, pg))
		if err != nil {
			return nil, err
		}
		for _, p := range ParseVolumePage(h, iid, vol) {
			if !seen[p] {
				seen[p] = true
				names = append(names, p)
			}
		}
		if !strings.Contains(h, fmt.Sprintf(`Picture/List/%d/%d/%d"`, iid, vol, pg+1)) {
			break
		}
	}
	sort.Slice(names, func(i, j int) bool {
		if names[i].n != names[j].n {
			return names[i].n < names[j].n
		}
		return names[i].name < names[j].name
	})
	return names, nil
}

// JPEGFromPDF returns the largest embedded DCTDecode JPEG stream, unchanged.
func JPEGFromPDF(pdf []byte) []byte {
	var best []byte
	for _, loc := range reDCT.FindAllIndex(pdf, -1) {
		start := loc[1]
		end := bytes.Index(pdf[start:], []byte("endstream"))
		if end < 0 {
			continue
		}
		data := bytes.TrimRight(pdf[start:start+end], "\r\n")
		if len(data) > 2 && data[0] == 0xff && data[1] == 0xd8 && len(data) > len(best) {
			best = data
		}
	}
	return best
}

func (c *Crawler) crawlItem(it Item) error {
	d := filepath.Join(c.DataDir, strconv.Itoa(it.ID))
	if err := os.MkdirAll(filepath.Join(d, "pages"), 0o755); err != nil {
		return err
	}
	mp := filepath.Join(d, "meta.json")
	meta, _ := readMeta(mp)
	if meta == nil || len(meta.Pages) == 0 {
		c.update(func(s *Status) { s.Message = "读取目录信息" })
		cover, err := c.getText(fmt.Sprintf("/enp/Cover/List/%d", it.ID))
		if err != nil {
			return err
		}
		vs := map[int]bool{}
		for _, m := range regexp.MustCompile(fmt.Sprintf(`/enp/Picture/List/%d/(\d+)/1"`, it.ID)).FindAllStringSubmatch(cover, -1) {
			v, _ := strconv.Atoi(m[1])
			vs[v] = true
		}
		var vols []int
		for v := range vs {
			vols = append(vols, v)
		}
		sort.Ints(vols)
		if len(vols) == 0 {
			vols = []int{1}
		}
		sh, err := c.getText(fmt.Sprintf("/enp/Book/Shoshi/%d", it.ID))
		if err != nil {
			return err
		}
		meta = &Meta{Item: it, Source: fmt.Sprintf("%s/enp/Cover/List/%d", Base, it.ID), Bibliographic: ParseShoshi(sh), Volumes: vols}
		for _, v := range vols {
			refs, err := c.volumePages(it.ID, v)
			if err != nil {
				return err
			}
			for _, r := range refs {
				meta.Pages = append(meta.Pages, Page{Vol: v, N: r.n, Name: r.name,
					PDF: "/enp/image/1/" + r.name + ".pdf", Thumb: "/enp/image/2/" + r.name + ".jpg"})
			}
		}
		for i := range meta.Pages {
			meta.Pages[i].File = fmt.Sprintf("pages/%04d.jpg", i+1)
		}
		meta.Date = meta.Bibliographic["板行年"]
		if meta.Date == "" {
			meta.Date = meta.Bibliographic["西暦"]
		}
		meta.PageCount = len(meta.Pages)
		if err := writeJSON(mp, meta); err != nil {
			return err
		}
	}
	c.update(func(s *Status) { s.ItemTotal = len(meta.Pages) })
	for i, p := range meta.Pages {
		c.update(func(s *Status) { s.ItemDone = i })
		fp := filepath.Join(d, filepath.FromSlash(p.File))
		if st, err := os.Stat(fp); err == nil && st.Size() > 0 {
			continue
		}
		c.update(func(s *Status) { s.Message = "下载 " + p.File })
		pdf, err := c.get(p.PDF)
		if err != nil {
			return err
		}
		jpg := JPEGFromPDF(pdf)
		if jpg == nil {
			log.Printf("no full-res for %s, using thumbnail", p.Name)
			if jpg, err = c.get(p.Thumb); err != nil {
				return err
			}
			meta.Pages[i].Lowres = true
		}
		if err := writeAtomic(fp, jpg); err != nil {
			return err
		}
	}
	c.update(func(s *Status) { s.ItemDone = len(meta.Pages); s.Message = "生成封面" })
	if _, err := MakeCover(d); err != nil {
		log.Printf("cover %d: %v", it.ID, err)
	}
	meta.Complete = true
	return writeJSON(mp, meta)
}

// --- background job ---

// Start queues items (skipping those already queued) and runs the queue.
func (c *Crawler) Start(items []Item) {
	c.mu.Lock()
	have := map[int]bool{}
	for _, q := range c.queue {
		have[q.ID] = true
	}
	added := 0
	for _, it := range items {
		if !have[it.ID] {
			c.queue = append(c.queue, it)
			have[it.ID] = true
			added++
		}
	}
	if !c.running {
		c.st.DoneItems, c.st.TotalItems, c.st.Errors = 0, len(c.queue), nil
	} else {
		c.st.TotalItems += added
	}
	c.saveQueue()
	c.mu.Unlock()
	c.Resume()
}

func (c *Crawler) Pause() {
	c.mu.Lock()
	if c.running {
		c.pause = true
		c.st.State = "pausing"
	}
	c.mu.Unlock()
	if c.OnChange != nil {
		c.OnChange()
	}
}

func (c *Crawler) Resume() {
	c.mu.Lock()
	if c.running || len(c.queue) == 0 {
		c.mu.Unlock()
		return
	}
	c.running, c.pause = true, false
	if c.st.TotalItems < len(c.queue) {
		c.st.TotalItems = len(c.queue) + c.st.DoneItems
	}
	c.st.State, c.st.Message = "running", ""
	c.mu.Unlock()
	go c.run()
}

func (c *Crawler) run() {
	defer func() {
		c.mu.Lock()
		c.running = false
		c.mu.Unlock()
		if c.OnChange != nil {
			c.OnChange()
		}
	}()
	for {
		c.mu.Lock()
		if len(c.queue) == 0 {
			c.st.State, c.st.Current, c.st.Message = "done", nil, "全部完成"
			c.mu.Unlock()
			return
		}
		it := c.queue[0]
		c.st.Current, c.st.ItemDone, c.st.ItemTotal = &it, 0, 0
		c.mu.Unlock()
		c.update(func(*Status) {})
		err := c.crawlItem(it)
		if errors.Is(err, errPaused) {
			c.update(func(s *Status) { s.State, s.Message = "paused", "已暂停" })
			return
		}
		if errors.Is(err, errRefused) {
			// Stop the whole queue rather than hammer the site; it resumes later.
			c.update(func(s *Status) { s.State, s.Message = "paused", err.Error() })
			return
		}
		c.mu.Lock()
		if err != nil {
			c.st.Errors = append(c.st.Errors, fmt.Sprintf("%d: %v", it.ID, err))
			log.Printf("item %d failed: %v", it.ID, err)
		}
		c.queue = c.queue[1:]
		c.st.DoneItems++
		c.saveQueue()
		c.mu.Unlock()
		c.update(func(*Status) {})
	}
}

func newClient() *http.Client {
	jar, _ := cookiejar.New(nil)
	return &http.Client{Timeout: 90 * time.Second, Jar: jar,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		}}
}

// setHeaders mirrors what the Python requests.Session sent, which the site accepts.
func setHeaders(req *http.Request) {
	req.Header.Set("User-Agent", UA)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Connection", "keep-alive")
}
