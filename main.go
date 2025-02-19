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
					return selected
					//return items[0]
				}
			} else if direction == -1 {
				if i-1 >= 0 {
					return items[i-1]
				} else {
					return selected
					//return items[len(items)-1]
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

func (b *Collection[T]) SelectFirst() *T {
	if len(b.items) > 0 {
		b.selected = b.items[0]
	}
	return b.selected
}

func (b *Collection[T]) SelectLast() *T {
	if len(b.items) > 0 {
		b.selected = b.items[len(b.items)-1]
	}
	return b.selected
}

func (b *Collection[T]) Select(dir int) *T {
	b.selected = updateSelected(b.items, b.selected, dir)
	return b.selected
}

// findIndex finds the index of a given item in the items slice.
func (c *Collection[T]) findIndex(item *T) (int, bool) {
	if item == nil {
		return -1, false // No item provided
	}

	for i, v := range c.items {
		if v == item {
			return i, true
		}
	}
	return -1, false // Item not found
}

func (c *Collection[T]) MoveSelected(dir int) *T {
	index, found := c.findIndex(c.selected)
	if !found {
		return nil
	}

	newIndex := index + dir
	if newIndex < 0 || newIndex >= len(c.items) {
		return c.items[index]
	}

	// Swap the selected item with the new position
	c.items[index], c.items[newIndex] = c.items[newIndex], c.items[index]
	return c.items[index]
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

func (b *Project) Select(dir int, skipdone bool) *Task {
	var t *Task
	for {
		p := b.tasks.selected //check if it did not move
		t = b.tasks.Select(dir)
		if t == nil || !skipdone || !t.done || p == t {
			break
		}
	}
	return t
}
func (b *Project) MoveSelected(dir int, skipdone bool) *Task {
	var t *Task
	for {
		p := b.tasks.selected //check if it did not move
		t = b.tasks.MoveSelected(dir)
		if t == nil || !skipdone || !t.done || p == t {
			break
		}
	}
	return t
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
	State_Task AppState = iota
	State_Project
)

func (s AppState) String() string {
	return [...]string{"Task", "Proj"}[s]
}

//---------- Main-------------------

const (
	STYLE_Checked      = "☐"
	STYLE_UnChecked    = "\U0001f5f9"
	STYLE_LineSelector = "»"
)

var (
	filename = "todo.md"
	tasks    Projects
	state    AppState = State_Task
	viewname          = "todo"
	dirty             = false
	autosave          = true
	hidedone          = true
	delete            = false
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

	g.SetKeybinding("", 'h', gocui.ModNone, func(g *gocui.Gui, cv *gocui.View) error {
		hidedone = !hidedone

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

func markDirty() {
	dirty = true
	if autosave {
		tasks.SaveToFile(filename)
		dirty = false
	}

}

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	if v, err := g.SetView(viewname, 0, 0, maxX-1, maxY-4, 0); err != nil {
		if !gocui.IsUnknownView(err) {
			return err
		}
		v.Title = filename

		if _, err := g.SetCurrentView(viewname); err != nil {
			return err
		}
		todoBinding(g)
	}

	if v, err := g.SetView("footer", 0, maxY-3, maxX-1, maxY-1, 0); err != nil {
		v.Frame = false
	}
	redraw(g)
	return nil
}

//---------Key binds-----------------------------

func deleteSelected() error {
	switch state {
	case State_Task:
		if tasks.selected != nil {
			tasks.selected.tasks.RemoveSelected()
		}
	case State_Project:
		tasks.RemoveSelected()
	}

	delete = false
	markDirty()
	return nil
}

func next(g *gocui.Gui, v *gocui.View) error {
	switch state {
	case State_Task:
		if tasks.selected != nil {

			p := tasks.selected.tasks.selected
			if p == tasks.selected.Select(-1, hidedone) {
				tasks.Select(-1)
				tasks.selected.tasks.SelectLast()
			}
		}
	case State_Project:
		tasks.Select(-1)
	}

	if delete {
		deleteSelected()
		prev(g, v)
	}
	redraw(g)
	return nil
}

func prev(g *gocui.Gui, v *gocui.View) error {
	switch state {
	case State_Task:
		if tasks.selected != nil {
			p := tasks.selected.tasks.selected

			if p == tasks.selected.Select(+1, hidedone) {
				tasks.Select(+1)
				tasks.selected.tasks.SelectFirst()
			}
		}
	case State_Project:
		tasks.Select(+1)
	}

	if delete {
		deleteSelected()
	}
	redraw(g)
	return nil
}

func swapup(g *gocui.Gui, v *gocui.View) error {
	switch state {
	case State_Task:
		if tasks.selected != nil {
			tasks.selected.MoveSelected(-1, hidedone)
			markDirty()
		}
	case State_Project:
		tasks.MoveSelected(-1)
		markDirty()

	}

	redraw(g)
	return nil
}

func swapdown(g *gocui.Gui, v *gocui.View) error {
	switch state {
	case State_Task:
		if tasks.selected != nil {
			tasks.selected.MoveSelected(+1, hidedone)
			markDirty()
		}
	case State_Project:
		tasks.MoveSelected(+1)
		markDirty()

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
	v, _ = g.View(viewname)

	if v == nil {
		return nil
	}

	g.SetKeybinding(viewname, gocui.KeyEsc, gocui.ModNone, func(g *gocui.Gui, cv *gocui.View) error {
		state = State_Task
		delete = false
		redraw(g)
		return nil
	})

	g.SetKeybinding(viewname, 'J', gocui.ModNone, swapup)
	g.SetKeybinding(viewname, 'K', gocui.ModNone, swapdown)
	g.SetKeybinding(viewname, 'j', gocui.ModNone, next)
	g.SetKeybinding(viewname, 'k', gocui.ModNone, prev)
	g.SetKeybinding(viewname, 'i', gocui.ModNone, addView)

	g.SetKeybinding(viewname, 'p', gocui.ModNone, func(g *gocui.Gui, cv *gocui.View) error {
		state = State_Project
		redraw(g)
		return nil
	})
	g.SetKeybinding(viewname, 't', gocui.ModNone, func(g *gocui.Gui, cv *gocui.View) error {
		state = State_Task
		redraw(g)
		return nil
	})

	g.SetKeybinding(viewname, gocui.KeySpace, gocui.ModNone, func(g *gocui.Gui, cv *gocui.View) error {
		if (tasks.selected != nil) && (tasks.selected.tasks.selected != nil) {
			tasks.selected.tasks.selected.done = !tasks.selected.tasks.selected.done
			markDirty()
		}
		redraw(g)
		return nil
	})

	g.SetKeybinding(viewname, 'd', gocui.ModNone, func(g *gocui.Gui, cv *gocui.View) error {
		if delete {
			deleteSelected()
		} else {
			delete = true
		}
		redraw(g)
		return nil
	})

	return nil
}

func redraw(g *gocui.Gui) {
	if v, e := g.View(viewname); e == nil {
		v.Clear()
		for _, group := range tasks.items {
			if group == tasks.selected {
				fmt.Fprintln(v, "\n", STYLE_LineSelector, group.name, "(", len(group.tasks.items), ")")
			} else {
				fmt.Fprintln(v, "\n", " ", group.name, "(", len(group.tasks.items), ")")
			}

			for _, task := range group.tasks.items {
				if !hidedone || !task.done {
					checked := STYLE_Checked
					if task.done {
						checked = STYLE_UnChecked
					}
					if (task == group.tasks.selected) && (group == tasks.selected) {

						fmt.Fprintln(v, STYLE_LineSelector, checked, task.name)
					} else {
						fmt.Fprintln(v, " ", checked, task.name)
					}
				}

			}
		}
	}
	if v, e := g.View("footer"); e == nil {
		//this needs more thought
		v.Clear()
		dirtyStr := " "
		if dirty {
			dirtyStr = "Dirty"
		}

		hidedoneStr := " "
		if hidedone {
			hidedoneStr = "Hide"
		}

		deleteStr := " "
		if delete {
			deleteStr = "Del"
		}
		fmt.Fprintln(v, state, dirtyStr, hidedoneStr, deleteStr)
	}

}

func addView(g *gocui.Gui, cv *gocui.View) error {
	maxX, maxY := g.Size()
	var title string
	var cmdname string

	delete = false

	switch state {
	case State_Task:
		if tasks.selected != nil {
			title = "Task for " + tasks.selected.name
			cmdname = "addTask"
		}
	case State_Project:
		title = "Name of Project"
		cmdname = "addProject"
	}

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
	ov, _ = g.View(viewname)

	if iv.Buffer() == "" {
		return nil
	}

	switch state {
	case State_Task:
		if tasks.selected != nil {
			tasks.selected.Add(iv.Buffer())
		}
	case State_Project:
		tasks.Add(newProject(iv.Buffer()))
	}

	markDirty()
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

	redraw(g)
	return err
}
