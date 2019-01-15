package widgets

import (
	"github.com/marcusolsson/tui-go"
)

type List struct {
	*tui.List

	items []string

	OnKeyEventX        func(ev tui.KeyEvent) bool
	selectedItemBefore int
}

func NewList() *List {
	return &List{
		List:               tui.NewList(),
		selectedItemBefore: -1,
	}
}

func (m *List) Draw(painter *tui.Painter) {
	for idx, item := range m.items {
		style := "list.item"

		if idx == m.Selected() {
			style += ".selected"
			//item = fmt.Sprintf("❯ %s ❮", item)
		} else {
			//item = fmt.Sprintf("  %s  ", item)
		}

		painter.WithStyle(style, func(painter *tui.Painter) {
			painter.FillRect(0, idx, m.Size().X, 1)
			painter.DrawText(0, idx, item)
		})
	}
}

func (m *List) AddItems(items ...string) {
	m.items = append(m.items, items...)
	m.List.AddItems(items...)
}

func (m *List) RemoveItem(i int) {
	copy(m.items[i:], m.items[i+1:])
	m.items[len(m.items)-1] = ""
	m.items = m.items[:len(m.items)-1]

	m.List.RemoveItem(i)
}

func (m *List) RemoveItems() {
	m.items = nil
	m.List.RemoveItems()
}

func (m *List) ReplaceItems(items ...string) {
	m.RemoveItems()
	m.AddItems(items...)
}

func (m *List) SetFocused(focused bool) {
	if !focused {
		m.selectedItemBefore = m.Selected()
		m.List.Select(-1)
		m.List.SetFocused(false)
		return
	}

	if focused && m.Length() > 0 {
		if m.selectedItemBefore == -1 {
			m.Select(0)
		} else {
			m.Select(m.selectedItemBefore)
		}
	}

	m.List.SetFocused(focused)
}

func (m *List) OnKeyEvent(ev tui.KeyEvent) {
	if !m.IsFocused() {
		return
	}

	if m.OnKeyEventX != nil && m.OnKeyEventX(ev) {
		return
	}

	switch ev.Key {
	case tui.KeyBacktab, tui.KeyUp:
		pos := m.List.Selected()
		m.List.OnKeyEvent(tui.KeyEvent{Key: tui.KeyUp})
		if m.List.Selected() == 0 && pos == 0 {
			m.List.Select(m.Length() - 1)
		}
	case tui.KeyTab, tui.KeyDown:
		pos := m.List.Selected()
		m.List.OnKeyEvent(tui.KeyEvent{Key: tui.KeyDown})
		if m.List.Selected() == pos && pos == m.Length()-1 {
			m.List.Select(0)
		}
	default:
		m.List.OnKeyEvent(ev)
	}
}
