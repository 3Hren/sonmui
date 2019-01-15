package main

import (
	"github.com/marcusolsson/tui-go"
)

const (
	logo = `
    ▄████████  ▄██████▄  ███▄▄▄▄     ▄▄▄▄███▄▄▄▄         ▄█   ▄████████  ▄█        ▄█
    ███    ███ ███    ███ ███▀▀▀██▄ ▄██▀▀▀███▀▀▀██▄      ███  ███    ███ ███       ███
    ███    █▀  ███    ███ ███   ███ ███   ███   ███      ███▌ ███    █▀  ███       ███▌
    ███        ███    ███ ███   ███ ███   ███   ███      ███▌ ███        ███       ███▌
  ▀███████████ ███    ███ ███   ███ ███   ███   ███      ███▌ ███        ███       ███▌
           ███ ███    ███ ███   ███ ███   ███   ███      ███  ███    █▄  ███       ███
     ▄█    ███ ███    ███ ███   ███ ███   ███   ███      ███  ███    ███ ███▌    ▄ ███
   ▄████████▀   ▀██████▀   ▀█   █▀   ▀█   ███   █▀       █▀   ████████▀  █████▄▄██ █▀
	                                                                      ▀              `

	welcomeText = `Welcome to SONM!
Login or create a new account.`
)

func DefaultTheme() *tui.Theme {
	styles := map[string]tui.Style{
		"list.item.selected":  {Reverse: tui.DecorationOn},
		"table.cell.selected": {Reverse: tui.DecorationOn},
		"button.focused":      {Reverse: tui.DecorationOn},
		"yellow":              {Fg: tui.ColorYellow},
		"label.logo":          {Fg: tui.ColorBlue},
		"label.title":         {Bold: tui.DecorationOn, Fg: tui.ColorBlue},
		"label.bold":          {Bold: tui.DecorationOn},
		"label.highlight":     {Bold: tui.DecorationOn, Underline: tui.DecorationOn},
		"label.normal":        {Bold: tui.DecorationOn, Underline: tui.DecorationOff},
		"label.ok":            {},
		"label.succ":          {Fg: tui.ColorGreen},
		"label.success":       {Fg: tui.ColorGreen},
		"label.warn":          {Fg: tui.ColorYellow},
		"label.error":         {Fg: tui.ColorRed},
	}

	theme := tui.NewTheme()
	for name, style := range styles {
		theme.SetStyle(name, style)
	}

	return theme
}
