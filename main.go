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
		next := b.findNext()
		b.Remove(b.selected)
		b.selected = next
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

func (b *Collection[T]) findNext() *T {
	index, found := b.findIndex(b.selected)

	if !found {
		return nil
	}

	newIndex := index + 1
	if newIndex < 0 || newIndex >= len(b.items) {
		return b.items[len(b.items)-1]
	}

	return b.items[newIndex]

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
	done  bool
	name  string
	tag   string
	notes string
}

func (t Task) String() string {
	var sb strings.Builder
	if t.done {
		sb.WriteString("[x] ")
	} else {
		sb.WriteString("[ ] ")
	}
	if t.tag != "" {
		sb.WriteString(fmt.Sprintf("%s ", t.tag))
	}
	sb.WriteString(t.name)
	if t.notes != "" {
		sb.WriteString(fmt.Sprintf("\n%v", t.notes))
	}
	return sb.String()
}

type Project struct {
	name  string
	tasks Collection[Task]
	notes string
}

func (p Project) String() string {
	result := fmt.Sprintf("## %s\n", p.name)
	if p.notes != "" {
	  result += (fmt.Sprintf("%v\n\n", p.notes))
	}

	for _, task := range p.tasks.items {
		result += fmt.Sprintf("- %s\n", task)
	}
	return result
}

