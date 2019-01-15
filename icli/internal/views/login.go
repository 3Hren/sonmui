package views

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/3Hren/sonmui/icli/internal/interactions"
	"github.com/3Hren/sonmui/icli/internal/mp"
	"github.com/3Hren/sonmui/icli/internal/widgets"
	"github.com/ethereum/go-ethereum/common"
	"github.com/marcusolsson/tui-go"
	"github.com/mitchellh/go-homedir"
	"github.com/sonm-io/core/accounts"
)

func filesystemHint(text string) []string {
	if len(text) == 0 {
		return nil
	}

	text, err := homedir.Expand(text)
	if err != nil {
		return nil
	}

	matches, err := filepath.Glob(text + "*")
	if err != nil {
		return nil
	}

	dirs := make([]string, 0, len(matches))
	for _, match := range matches {
		fileInfo, err := os.Lstat(match)
		if err != nil {
			continue
		}

		if fileInfo.IsDir() {
			dirs = append(dirs, match+"/")
		}
	}

	return dirs
}

type StyledBox struct {
	Style string
	*tui.Box
}

// Draw decorates the Draw call to the widget with a style.
func (s *StyledBox) Draw(p *tui.Painter) {
	p.WithStyle(s.Style, func(p *tui.Painter) {
		s.Box.Draw(p)
	})
}

type EditHint struct {
	*tui.Box

	entry                  *tui.Entry
	suggestionsFilterEntry *tui.Entry
	suggestionsList        *widgets.List
	suggestions            []string
	onSubmit               func(entry *tui.Entry)
	OnHintRequested        func(text string) []string
}

func NewEditHint() *EditHint {
	entry := tui.NewEntry()
	box := tui.NewVBox(entry)

	suggestionsList := widgets.NewList()
	suggestionsList.OnSelectionChanged(func(list *tui.List) {
		if list.Length() == 0 || list.Selected() == -1 {
			return
		}
		entry.SetText(list.SelectedItem())
	})

	m := &EditHint{
		Box:                    box,
		entry:                  entry,
		suggestionsFilterEntry: tui.NewEntry(),
		suggestionsList:        suggestionsList,
	}

	suggestionsList.OnItemActivated(func(hints *tui.List) {
		if hints.Length() == 0 {
			return
		}

		entry.SetText(hints.SelectedItem())
		m.suggestions = nil
		m.suggestionsFilterEntry.SetText("")
		hints.RemoveItems()
		m.hideSuggestions()

		hints.SetFocused(false)
		entry.SetFocused(true)
	})

	entry.OnSubmit(func(entry *tui.Entry) {
		suggestionsList.RemoveItems()
		m.hideSuggestions()

		if m.onSubmit != nil {
			m.onSubmit(entry)
		}
	})

	return m
}

func (m *EditHint) showFilterForm() {
	if m.Length() == 3 {
		return
	}

	m.Insert(1, &StyledBox{Style: "yellow", Box: tui.NewHBox(m.suggestionsFilterEntry)})
}

func (m *EditHint) hideFilterForm() {
	for m.Length() != 3 {
		return
	}

	m.Remove(1)
}

func (m *EditHint) showSuggestions() {
	for m.Length() != 1 {
		panic("")
	}

	m.Append(m.suggestionsList)
}

func (m *EditHint) hideSuggestions() {
	for m.Length() != 1 {
		m.Remove(m.Length() - 1)
	}
}

func (m *EditHint) SetFocused(v bool) {
	m.entry.SetFocused(v)
}

