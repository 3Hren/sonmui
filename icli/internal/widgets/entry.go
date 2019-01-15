package widgets

import (
	"image"

	"github.com/marcusolsson/tui-go"
)

type Entry struct {
	*tui.Entry

	sizeHint image.Point
}

func NewEntry() *Entry {
	return &Entry{Entry: tui.NewEntry()}
}

func (m *Entry) SetSizeHint(sizeHint image.Point) {
	m.sizeHint = sizeHint
}

func (m *Entry) SizeHint() image.Point {
	return m.sizeHint
}
