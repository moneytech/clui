package clui

import (
	term "github.com/nsf/termbox-go"
	"strings"
)

/*
ListBox is control to display a list of items and allow to user to select any of them.
Content is scrollable with arrow keys or by clicking up and bottom buttons
on the scroll(now content is scrollable with mouse dragging only on Windows).

ListBox calls onSelectItem item function after a user changes currently
selected item with mouse or using keyboard (extra case: the event is emitted
when a user presses Enter - the case is used in ComboBox to select an item
from drop down list). Event structure has 2 fields filled: Y - selected
item number in list(-1 if nothing is selected), Msg - text of the selected item.
*/
type ListBox struct {
	ControlBase
	// own listbox members
	items         []string
	currSelection int
	topLine       int
	maxItems      int
	buttonPos     int

	onSelectItem func(Event)
}

/*
NewListBox creates a new frame.
view - is a View that manages the control
parent - is container that keeps the control. The same View can be a view and a parent at the same time.
width and heigth - are minimal size of the control.
scale - the way of scaling the control when the parent is resized. Use DoNotScale constant if the
control should keep its original size.
*/
func NewListBox(view View, parent Control, width, height int, scale int) *ListBox {
	l := new(ListBox)

	if height == AutoSize {
		height = 3
	}
	if width == AutoSize {
		width = 5
	}

	l.SetSize(width, height)
	l.SetConstraints(width, height)
	l.currSelection = -1
	l.items = make([]string, 0)
	l.topLine = 0
	l.parent = parent
	l.view = view
	l.maxItems = 0
	l.buttonPos = -1

	l.SetTabStop(true)

	l.onSelectItem = nil

	if parent != nil {
		parent.AddChild(l, scale)
	}

	return l
}

func (l *ListBox) redrawScroll(canvas Canvas, tm Theme) {
	fg, bg := RealColor(tm, l.fg, ColorScrollText), RealColor(tm, l.bg, ColorScrollBack)
	fgThumb, bgThumb := RealColor(tm, l.fg, ColorThumbText), RealColor(tm, l.bg, ColorThumbBack)

	pos := -1
	if l.currSelection != -1 {
		if len(l.items) > 1 {
			ydiff := int(float32(l.currSelection) / float32(len(l.items)-1.0) * float32(l.height-3))
			l.buttonPos = ydiff + 1
			if l.height == 3 {
				pos = 1
			} else {
				pos = l.buttonPos
			}
		} else {
			pos = 1
		}
	}
	canvas.DrawScroll(l.x+l.width-1, l.y, 1, l.height, pos, fg, bg, fgThumb, bgThumb, tm.SysObject(ObjScrollBar))
}

func (l *ListBox) redrawItems(canvas Canvas, tm Theme) {
	maxCurr := len(l.items) - 1
	curr := l.topLine
	dy := 0
	maxDy := l.height - 1
	maxWidth := l.width - 1

	fg, bg := RealColor(tm, l.fg, ColorEditText), RealColor(tm, l.bg, ColorEditBack)
	if l.Active() {
		fg, bg = RealColor(tm, l.fg, ColorEditActiveText), RealColor(tm, l.bg, ColorEditActiveBack)
	}
	fgSel, bgSel := RealColor(tm, l.fgActive, ColorSelectionText), RealColor(tm, l.bgActive, ColorSelectionBack)

	for curr <= maxCurr && dy <= maxDy {
		f, b := fg, bg
		if curr == l.currSelection {
			f, b = fgSel, bgSel
		}

		canvas.FillRect(l.x, l.y+dy, l.width-1, 1, term.Cell{Bg: b, Ch: ' ', Fg: f})
		_, text := AlignText(l.items[curr], maxWidth, AlignLeft)
		canvas.PutText(l.x, l.y+dy, text, f, b)

		curr++
		dy++
	}
}

// Repaint draws the control on its View surface
func (l *ListBox) Repaint() {
	canvas := l.view.Canvas()
	tm := l.view.Screen().Theme()

	x, y := l.Pos()
	w, h := l.Size()

	bg := RealColor(tm, l.bg, ColorEditBack)
	if l.Active() {
		bg = RealColor(tm, l.bg, ColorEditActiveBack)
	}
	canvas.FillRect(x, y, w, h, term.Cell{Bg: bg, Ch: ' '})
	l.redrawItems(canvas, tm)
	l.redrawScroll(canvas, tm)
}

func (l *ListBox) home() {
	if len(l.items) > 0 {
		l.currSelection = 0
	}
	l.topLine = 0
}

func (l *ListBox) end() {
	length := len(l.items)

	if length == 0 {
		return
	}

	l.currSelection = length - 1
	if length > l.height {
		l.topLine = length - l.height
	}
}

func (l *ListBox) moveUp() {
	if l.topLine == 0 && l.currSelection == 0 {
		return
	}

	if l.currSelection == -1 {
		if len(l.items) != 0 {
			l.currSelection = 0
		}
		return
	}

	l.currSelection--
	l.EnsureVisible()
}

func (l *ListBox) moveDown() {
	length := len(l.items)

	if length == 0 || l.currSelection == length-1 {
		return
	}

	l.currSelection++
	l.EnsureVisible()
}

