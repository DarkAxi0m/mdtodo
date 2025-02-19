package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/jesseduffield/gocui"
)

// ---------- Misc function-------------------
func updateSelected[T any](items []*T, selected *T, direction int) *T {
	if len(items) == 0 {
		return nil
	}

	selectedFound := false
	for i, item := range items {
		if selected == item {
			selectedFound = true
			if direction == 1 {
				if i+1 < len(items) {
					return items[i+1]
				} else {
					return items[0]
				}
			} else if direction == -1 {
				if i-1 >= 0 {
					return items[i-1]
				} else {
					return items[len(items)-1]
				}
			}
			break
		}
	}
	if !selectedFound {
		return items[0]
	}
	return selected
}

//---------- Collection-------------------

type Collection[T any] struct {
	items    []*T
	selected *T
}

func (b *Collection[T]) Add(item *T) *T {
	b.items = append(b.items, item)
	b.selected = item
	return item
}

func (b *Collection[T]) RemoveSelected() {
	if b.selected != nil {
		b.Remove(b.selected)
	}
}

func (b *Collection[T]) Remove(item *T) {
	for i, t := range b.items {
		if t == item {
			b.Select(-1)
			b.items = append(b.items[:i], b.items[i+1:]...)
			break
		}
	}
}

func (b *Collection[T]) Select(dir int) T {
	b.selected = updateSelected(b.items, b.selected, dir)
	return *b.selected
}

// ---------- Task ann Projects-------------------
type Task struct {
	done bool
	name string
}

func (t Task) String() string {
	checkmark := "[ ]" // Default to not done
	if t.done {
		checkmark = "[x]"
	}
	return fmt.Sprintf("%s %s", checkmark, t.name)
}

type Project struct {
	name  string
	tasks Collection[Task]
}

func (p Project) String() string {
	result := fmt.Sprintf("## %s\n", p.name)
	for _, task := range p.tasks.items {
		result += fmt.Sprintf("- %s\n", task)
	}
	return result
}

func (b *Project) Add(name string) *Task {
	item := &Task{
		done: false,
		name: strings.TrimSuffix(name, "\n"),
	}

	b.tasks.Add(item)

	return item
}

func newProject(name string) *Project {
	tasks := &Project{
		tasks: Collection[Task]{items: make([]*Task, 0)},
		name:  strings.TrimSuffix(name, "\n"),
	}
	return tasks
}

type Projects struct {
	Collection[Project]
}

func (ps Projects) String() string {
	result := "# Todo\n\n"
	for _, project := range ps.items {
		result += fmt.Sprintf("%s\n", project)
	}
	return result
}

func (ps Projects) SaveToFile(filename string) error {
	content := ps.String()
	return os.WriteFile(filename, []byte(content), 0644) // Write to file with appropriate permissions
}

func newProjects() Projects {
	return Projects{
		Collection: Collection[Project]{items: make([]*Project, 0)},
	}
}

func ReadFromFile(filename string) (Projects, error) {
	file, err := os.Open(filename)
	if err != nil {
		return newProjects(), err
	}
	defer file.Close()

	var projects Projects
	var currentProject *Project

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "## ") { // Detect project name
			projectName := strings.TrimPrefix(line, "## ")
			currentProject = &Project{name: projectName}
			projects.Add(currentProject)
		} else if strings.HasPrefix(line, "- [") { // Detect task
			if currentProject != nil {
				taskDone := strings.HasPrefix(line, "- [x]") // Task completion check
				taskName := strings.TrimSpace(line[5:])      // Remove `- [ ] ` or `- [x] `
				task := &Task{done: taskDone, name: taskName}
				currentProject.tasks.Add(task)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return newProjects(), err
	}

	return projects, nil
}

//---------- AppState-------------------

type AppState int

const (
	Normal AppState = iota
	Insert
	DeleteTask
)

func (s AppState) String() string {
	return [...]string{"N", "I", "D"}[s]
}

//---------- Main-------------------

var (
	filename = "todo.md"
	tasks    Projects
	state    AppState
	name     = "todo"
	dirty    = false
)

func main() {
	tasks, _ = ReadFromFile(filename)

	g, err := gocui.NewGui(gocui.NewGuiOpts{
		OutputMode: gocui.OutputTrue,
		//RuneReplacements: map[rune]string{},
	})
	if err != nil {
		log.Panicln(err)
	}
	defer g.Close()

	g.SetManagerFunc(layout)

	g.SetKeybinding("", 'w', gocui.ModNone, func(g *gocui.Gui, cv *gocui.View) error {
		tasks.SaveToFile(filename)
		dirty = false
		redraw(g)
		return nil
	})

	g.SetKeybinding("", 'r', gocui.ModNone, func(g *gocui.Gui, cv *gocui.View) error {
		tasks, _ = ReadFromFile(filename)
		redraw(g)
		return nil
	})

	if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		log.Panicln(err)
	}
	if err := g.SetKeybinding("", 'q', gocui.ModNone, quit); err != nil {
		log.Panicln(err)
	}

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Panicln(err)
	}
}

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	if v, err := g.SetView(name, 0, 0, maxX-1, maxY-4, 0); err != nil {
		if !gocui.IsUnknownView(err) {
			return err
		}
		v.Title = filename

		if _, err := g.SetCurrentView(name); err != nil {
			return err
		}

		todoBinding(g)

		redraw(g)

	}

	if v, err := g.SetView("footer", 0, maxY-3, maxX-1, maxY-1, 0); err != nil {
		v.Frame = false
	}

	return nil
}

