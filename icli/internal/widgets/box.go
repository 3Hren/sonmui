package widgets

import (
	"github.com/3Hren/sonmui/icli/internal/interactions"
	"github.com/marcusolsson/tui-go"
)

//
//
// Note that this box consumes all <Tab> events.
type FocusBox struct {
	*tui.Box

	controller *interactions.FocusController
}

func NewFocusBox(box *tui.Box, controller *interactions.FocusController) *FocusBox {
	return &FocusBox{Box: box, controller: controller}
}

func (m *FocusBox) SetFocusController(controller *interactions.FocusController) {
	m.controller = controller
}

func (m *FocusBox) OnKeyEvent(ev tui.KeyEvent) {
	if m.IsFocused() {
		switch ev.Key {
		case tui.KeyTab:
			m.controller.FocusNextWidget()
			return
		}
	}

	m.Box.OnKeyEvent(ev)
}
