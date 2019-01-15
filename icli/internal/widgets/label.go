package widgets

import (
	"context"
	"time"

	"github.com/3Hren/sonmui/icli/internal/mp"
	"github.com/marcusolsson/tui-go"
)

const (
	defaultProgressInterval = 100 * time.Millisecond
)

var (
	defaultProgressSymbols = [...]string{`.`, `..`, `...`}
)

type runProgressEvent struct{}

type completeProgressEvent struct {
	Text string
}

type AsyncLabel struct {
	*tui.Label

	eventTxRx chan<- interface{}
}

func NewAsyncLabel(ctx context.Context, text string, router *mp.Router) *AsyncLabel {
	eventTxRx := make(chan interface{}, 16)

	m := &AsyncLabel{
		Label: tui.NewLabel(text),

		eventTxRx: eventTxRx,
	}

	go m.run(ctx, eventTxRx, router)

	return m
}

func (m *AsyncLabel) run(ctx context.Context, eventTxRx <-chan interface{}, router *mp.Router) {
	var timer *time.Ticker
	var timerRx <-chan time.Time
	counter := 0

	for {
		select {
		case <-ctx.Done():
			return
		case ev := <-eventTxRx:
			switch event := ev.(type) {
			case *runProgressEvent:
				if timer == nil {
					timer = time.NewTicker(defaultProgressInterval)
					timerRx = timer.C
					counter = 0
				}
			case *completeProgressEvent:
				if timer != nil {
					timer.Stop()
					timer = nil
					timerRx = nil
				}

				router.Execute(func() {
					m.SetStyleName("ok")
					m.SetText(event.Text)
				})
			}
		case <-timerRx:
			counter++
			router.Execute(func() {
				m.SetStyleName("normal")
				m.SetText(defaultProgressSymbols[counter%len(defaultProgressSymbols)])
			})
		}
	}
}

func (m *AsyncLabel) RunProgress(ctx context.Context) {
	m.eventTxRx <- &runProgressEvent{}
}

func (m *AsyncLabel) StopProgress(text string) {
	m.eventTxRx <- &completeProgressEvent{Text: text}
}

func (m *AsyncLabel) SetTextAsync(ctx context.Context, fn func(ctx context.Context) string) {
	m.RunProgress(ctx)
	go func() {
		m.StopProgress(fn(ctx))
	}()
}