//---------Key binds-----------------------------

func nextProject(g *gocui.Gui, v *gocui.View) error {
	state = Normal
	tasks.Select(-1)
	redraw(g)
	return nil
}
func prevProject(g *gocui.Gui, v *gocui.View) error {
	state = Normal
	tasks.Select(+1)

	redraw(g)
	return nil
}

func nextTask(g *gocui.Gui, v *gocui.View) error {
	if tasks.selected != nil {
		tasks.selected.tasks.Select(-1)
	}
	if state == DeleteTask {
		tasks.selected.tasks.RemoveSelected()
		state = Normal
		dirty = true
	}
	redraw(g)
	return nil
}
func prevTask(g *gocui.Gui, v *gocui.View) error {
	if tasks.selected != nil {
		tasks.selected.tasks.Select(+1)
	}
	if state == DeleteTask {
		tasks.selected.tasks.RemoveSelected()
		state = Normal
		dirty = true
	}
	redraw(g)
	return nil
}
func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

//--------------------------------------

func todoBinding(g *gocui.Gui) error {

	var v *gocui.View
	v, _ = g.View(name)

	if v == nil {
		return nil
	}

	g.SetKeybinding(name, gocui.KeyEsc, gocui.ModNone, func(g *gocui.Gui, cv *gocui.View) error {
		state = Normal
		redraw(g)
		return nil
	})

	g.SetKeybinding(name, 'J', gocui.ModNone, nextProject)
	g.SetKeybinding(name, 'K', gocui.ModNone, prevProject)
	g.SetKeybinding(name, 'j', gocui.ModNone, nextTask)
	g.SetKeybinding(name, 'k', gocui.ModNone, prevTask)
	g.SetKeybinding(name, gocui.KeyEnter, gocui.ModNone, inputView)
	g.SetKeybinding(name, 'a', gocui.ModNone, func(g *gocui.Gui, cv *gocui.View) error {
		state = Normal
		redraw(g)
		return inputView(g, cv)
	})
	g.SetKeybinding(name, 'A', gocui.ModNone, func(g *gocui.Gui, cv *gocui.View) error {
		tasks.selected = nil
		state = Normal
		redraw(g)
		return inputView(g, cv)
	})

	g.SetKeybinding(name, gocui.KeySpace, gocui.ModNone, func(g *gocui.Gui, cv *gocui.View) error {
		if (tasks.selected != nil) && (tasks.selected.tasks.selected != nil) {
			tasks.selected.tasks.selected.done = !tasks.selected.tasks.selected.done
		}
		redraw(g)
		return nil
	})
	g.SetKeybinding(name, 'd', gocui.ModNone, func(g *gocui.Gui, cv *gocui.View) error {
		if state == DeleteTask {
			tasks.selected.tasks.RemoveSelected()
			state = Normal
			dirty = true
		} else {
			state = DeleteTask
		}
		redraw(g)
		return nil
	})

	return nil
}
func redraw(g *gocui.Gui) {
	if v, e := g.View(name); e == nil {
		v.Clear()
		for _, group := range tasks.items {
			if group == tasks.selected {
				fmt.Fprintln(v, "\n»", group.name, "(", len(group.tasks.items), ")")
			} else {
				fmt.Fprintln(v, "\n ", group.name, "(", len(group.tasks.items), ")")
			}
			for _, task := range group.tasks.items {
				checked := "☐"
				if task.done {
					checked = "\U0001f5f9"
				}
				if (task == group.tasks.selected) && (group == tasks.selected) {

					fmt.Fprintln(v, " »", checked, task.name)
				} else {
					fmt.Fprintln(v, "  ", checked, task.name)
				}

			}
		}
	}
	if v, e := g.View("footer"); e == nil {
		v.Clear()
		dirtyStr := " "
		if dirty {
			dirtyStr = "D"
		}
		fmt.Fprintln(v, state, dirtyStr)
	}

}

func inputView(g *gocui.Gui, cv *gocui.View) error {
	maxX, maxY := g.Size()
	var title string
	var cmdname string

	if tasks.selected == nil {
		title = "Name of Project"
		cmdname = "addProject"
	} else {
		title = "Task for " + tasks.selected.name
		cmdname = "addTask"
	}

	state = Insert
	redraw(g)

	if iv, err := g.SetView(cmdname, 3, maxY/2, maxX-3, maxY/2+2, 0); err != nil {
		if !gocui.IsUnknownView(err) {
			return err
		}
		iv.Title = title
		iv.Editable = true
		g.Cursor = true
		if _, err := g.SetCurrentView(cmdname); err != nil {
			return err
		}
		g.SetKeybinding(cmdname, gocui.KeyEnter, gocui.ModNone, copyInput)

	}
	return nil
}

func copyInput(g *gocui.Gui, iv *gocui.View) error {
	var err error
	iv.Rewind()
	var ov *gocui.View
	ov, _ = g.View(name)

	if iv.Buffer() == "" {
		inputView(g, ov)
		return nil
	}
	switch iv.Name() {
	case "addProject":
		tasks.Add(newProject(iv.Buffer()))

	case "addTask":
		tasks.selected.Add(iv.Buffer())

	}
	dirty = true
	iv.Clear()
	g.Cursor = false
	g.DeleteViewKeybindings(iv.Name())
	if err = g.DeleteView(iv.Name()); err != nil {
		return err
	}
	// Set the view back.
	if _, err = g.SetCurrentView(ov.Name()); err != nil {
		return err
	}
	state = Normal

	redraw(g)
	return err
}
