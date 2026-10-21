package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"golang.org/x/image/draw"
)

// Repo is the GitHub repository whose latest release carries sample-data.zip.
const Repo = "mcxen/ehon-shelf"

var SampleURL = "https://github.com/" + Repo + "/releases/latest/download/sample-data.zip"

// Page is one image of a work, as in meta.json.
type Page struct {
	Vol    int    `json:"vol"`
	N      int    `json:"n"`
	Name   string `json:"name"`
	PDF    string `json:"pdf"`
	Thumb  string `json:"thumb"`
	File   string `json:"file"`
	Lowres bool   `json:"lowres,omitempty"`
}

// Item is a work in the remote catalog.
type Item struct {
	ID        int    `json:"id"`
	Title     string `json:"title"`
	TitleKana string `json:"title_kana"`
	Artist    string `json:"artist"`
}

// Meta is meta.json of a downloaded work (same format as the Python version).
type Meta struct {
	Item
	Source        string            `json:"source"`
	Bibliographic map[string]string `json:"bibliographic"`
	Volumes       []int             `json:"volumes"`
	Pages         []Page            `json:"pages"`
	Date          string            `json:"date"`
	PageCount     int               `json:"page_count"`
	Complete      bool              `json:"complete"`
}

// Book is a work on the local shelf.
type Book struct {
	Meta
	Dir   string
	Cover string // path of cover.jpg (or cover_thumb.jpg)
}

// Config is ~/EhonShelf/config.json.
type Config struct {
	DataDir   string         `json:"data_dir"`
	Delay     float64        `json:"delay"`
	Positions map[string]int `json:"positions,omitempty"`
	RTL       *bool          `json:"rtl,omitempty"`
	Spread    bool           `json:"spread,omitempty"`
}

var confMu sync.Mutex

func homeDir() string {
	if h := os.Getenv("EHONSHELF_HOME"); h != "" {
		return h
	}
	h, _ := os.UserHomeDir()
	return filepath.Join(h, "EhonShelf")
}

func confPath() string { return filepath.Join(homeDir(), "config.json") }

func LoadConfig() *Config {
	c := &Config{DataDir: filepath.Join(homeDir(), "data"), Delay: 1.2}
	if b, err := os.ReadFile(confPath()); err == nil {
		_ = json.Unmarshal(b, c)
	}
	if c.Delay < MinDelay {
		c.Delay = MinDelay
	}
	if c.Positions == nil {
		c.Positions = map[string]int{}
	}
	return c
}

func (c *Config) Save() error {
	confMu.Lock()
	defer confMu.Unlock()
	b, _ := json.MarshalIndent(c, "", " ")
	if err := os.MkdirAll(homeDir(), 0o755); err != nil {
		return err
	}
	return writeAtomic(confPath(), b)
}

func writeAtomic(path string, b []byte) error {
	if err := os.WriteFile(path+".part", b, 0o644); err != nil {
		return err
	}
	return os.Rename(path+".part", path)
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", " ")
	if err != nil {
		return err
	}
	return writeAtomic(path, b)
}

func readMeta(path string) (*Meta, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	m := &Meta{}
	return m, json.Unmarshal(b, m)
}