func (b *Project) Add(name string) *Task {
	item := &Task{
		done: false,
		name: strings.TrimSuffix(name, "\n"),
		notes: "",
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
	SendHeartbeat(filename, "")
	content := ps.String()
	return os.WriteFile(filename, []byte(content), 0644) // Write to file with appropriate permissions
}

func newProjects() Projects {
	return Projects{
		Collection: Collection[Project]{items: make([]*Project, 0)},
	}
}

func ReadFromFile(filename string) (Projects, error) {
	SendHeartbeat(filename, "")
	file, err := os.Open(filename)
	if err != nil {
		return newProjects(), err
	}
	defer file.Close()

	var projects Projects
	var currentProject *Project
	var currentTask *Task
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if strings.HasPrefix(line, "## ") { // Detect project name
			projectName := strings.TrimPrefix(line, "## ")
			currentProject = &Project{name: projectName, notes: ""}
			projects.Add(currentProject)
			currentTask = nil
		} else if strings.HasPrefix(line, "- [") { // Detect task
			if currentProject != nil {
				taskDone := strings.HasPrefix(line, "- [x]") // Task completion check
				//				taskName := strings.TrimSpace(line[5:])      // Remove `- [ ] ` or `- [x] `

				emoji, taskName := extractEmoji(strings.TrimSpace(line[5:]))
				currentTask = &Task{done: taskDone, name: taskName, tag: emoji, notes: ""}
				currentProject.tasks.Add(currentTask)
			}
		} else if currentTask != nil {
			if currentTask.notes != "" {
				currentTask.notes += "\n"
			}
			currentTask.notes += strings.TrimSpace(line)
		} else if currentProject != nil {
			if currentProject.notes != "" {
				currentProject.notes += "\n"
			}
			currentProject.notes += strings.TrimSpace(line)

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
	STYLE_HasNotes     = "🗒️"
	STYLE_Boldline     = "━"
	STYLE_Thinline     = "―"
)

var (
	filename  = "todo.md"
	tasks     Projects
	state     AppState = State_Task
	viewname           = "todo"
	dirty              = false
	autosave           = true
	hidedone           = true
	delete             = false
	showNotes          = false
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

	g.SetKeybinding("", 'n', gocui.ModNone, func(g *gocui.Gui, cv *gocui.View) error {
		showNotes = !showNotes
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

	g.SetKeybinding(viewname, 'K', gocui.ModNone, swapup)
	g.SetKeybinding(viewname, 'J', gocui.ModNone, swapdown)
	g.SetKeybinding(viewname, 'j', gocui.ModNone, prev)
	g.SetKeybinding(viewname, 'k', gocui.ModNone, next)
	g.SetKeybinding(viewname, 'i', gocui.ModNone, addView)
	g.SetKeybinding(viewname, 'I', gocui.ModNone, editView)

	g.SetKeybinding(viewname, 'e', gocui.ModNone, func(g *gocui.Gui, cv *gocui.View) error {

		if (tasks.selected != nil) && (tasks.selected.tasks.selected != nil) {
			if tasks.selected.tasks.selected.tag == "" {
				tasks.selected.tasks.selected.tag = "🔥"
			} else {
				tasks.selected.tasks.selected.tag = ""
			}
			markDirty()
		}
		redraw(g)
		return nil
	})

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
	maxX,_:= g.Size()
	doneCount := 0
	taskCount := 0
	if v, e := g.View(viewname); e == nil {
		v.Clear()
		for _, group := range tasks.items {

			noteIcon := ""
			if !showNotes && group.notes != "" {
				noteIcon = STYLE_HasNotes
			}

			if group == tasks.selected {
				fmt.Fprintln(v, "\n", STYLE_LineSelector, group.name, "(", len(group.tasks.items), ")", noteIcon)
			} else {
				fmt.Fprintln(v, "\n", " ", group.name, "(", len(group.tasks.items), ")", noteIcon)
			}

			fmt.Fprintln(v, strings.Repeat(STYLE_Boldline, maxX-2))

			if (group.notes != "") && (showNotes) {
				fmt.Fprintln(v, "\x1b[2m"+group.notes+"\x1b[0m")
				fmt.Fprintln(v, strings.Repeat(STYLE_Thinline, maxX-2))

			}

			for _, task := range group.tasks.items {
				taskCount++
				if task.done {
					doneCount++
				}
				if !hidedone || !task.done {
					noteIcon := ""
					if !showNotes && task.notes != "" {
						noteIcon = STYLE_HasNotes
					}

					checked := STYLE_Checked
					if task.done {
						checked = STYLE_UnChecked
					}
					if (task == group.tasks.selected) && (group == tasks.selected) {

						fmt.Fprintln(v, STYLE_LineSelector, checked, task.tag, task.name, noteIcon)
					} else {
						fmt.Fprintln(v, " ", checked, task.tag, task.name, noteIcon)
					}

					if task.notes != "" && showNotes {

						fmt.Fprintln(v, "\x1b[2m"+task.notes+"\x1b[0m")
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
			hidedoneStr = "Hide Done"
		}

		deleteStr := " "
		if delete {
			deleteStr = "Del"
		}

		fmt.Fprintln(v, state, dirtyStr, hidedoneStr, deleteStr, fmt.Sprintf("%d/%d", doneCount, taskCount))
	}

}

func editView(g *gocui.Gui, cv *gocui.View) error {
	delete = false

	var title string
	var val string

	switch state {
	case State_Task:
		if tasks.selected == nil {
			return nil
		}
		title = "Edit Task for " + tasks.selected.name
		val = tasks.selected.tasks.selected.name
	case State_Project:
		title = "Edit Project Name"
		val = tasks.selected.name
	}

	return showInput(g, "edit", title, val)
}

func addView(g *gocui.Gui, cv *gocui.View) error {
	delete = false

	var title string
	switch state {
	case State_Task:
		if tasks.selected == nil {
			return nil
		}
		title = "New Task for " + tasks.selected.name
	case State_Project:
		title = "New Project"
	}

	return showInput(g, "add", title, "")
}

func showInput(g *gocui.Gui, cmdname string, title string, val string) error {
	maxX, maxY := g.Size()
	iv, err := g.SetView(cmdname, 3, maxY/2, maxX-3, maxY/2+2, 0)

	if err != nil {
		if !gocui.IsUnknownView(err) {
			return err
		}

		iv.Title = title
		iv.TitleColor = gocui.ColorYellow
		iv.FrameColor = gocui.ColorRed
		iv.FrameRunes = []rune{'═', '║', '╔', '╗', '╚', '╝', '╠', '╣', '╦', '╩', '╬'}
		iv.Editable = true
		g.Cursor = true
		fmt.Fprint(iv, val)
		iv.SetCursorX(len(val))
		iv.TextArea.TypeString(val)

		if _, err := g.SetCurrentView(cmdname); err != nil {
			return err
		}
		g.SetKeybinding(cmdname, gocui.KeyEnter, gocui.ModNone, copyInput)
		g.SetKeybinding(cmdname, gocui.KeyEsc, gocui.ModNone, closeInput)
	}

	return nil
}

func closeInput(g *gocui.Gui, iv *gocui.View) error {
	var err error
	var ov *gocui.View
	ov, _ = g.View(viewname)

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

func copyInput(g *gocui.Gui, iv *gocui.View) error {
	iv.Rewind()

	if iv.Buffer() == "" {
		return closeInput(g, iv)
	}

	switch iv.Name() {
	case "add":
		switch state {
		case State_Task:
			if tasks.selected != nil {
				tasks.selected.Add(iv.Buffer())
			}
		case State_Project:
			tasks.Add(newProject(iv.Buffer()))
		}

	case "edit":
		switch state {
		case State_Task:
			if tasks.selected != nil {
				tasks.selected.tasks.selected.name = iv.Buffer()
			}
		case State_Project:
			tasks.selected.name = iv.Buffer()
		}
	}

	markDirty()
	return closeInput(g, iv)
}