// EnsureVisible makes the currently selected item visible and scrolls the item list if it is required
func (l *ListBox) EnsureVisible() {
	length := len(l.items)

	if length <= l.height || l.currSelection == -1 {
		return
	}

	diff := l.currSelection - l.topLine
	if diff >= 0 && diff < l.height {
		return
	}

	if diff < 0 {
		l.topLine = l.currSelection
	} else {
		top := l.currSelection - l.height + 1
		if length-top > l.height {
			l.topLine = top
		} else {
			l.topLine = length - l.height
		}
	}
}

// Clear deletes all ListBox items
func (l *ListBox) Clear() {
	l.items = make([]string, 0)
	l.currSelection = -1
	l.topLine = 0
}

func (l *ListBox) processMouseClick(ev Event) bool {
	if ev.Key != term.MouseLeft {
		return false
	}

	dx := ev.X - l.x
	dy := ev.Y - l.y

	if dx == l.width-1 {
		if dy < 0 || dy >= l.height || len(l.items) < 2 {
			return true
		}

		if dy == 0 {
			l.moveUp()
			return true
		}
		if dy == l.height-1 {
			l.moveDown()
			return true
		}

		l.buttonPos = dy
		l.recalcPositionByScroll()
		return true
	}

	if dx < 0 || dx >= l.width || dy < 0 || dy >= l.height {
		return true
	}

	if dy >= len(l.items) {
		return true
	}

	l.SelectItem(l.topLine + dy)
	if l.onSelectItem != nil {
		ev := Event{Y: l.topLine + dy, Msg: l.SelectedItemText()}
		go l.onSelectItem(ev)
	}

	return true
}

func (l *ListBox) recalcPositionByScroll() {
	if len(l.items) < 2 {
		return
	}

	newPos := int(float32(len(l.items)-1)*float32(l.buttonPos-1)/float32(l.height-3) + 0.9)

	if newPos < 0 {
		newPos = 0
	} else if newPos >= len(l.items) {
		newPos = len(l.items) - 1
	}

	l.currSelection = newPos
	l.EnsureVisible()
}

/*
ProcessEvent processes all events come from the control parent. If a control
processes an event it should return true. If the method returns false it means
that the control do not want or cannot process the event and the caller sends
the event to the control parent
*/
func (l *ListBox) ProcessEvent(event Event) bool {
	if !l.Active() || !l.Enabled() {
		return false
	}

	switch event.Type {
	case EventKey:
		switch event.Key {
		case term.KeyHome:
			l.home()
			return true
		case term.KeyEnd:
			l.end()
			return true
		case term.KeyArrowUp:
			l.moveUp()
			return true
		case term.KeyArrowDown:
			l.moveDown()
			return true
		case term.KeyCtrlM:
			if l.currSelection != -1 && l.onSelectItem != nil {
				ev := Event{Y: l.currSelection, Msg: l.SelectedItemText()}
				go l.onSelectItem(ev)
			}
		default:
			return false
		}
	case EventMouse:
		return l.processMouseClick(event)
	}

	return false
}

// own methods

// AddItem adds a new item to item list. If the maximun item
// is greater than 0 and the number of item greater maximum
// then the first item is deleted.
// Returns true if the operation is successful
func (l *ListBox) AddItem(item string) bool {
	if l.maxItems > 0 && len(l.items) > l.maxItems {
		l.RemoveItem(0)
	}

	l.items = append(l.items, item)
	return true
}

// SelectItem slects item which number in the list equals
// id. If the item exists the ListBox scrolls the list to
// make the item visible.
// Returns true if the item is selected successfully
func (l *ListBox) SelectItem(id int) bool {
	if len(l.items) <= id || id < 0 {
		return false
	}

	l.currSelection = id
	l.EnsureVisible()
	return true
}

// FindItem looks for an item in list which text equals
// to text, by default the search is casesensitive.
// Returns item number in item list or -1 if nothing is found.
func (l *ListBox) FindItem(text string, caseSensitive bool) int {
	for idx, itm := range l.items {
		if itm == text || (caseSensitive && strings.EqualFold(itm, text)) {
			return idx
		}
	}

	return -1
}

// SelectedItem returns currently selected item id
func (l *ListBox) SelectedItem() int {
	return l.currSelection
}

// SelectedItemText returns text of currently selected item or empty sting if nothing is
// selected or ListBox is empty.
func (l *ListBox) SelectedItemText() string {
	if l.currSelection == -1 {
		return ""
	}

	return l.items[l.currSelection]
}

// RemoveItem deletes an item which number is id in item list
// Returns true if item is deleted
func (l *ListBox) RemoveItem(id int) bool {
	if id < 0 || id >= len(l.items) {
		return false
	}

	l.items = append(l.items[:id], l.items[id+1:]...)
	return true
}

// OnSelectItem sets a callback that is called every time
// the selected item is changed
func (l *ListBox) OnSelectItem(fn func(Event)) {
	l.onSelectItem = fn
}

// MaxItems returns the maximum number of items that the
// ListBox can keep. 0 means unlimited. It makes a ListBox
// work like a FIFO queue: the oldest(the first) items are
// deleted if one adds an item to a full ListBox
func (l *ListBox) MaxItems() int {
	return l.maxItems
}

// SetMaxItems sets the maximum items that ListBox keeps
func (l *ListBox) SetMaxItems(max int) {
	l.maxItems = max
}

// ItemCount returns the number of items in the ListBox
func (l *ListBox) ItemCount() int {
	return len(l.items)
}