func (m *EditHint) OnKeyEvent(ev tui.KeyEvent) {
	switch ev.Key {
	case tui.KeyRune:
		if m.suggestionsList.IsFocused() {
			m.showFilterForm()
			m.suggestionsFilterEntry.SetText(fmt.Sprintf("%s%c", m.suggestionsFilterEntry.Text(), ev.Rune))

			var filtered []string
			for _, hint := range m.suggestions {
				if strings.Contains(hint, m.suggestionsFilterEntry.Text()) {
					filtered = append(filtered, hint)
				}
			}
			m.suggestionsList.ReplaceItems(filtered...)

			if len(filtered) > 0 {
				m.suggestionsList.Select(0)
			}

			return
		}
	case tui.KeyBackspace, tui.KeyBackspace2:
		if m.suggestionsList.IsFocused() {
			if len(m.suggestionsFilterEntry.Text()) == 1 {
				m.hideFilterForm()
			}

			if len(m.suggestionsFilterEntry.Text()) == 0 {
				m.suggestions = nil
				m.suggestionsList.SetFocused(false)
				m.suggestionsList.RemoveItems()
				m.hideSuggestions()
				m.entry.SetFocused(true)
			} else {
				m.suggestionsFilterEntry.SetText(m.suggestionsFilterEntry.Text()[:len(m.suggestionsFilterEntry.Text())-1])

				var filtered []string
				for _, hint := range m.suggestions {
					if strings.Contains(hint, m.suggestionsFilterEntry.Text()) {
						filtered = append(filtered, hint)
					}
				}
				m.suggestionsList.ReplaceItems(filtered...)

				if len(filtered) > 0 {
					m.suggestionsList.Select(0)
				}
			}
		}
	case tui.KeyTab:
		if m.entry.IsFocused() {
			suggestions := m.OnHintRequested(m.entry.Text())

			switch len(suggestions) {
			case 0:
				return
			case 1:
				m.entry.SetText(suggestions[0])
			default:
				if m.suggestionsList.Length() != 0 {
					m.entry.SetFocused(false)
					m.suggestionsList.SetFocused(true)
					m.showSuggestions()

					m.suggestions = suggestions
					m.suggestionsList.ReplaceItems(suggestions...)
					m.suggestionsList.Select(0)
				} else {
					// On first "Tab" we just show available hints without focusing.
					m.suggestions = suggestions
					m.suggestionsList.ReplaceItems(suggestions...)
				}
			}

			return
		}
	}

	m.Box.OnKeyEvent(ev)
}

func (m *EditHint) OnSubmit(fn func(entry *tui.Entry)) {
	m.onSubmit = fn
}

type LoginView struct {
	*tui.Box

	keystoreLabel       *tui.Label
	keystoreEdit        *EditHint
	accountLabel        *tui.Label
	accountEdit         *EditHint
	passwordLabel       *tui.Label
	passwordEdit        *tui.Entry
	accountUnlockButton *tui.Button
	cancelButton        *tui.Button

	onKeyEvent func(ev tui.KeyEvent) bool
}

func NewLoginView() *LoginView {
	keystoreLabel := tui.NewLabel("Keystore Path:")
	keystoreLabel.SetStyleName("bold")

	keystoreEdit := NewEditHint()
	keystoreEdit.SetSizePolicy(tui.Expanding, tui.Preferred)
	keystoreEdit.OnHintRequested = filesystemHint

	accountLabel := tui.NewLabel("Account:")
	accountLabel.SetStyleName("bold")

	accountEdit := NewEditHint()
	accountEdit.SetSizePolicy(tui.Expanding, tui.Preferred)

	passwordLabel := tui.NewLabel("Password:")
	passwordLabel.SetStyleName("bold")

	passwordEdit := tui.NewEntry()
	passwordEdit.SetEchoMode(tui.EchoModePassword)
	passwordEdit.SetSizePolicy(tui.Expanding, tui.Preferred)

	accountUnlockButton := tui.NewButton("[Unlock]")
	cancelButton := tui.NewButton("[Cancel]")

	box := tui.NewVBox(
		tui.NewPadder(50, 0, tui.NewSpacer()),
		tui.NewHBox(tui.NewSpacer(), tui.NewLabel("Please Sign In"), tui.NewSpacer()),
		tui.NewHBox(
			keystoreLabel,
			tui.NewPadder(1, 0, keystoreEdit),
		),
		tui.NewHBox(
			accountLabel,
			tui.NewPadder(7, 0, accountEdit),
		),
		tui.NewHBox(
			passwordLabel,
			tui.NewPadder(6, 0, passwordEdit),
		),
		tui.NewHBox(
			tui.NewSpacer(),
			tui.NewPadder(1, 0, accountUnlockButton),
			tui.NewPadder(1, 0, cancelButton),
		),
	)
	box.SetBorder(true)

	window := tui.NewHBox(
		tui.NewSpacer(),
		tui.NewVBox(
			tui.NewSpacer(),
			box,
			tui.NewSpacer(),
			tui.NewSpacer(),
		),
		tui.NewSpacer(),
	)

	return &LoginView{
		Box:                 window,
		keystoreLabel:       keystoreLabel,
		keystoreEdit:        keystoreEdit,
		accountLabel:        accountLabel,
		accountEdit:         accountEdit,
		passwordLabel:       passwordLabel,
		passwordEdit:        passwordEdit,
		accountUnlockButton: accountUnlockButton,
		cancelButton:        cancelButton,
	}
}

