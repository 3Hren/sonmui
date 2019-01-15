package interactions

import (
	"github.com/marcusolsson/tui-go"
)

type FocusChain struct {
	widgets []tui.Widget
}

func NewFocusChain(widgets ...tui.Widget) *FocusChain {
	return &FocusChain{
		widgets: widgets,
	}
}

func (m *FocusChain) AddWidget(widget tui.Widget) {
	m.widgets = append(m.widgets, widget)
}

// FocusNext returns the widget in the ring that is after the given widget.
func (m *FocusChain) FocusNext(current tui.Widget) tui.Widget {
	for i, w := range m.widgets {
		if w != current {
			continue
		}
		if i < len(m.widgets)-1 {
			return m.widgets[i+1]
		}
		return m.widgets[0]
	}
	return nil
}

// FocusPrev returns the widget in the ring that is before the given widget.
func (m *FocusChain) FocusPrev(current tui.Widget) tui.Widget {
	for i, w := range m.widgets {
		if w != current {
			continue
		}
		if i <= 0 {
			return m.widgets[len(m.widgets)-1]
		}
		return m.widgets[i-1]
	}
	return nil
}

// FocusDefault returns the default widget for when there is no widget
// currently focused.
func (m *FocusChain) FocusDefault() tui.Widget {
	if len(m.widgets) == 0 {
		return nil
	}
	return m.widgets[0]
}

type FocusController struct {
	FocusedWidget tui.Widget

	focusChain *FocusChain
}

func NewFocusController(focusChain *FocusChain) *FocusController {
	return &FocusController{
		FocusedWidget: focusChain.FocusDefault(),

		focusChain: focusChain,
	}
}

func (m *FocusController) FocusDefaultWidget() {
	if m.FocusedWidget != nil {
		m.FocusedWidget.SetFocused(false)
	}

	m.FocusedWidget = m.focusChain.FocusDefault()
	m.FocusedWidget.SetFocused(true)
}

func (m *FocusController) FocusNextWidget() {
	if m.FocusedWidget != nil {
		m.FocusedWidget.SetFocused(false)
	}

	m.FocusedWidget = m.focusChain.FocusNext(m.FocusedWidget)
	m.FocusedWidget.SetFocused(true)
}

func (m *FocusController) FocusPrevWidget() {
	if m.FocusedWidget != nil {
		m.FocusedWidget.SetFocused(false)
	}

	m.FocusedWidget = m.focusChain.FocusPrev(m.FocusedWidget)
	m.FocusedWidget.SetFocused(true)
}