// ScanLibrary lists the works in dataDir, making missing covers.
func ScanLibrary(dataDir string) []Book {
	ents, _ := os.ReadDir(dataDir)
	var out []Book
	for _, e := range ents {
		if _, err := strconv.Atoi(e.Name()); err != nil || !e.IsDir() {
			continue
		}
		d := filepath.Join(dataDir, e.Name())
		m, err := readMeta(filepath.Join(d, "meta.json"))
		if err != nil {
			continue
		}
		b := Book{Meta: *m, Dir: d}
		if p, err := MakeCover(d); err == nil {
			b.Cover = p
		} else if _, err := os.Stat(filepath.Join(d, "cover_thumb.jpg")); err == nil {
			b.Cover = filepath.Join(d, "cover_thumb.jpg")
		}
		out = append(out, b)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// MakeCover writes cover.jpg, 400 px wide, from pages/0001.jpg when missing or stale.
func MakeCover(dir string) (string, error) {
	src, dst := filepath.Join(dir, "pages", "0001.jpg"), filepath.Join(dir, "cover.jpg")
	si, err := os.Stat(src)
	if err != nil {
		if _, e2 := os.Stat(dst); e2 == nil {
			return dst, nil
		}
		return "", err
	}
	if di, err := os.Stat(dst); err == nil && !di.ModTime().Before(si.ModTime()) {
		return dst, nil
	}
	img, err := loadScaled(src, 400, 0)
	if err != nil {
		return "", err
	}
	f, err := os.Create(dst + ".part")
	if err != nil {
		return "", err
	}
	err = jpeg.Encode(f, img, &jpeg.Options{Quality: 85})
	f.Close()
	if err != nil {
		return "", err
	}
	return dst, os.Rename(dst+".part", dst)
}

// loadScaled decodes a JPEG and shrinks it to width w (if w>0) or so its
// longest side is at most maxSide (if maxSide>0).
func loadScaled(path string, w, maxSide int) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return nil, err
	}
	b := img.Bounds()
	nw, nh := b.Dx(), b.Dy()
	if w > 0 && nw > w {
		nw, nh = w, b.Dy()*w/b.Dx()
	}
	if maxSide > 0 && max(nw, nh) > maxSide {
		if nw >= nh {
			nw, nh = maxSide, nh*maxSide/nw
		} else {
			nw, nh = nw*maxSide/nh, maxSide
		}
	}
	if nw == b.Dx() && nh == b.Dy() {
		return img, nil
	}
	dst := image.NewRGBA(image.Rect(0, 0, nw, nh))
	draw.CatmullRom.Scale(dst, dst.Bounds(), img, b, draw.Src, nil)
	return dst, nil
}

// ImportSample unpacks sample-data.zip into dataDir without overwriting.
func ImportSample(zipPath, dataDir string) error {
	z, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer z.Close()
	for _, f := range z.File {
		n := filepath.FromSlash(f.Name)
		if strings.HasSuffix(f.Name, "/") || strings.Contains(f.Name, "..") || filepath.IsAbs(n) {
			continue
		}
		tgt := filepath.Join(dataDir, n)
		if _, err := os.Stat(tgt); err == nil {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(tgt), 0o755); err != nil {
			return err
		}
		r, err := f.Open()
		if err != nil {
			return err
		}
		w, err := os.Create(tgt + ".part")
		if err == nil {
			_, err = io.Copy(w, r)
			w.Close()
		}
		r.Close()
		if err != nil {
			return err
		}
		if err := os.Rename(tgt+".part", tgt); err != nil {
			return err
		}
	}
	return nil
}

// ImportZips imports several data zips (same layout as sample-data.zip) and
// returns how many were imported.
func ImportZips(paths []string, dataDir string) (int, error) {
	for i, p := range paths {
		if err := ImportSample(p, dataDir); err != nil {
			return i, fmt.Errorf("%s: %w", filepath.Base(p), err)
		}
	}
	return len(paths), nil
}

// DownloadsDir is ~/Downloads when it exists, else the home directory.
func DownloadsDir() string {
	h, _ := os.UserHomeDir()
	if d := filepath.Join(h, "Downloads"); h != "" {
		if st, err := os.Stat(d); err == nil && st.IsDir() {
			return d
		}
	}
	return h
}

// DownloadSample fetches sample-data.zip from the latest release and imports it.
func DownloadSample(dataDir string, progress func(string)) error {
	req, _ := http.NewRequest("GET", SampleURL, nil)
	req.Header.Set("User-Agent", "EhonShelf/0.2")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}
	tmp := filepath.Join(dataDir, "sample-data.zip.part")
	defer os.Remove(tmp)
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	buf := make([]byte, 1<<20)
	var got int64
	for {
		n, rerr := resp.Body.Read(buf)
		if n > 0 {
			if _, err := f.Write(buf[:n]); err != nil {
				f.Close()
				return err
			}
			got += int64(n)
			msg := fmt.Sprintf("已下载 %d MB", got>>20)
			if resp.ContentLength > 0 {
				msg += fmt.Sprintf(" / %d MB", resp.ContentLength>>20)
			}
			progress(msg)
		}
		if rerr == io.EOF {
			break
		}
		if rerr != nil {
			f.Close()
			return rerr
		}
	}
	f.Close()
	progress("正在导入")
	return ImportSample(tmp, dataDir)
}