func (m *LoginView) OnKeyEvent(ev tui.KeyEvent) {
	if m.onKeyEvent != nil && m.onKeyEvent(ev) {
		return
	}

	m.Box.OnKeyEvent(ev)
}

func (m *LoginView) SetFocused(v bool) {
	m.keystoreEdit.SetFocused(v)
}

type LabeledWidget struct {
	Label  *tui.Label
	Widget tui.Widget
}

type LoginController struct {
	view *LoginView

	focusController *interactions.FocusController

	// Signals

	OnUnlocked *mp.Signal
	OnCancel   *mp.Signal
}

func NewLoginController(view *LoginView, router *mp.Router) *LoginController {
	m := &LoginController{
		view:            view,
		focusController: interactions.NewFocusController(interactions.NewFocusChain(view.keystoreEdit, view.cancelButton)),

		OnUnlocked: router.NewSignal(),
		OnCancel:   router.NewSignal(),
	}

	view.keystoreEdit.OnSubmit(func(entry *tui.Entry) {
		keystore, err := accounts.NewMultiKeystore(accounts.NewKeystoreConfig(entry.Text()), accounts.NewStaticPassPhraser(""))
		if err != nil {
			return
		}

		if len(keystore.List()) > 0 {
			m.setInvalidAccountState()
		}

		router.Execute(func() {
			if entry.IsFocused() {
				m.focusController.FocusNextWidget()
				m.HighlightActiveWidget()
				view.accountEdit.OnHintRequested = func(text string) []string {
					accounts := make([]string, 0, len(keystore.List()))
					for _, account := range keystore.List() {
						accounts = append(accounts, account.Address.Hex())
					}

					return accounts
				}
			}
		})
	})

	view.accountEdit.OnSubmit(func(entry *tui.Entry) {
		m.setValidAccountState()

		router.Execute(func() {
			if entry.IsFocused() {
				// todo: error check
				m.focusController.FocusNextWidget()
				m.HighlightActiveWidget()
			}
		})
	})

	view.passwordEdit.OnSubmit(func(entry *tui.Entry) {
		m.focusController.FocusNextWidget()
		m.HighlightActiveWidget()
	})

	m.view.onKeyEvent = func(ev tui.KeyEvent) bool {
		switch ev.Key {
		case tui.KeyTab:
			switch m.focusController.FocusedWidget {
			case m.view.keystoreEdit, m.view.accountEdit:
			default:
				m.focusController.FocusNextWidget()
				m.HighlightActiveWidget()
				return true
			}
		}

		return false
	}

	view.accountUnlockButton.OnActivated(func(*tui.Button) {
		keystore, err := accounts.NewMultiKeystore(accounts.NewKeystoreConfig(view.keystoreEdit.entry.Text()), accounts.NewStaticPassPhraser(""))
		if err != nil {
			return
		}

		account, err := keystore.GetKeyWithPass(common.HexToAddress(view.accountEdit.entry.Text()), view.passwordEdit.Text())
		if err != nil {
			return
		}

		m.OnUnlocked.Emit(account)
	})
	view.cancelButton.OnActivated(func(*tui.Button) {
		m.OnCancel.Emit(struct{}{})
	})

	return m
}

func (m *LoginController) HighlightActiveWidget() {
	availablePairs := []LabeledWidget{
		{Label: m.view.keystoreLabel, Widget: m.view.keystoreEdit},
		{Label: m.view.accountLabel, Widget: m.view.accountEdit},
		{Label: m.view.passwordLabel, Widget: m.view.passwordEdit},
	}

	for _, widget := range availablePairs {
		style := "normal"
		if widget.Widget.IsFocused() {
			style = "highlight"
		}
		widget.Label.SetStyleName(style)
	}
}

func (m *LoginController) SetInvalidAccountPathState() {
	m.focusController = interactions.NewFocusController(interactions.NewFocusChain(m.view.keystoreEdit, m.view.cancelButton))
	m.HighlightActiveWidget()
}

func (m *LoginController) setInvalidAccountState() {
	m.focusController = interactions.NewFocusController(interactions.NewFocusChain(m.view.keystoreEdit, m.view.accountEdit, m.view.cancelButton))
	m.HighlightActiveWidget()
}

func (m *LoginController) setValidAccountState() {
	m.focusController = interactions.NewFocusController(interactions.NewFocusChain(m.view.accountEdit, m.view.passwordEdit, m.view.accountUnlockButton, m.view.cancelButton, m.view.keystoreEdit))
	m.HighlightActiveWidget()
}

func (m *LoginController) Reset() {
	m.SetInvalidAccountPathState()
	m.view.SetFocused(true)
}
