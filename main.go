package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/awesome-gocui/gocui"
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

type Project struct {
	name  string
	tasks Collection[Task]
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

func newProjects() Collection[Project] {
	return Collection[Project]{items: make([]*Project, 0)}
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
	filename string
	tasks    Collection[Project]
	state    AppState
	name     = "todo"
)

/*
func LoadMd(v *gocui.View) {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Println("Error opening file:", err)
		return
	}
	defer file.Close()

	fmt.Fprintln(v, filename)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fmt.Fprintln(v, scanner.Text())
		//	fmt.Println() // Print each line
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading file:", err)
	}
}*/

func main() {
	filename = "todo.md"

	tasks = newProjects()
	main := tasks.Add(newProject("Main"))

	main.Add("ONE")
	main.Add("two")
	main.Add("three")

	main2 := tasks.Add(newProject("Main222"))

	main2.Add("322 3234 4")
	main2.Add("tw 234 234o")
	main2.Add("thr243  w gg2 3gee")

	g, err := gocui.NewGui(gocui.OutputNormal, true)
	if err != nil {
		log.Panicln(err)
	}
	defer g.Close()
	g.SelFgColor = gocui.ColorGreen
	g.SetManagerFunc(layout)

	g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit)

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Panicln(err)
	}
}

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	if v, err := g.SetView(name, 0, 0, maxX-1, maxY-4, 0); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Title = filename
		todoBinding(g)

		redraw(g)

		g.SetCurrentView(name)
	}

	if v, err := g.SetView("footer", 0, maxY-3, maxX-1, maxY-1, 0); err != nil {
		v.Frame = false
	}

	return nil
}

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
	}
	redraw(g)
	return nil
}

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

	g.SetKeybinding(name, ";", gocui.ModNone, func(g *gocui.Gui, cv *gocui.View) error {
		tasks.selected.Add("ASD")
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
		fmt.Fprintln(v, state)
	}

}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func inputView(g *gocui.Gui, cv *gocui.View) error {
	maxX, maxY := g.Size()
	var title string
	var name string

	if tasks.selected == nil {
		title = "Name of Project"
		name = "addProject"
	} else {
		title = "Task for " + tasks.selected.name
		name = "addTask"

	}

	state = Insert
	redraw(g)

	if iv, err := g.SetView(name, maxX/2-12, maxY/2, maxX/2+12, maxY/2+2, 0); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		iv.Title = title
		iv.Editable = true
		g.Cursor = true
		if _, err := g.SetCurrentView(name); err != nil {
			return err
		}
		g.SetKeybinding(name, gocui.KeyEnter, gocui.ModNone, copyInput)

	}
	return nil
}

func copyInput(g *gocui.Gui, iv *gocui.View) error {
	var err error
	iv.Rewind()
	var ov *gocui.View
	ov, _ = g.View("todo")
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
	iv.Clear()
	g.Cursor = false
	g.DeleteKeybindings(iv.Name())
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
