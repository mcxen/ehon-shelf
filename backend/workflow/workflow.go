// Package workflow is the shell's task catalog: named commands grouped
// into verify and release, the two kinds of work the shell knows how to
// run for a project built on it.
package workflow

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"time"
)

// Group is a class of tasks, as verify or release.
type Group struct {
	ID       string `json:"id"`
	TitleKey string `json:"titleKey"`
}

// Task is one command. Argv is executed directly, not through a shell.
type Task struct {
	ID        string   `json:"id"`
	Group     string   `json:"group"`
	TitleKey  string   `json:"titleKey"`
	DetailKey string   `json:"detailKey"`
	Argv      []string `json:"argv"`
}

// Catalog is a classified list of tasks.
type Catalog struct {
	Groups []Group `json:"groups"`
	Tasks  []Task  `json:"tasks"`
}

// Result is one run of a task.
type Result struct {
	OK     bool
	Output string
	Err    string
	At     time.Time
}

// Parse reads a catalog and checks that every task names a group the
// file itself declares, and that ids and argv are present.
func Parse(b []byte) (Catalog, error) {
	var c Catalog
	if err := json.Unmarshal(b, &c); err != nil {
		return Catalog{}, err
	}
	if len(c.Groups) == 0 {
		return Catalog{}, fmt.Errorf("workflow: no groups")
	}
	seen := map[string]bool{}
	var ids []string
	for _, g := range c.Groups {
		if g.ID == "" || g.TitleKey == "" || seen[g.ID] {
			return Catalog{}, fmt.Errorf("workflow: bad group %q", g.ID)
		}
		seen[g.ID] = true
		ids = append(ids, g.ID)
	}
	if len(c.Tasks) == 0 {
		return Catalog{}, fmt.Errorf("workflow: no tasks")
	}
	taskIDs := map[string]bool{}
	for _, task := range c.Tasks {
		if task.ID == "" || taskIDs[task.ID] || task.TitleKey == "" || len(task.Argv) == 0 || task.Argv[0] == "" {
			return Catalog{}, fmt.Errorf("workflow: bad task %q", task.ID)
		}
		if !seen[task.Group] {
			return Catalog{}, fmt.Errorf("workflow: task %q is in unknown group %q (have %s)", task.ID, task.Group, ids)
		}
		taskIDs[task.ID] = true
	}
	return c, nil
}

// LoadFile parses the catalog at path.
func LoadFile(path string) (Catalog, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Catalog{}, err
	}
	return Parse(b)
}

// ByID returns the task, and whether it is in the catalog.
func (c Catalog) ByID(id string) (Task, bool) {
	for _, task := range c.Tasks {
		if task.ID == id {
			return task, true
		}
	}
	return Task{}, false
}

// GroupIDs returns the declared groups, in order.
func (c Catalog) GroupIDs() []string {
	out := make([]string, len(c.Groups))
	for i, g := range c.Groups {
		out[i] = g.ID
	}
	return out
}

// TasksIn returns the tasks of one group.
func (c Catalog) TasksIn(group string) []Task {
	var out []Task
	for _, task := range c.Tasks {
		if task.Group == group {
			out = append(out, task)
		}
	}
	return out
}

// Execute runs task in dir. An empty dir or an empty argv does not start
// a process.
func Execute(ctx context.Context, task Task, dir string) Result {
	res := Result{At: time.Now()}
	if dir == "" {
		res.Err = "no module root"
		return res
	}
	if len(task.Argv) == 0 {
		res.Err = "no command"
		return res
	}
	cmd := exec.CommandContext(ctx, task.Argv[0], task.Argv[1:]...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if len(out) > 16<<10 {
		out = out[len(out)-16<<10:]
	}
	res.Output = string(out)
	if err != nil {
		res.Err = err.Error()
		return res
	}
	res.OK = true
	return res
}

// Classify reports groups that have no task, for the catalog test.
func (c Catalog) EmptyGroups() []string {
	var out []string
	for _, g := range c.Groups {
		if len(c.TasksIn(g.ID)) == 0 {
			out = append(out, g.ID)
		}
	}
	slices.Sort(out)
	return out
}
