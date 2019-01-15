package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"image"
	"os"
	"time"

	"github.com/3Hren/sonmui/icli/internal/config"
	"github.com/3Hren/sonmui/icli/internal/interactions"
	"github.com/3Hren/sonmui/icli/internal/mp"
	"github.com/3Hren/sonmui/icli/internal/views"
	"github.com/3Hren/sonmui/icli/internal/widgets"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/marcusolsson/tui-go"
	"github.com/sonm-io/core/accounts"
	"github.com/sonm-io/core/insonmnia/auth"
	"github.com/sonm-io/core/insonmnia/version"
	"github.com/sonm-io/core/proto"
	"github.com/sonm-io/core/util"
	"github.com/sonm-io/core/util/xgrpc"
	"google.golang.org/grpc"
)

type MainView struct {
	*tui.Box

	currentNodeNLabel    *tui.Label
	currentNodeVLabel    *widgets.AsyncLabel
	currentAccountVLabel *widgets.AsyncLabel
	currentBalanceVLabel *widgets.AsyncLabel
	orderCountVLabel     *widgets.AsyncLabel
	dealCountVLabel      *widgets.AsyncLabel
	menuList             *widgets.List
	submenuList          *widgets.List

	controlBox  *tui.Box
	menuBox     *tui.Box
	submenuBox  *tui.Box
	workersView *WorkerListWidget
}

func NewMainView(ctx context.Context, router *mp.Router) *MainView {
	currentNodeNLabel := tui.NewLabel("Node:")
	currentNodeNLabel.SetStyleName("bold")
	currentNodeVLabel := widgets.NewAsyncLabel(ctx, "-", router)
	currentAccountNLabel := tui.NewLabel("Account:")
	currentAccountNLabel.SetStyleName("bold")
	currentAccountVLabel := widgets.NewAsyncLabel(ctx, "-", router)
	currentBalanceNLabel := tui.NewLabel("Balance:")
	currentBalanceNLabel.SetStyleName("bold")
	currentBalanceVLabel := widgets.NewAsyncLabel(ctx, "-", router)
	orderCountNLabel := tui.NewLabel("Orders:")
	orderCountNLabel.SetStyleName("bold")
	orderCountVLabel := widgets.NewAsyncLabel(ctx, "-", router)
	dealCountNLabel := tui.NewLabel("Deals:")
	dealCountNLabel.SetStyleName("bold")
	dealCountVLabel := widgets.NewAsyncLabel(ctx, "-", router)

	nColumnBox := tui.NewVBox(
		currentNodeNLabel,
		currentAccountNLabel,
		currentBalanceNLabel,
		orderCountNLabel,
		dealCountNLabel,
	)
	vColumnBox := tui.NewVBox(
		currentNodeVLabel,
		currentAccountVLabel,
		currentBalanceVLabel,
		orderCountVLabel,
		dealCountVLabel,
	)
	vColumnBox.SetSizePolicy(tui.Expanding, tui.Preferred)

	summaryBox := tui.NewHBox(tui.NewPadder(1, 0, nColumnBox), vColumnBox)
	summaryBox.SetBorder(true)
	summaryBox.SetSizePolicy(tui.Preferred, tui.Minimum)

	menuList := widgets.NewList()
	menuList.AddItems("Workers", "Exit")

	menuBox := tui.NewVBox(tui.NewPadder(1, 0, menuList), tui.NewPadder(32, 0, tui.NewSpacer()))
	menuBox.SetBorder(true)
	menuBox.SetSizePolicy(tui.Preferred, tui.Preferred)

	submenuList := widgets.NewList()

	submenuBox := tui.NewHBox(tui.NewPadder(1, 0, submenuList))
	submenuBox.SetBorder(true)
	submenuBox.SetSizePolicy(tui.Expanding, tui.Preferred)

	workersView := NewWorkerListWidget(router)
	workersView.SetBorder(true)
	workersView.SetSizePolicy(tui.Expanding, tui.Preferred)

	controlBox := tui.NewHBox(menuBox, submenuBox)
	controlBox.SetSizePolicy(tui.Preferred, tui.Expanding)

	box := tui.NewVBox(
		summaryBox,
		controlBox,
	)

	return &MainView{
		Box: box,

		currentNodeNLabel:    currentNodeNLabel,
		currentNodeVLabel:    currentNodeVLabel,
		currentAccountVLabel: currentAccountVLabel,
		currentBalanceVLabel: currentBalanceVLabel,
		orderCountVLabel:     orderCountVLabel,
		dealCountVLabel:      dealCountVLabel,
		menuList:             menuList,
		submenuList:          submenuList,

		controlBox:  controlBox,
		menuBox:     menuBox,
		submenuBox:  submenuBox,
		workersView: workersView,
	}
}

type nodeConnectEvent struct {
	Addr       string
	PrivateKey *ecdsa.PrivateKey
}

type nodeConnectionResultEvent struct {
	Addr  string
	Conn  *grpc.ClientConn
	Error error
}

type workersListUpdateEvent struct{}

type workerConfirmEvent struct {
	ID string
}

type workerConfirmDoneEvent struct {
	ID    string
	Error error
}

type workersUpdateUptimeEvent struct{}

type confirmationStatus int

const (
	Unconfirmed confirmationStatus = iota
	InProgress
	Confirmed
)

type workerItem struct {
	Addr               common.Address
	ConfirmationStatus confirmationStatus
}

// ========================================================================================================================

// WorkerListWidget is an interactive worker list.
//
// | 0x... | [v] |
type WorkerListWidget struct {
	*tui.Box

	router *mp.Router

	workers          []*workerItem
	workersList      *widgets.List
	workersStatusBox *tui.Box
	workersUptimeBox *tui.Box

	OnSelectionChanged *mp.Signal
}

func NewWorkerListWidget(router *mp.Router) *WorkerListWidget {
	workersList := widgets.NewList()
	workersStatusBox := tui.NewVBox(tui.NewSpacer())
	workersUptimeBox := tui.NewVBox(tui.NewSpacer())

	workerETHAddrLabel := tui.NewLabel("ETH Address")
	workerETHAddrLabel.SetStyleName("highlight")

	workerConfirmedLabel := tui.NewLabel("Confirmed")
	workerConfirmedLabel.SetStyleName("highlight")

	box := tui.NewHBox(
		tui.NewVBox(tui.NewPadder(1, 0, workerETHAddrLabel), tui.NewPadder(1, 1, workersList)),
		tui.NewVBox(tui.NewPadder(1, 0, workerConfirmedLabel), tui.NewPadder(1, 1, workersStatusBox)),
		tui.NewSpacer(),
	)

	return &WorkerListWidget{
		Box: tui.NewVBox(box, tui.NewSpacer()),

		router: router,

		workersList:      workersList,
		workersStatusBox: workersStatusBox,
		workersUptimeBox: workersUptimeBox,

		OnSelectionChanged: router.NewSignal(),
	}
}

func (m *WorkerListWidget) pos(addr common.Address) int {
	pos := -1
	for id, worker := range m.workers {
		if worker.Addr == addr {
			pos = id
			break
		}
	}

	return pos
}

func (m *WorkerListWidget) AddItem(item *workerItem) {
	ctx := context.Background()
	m.workersList.AddItems(item.Addr.Hex())

	var status *widgets.AsyncLabel
	switch item.ConfirmationStatus {
	case Unconfirmed:
		status = widgets.NewAsyncLabel(ctx, "✖", m.router)
		status.SetStyleName("warn")
	case InProgress:
		status = widgets.NewAsyncLabel(ctx, "✖", m.router)
		status.RunProgress(ctx)
		status.SetStyleName("warn")
	case Confirmed:
		status = widgets.NewAsyncLabel(ctx, "✓", m.router)
		status.SetStyleName("succ")
	}

	m.workersStatusBox.Append(tui.NewHBox(status))
	m.workers = append(m.workers, item)
}

func (m *WorkerListWidget) ReplaceItem(item *workerItem) {
	ctx := context.Background()

	pos := m.pos(item.Addr)
	if pos == -1 {
		return
	}

	var status *widgets.AsyncLabel
	switch item.ConfirmationStatus {
	case Unconfirmed:
		status = widgets.NewAsyncLabel(ctx, "✖", m.router)
		status.SetStyleName("warn")
	case InProgress:
		status = widgets.NewAsyncLabel(ctx, "✖", m.router)
		status.RunProgress(ctx)
		status.SetStyleName("warn")
	case Confirmed:
		status = widgets.NewAsyncLabel(ctx, "✓", m.router)
		status.SetStyleName("succ")
	}

	m.workersStatusBox.Remove(pos)
	m.workersStatusBox.Insert(pos, tui.NewHBox(status))
}

func (m *WorkerListWidget) SetFocused(focused bool) {
	m.workersList.SetFocused(focused)
}

func (m *WorkerListWidget) Length() int {
	return m.workersList.Length()
}

func (m *WorkerListWidget) Selected() int {
	return m.workersList.Selected()
}

func (m *WorkerListWidget) SelectedItem() string {
	return m.workersList.SelectedItem()
}

func (m *WorkerListWidget) Select(v int) {
	if m.Length() > 0 {
		m.workersList.Select(v)
	}
}

func (m *WorkerListWidget) Clear() {
	m.workersList.ReplaceItems()
	for m.workersStatusBox.Length() > 0 {
		m.workersStatusBox.Remove(0)
	}
	m.workers = nil
}

func (m *WorkerListWidget) OverrideOnKeyEvent(fn func(ev tui.KeyEvent) bool) {
	m.workersList.OnKeyEventX = fn
}

// ========================================================================================================================

type MainController struct {
	view *MainView

	eventTxRx chan interface{}
}

func NewMainController(ctx context.Context, view *MainView) *MainController {
	eventTxRx := make(chan interface{}, 128)

	view.menuList.OnSelectionChanged(func(menu *tui.List) {
		if menu.Selected() == -1 {
			return
		}

		switch menu.SelectedItem() {
		case "Accounts":
			view.submenuList.ReplaceItems("Switch", "Create", "Logout")
		case "Workers":
			view.controlBox.Remove(1)
			view.controlBox.Insert(1, view.workersView)

			eventTxRx <- &workersListUpdateEvent{}
		default:
			view.controlBox.Remove(1)
			view.controlBox.Insert(1, view.submenuBox)
			view.submenuList.ReplaceItems()
		}
	})
	view.menuList.OnItemActivated(func(menu *tui.List) {
		switch menu.SelectedItem() {
		case "Workers":
			view.menuList.SetFocused(false)
			view.workersView.SetFocused(true)
			view.workersView.Select(0)
		case "Exit":
			print("\033[H\033[2J")
			os.Exit(0)
		default:
		}
	})
	view.menuList.OnKeyEventX = func(ev tui.KeyEvent) bool {
		switch ev.Key {
		case tui.KeyRight:
			selectedItem := view.menuList.SelectedItem()

			switch selectedItem {
			case "Workers":
				view.menuList.SetFocused(false)
				view.workersView.SetFocused(true)
				view.workersView.Select(0)
			default:
				return false
			}
			return true
		default:
			return false
		}
	}
	view.workersView.OverrideOnKeyEvent(func(ev tui.KeyEvent) bool {
		switch ev.Key {
		case tui.KeyLeft:
			view.menuList.SetFocused(true)
			view.workersView.SetFocused(false)
			view.workersView.Select(-1)
			return true
		case tui.KeyRune:
			switch ev.Rune {
			case 'c':
				eventTxRx <- &workerConfirmEvent{ID: view.workersView.SelectedItem()}
				return true
			}
			return false
		default:
			return false
		}
	})

	m := &MainController{
		view:      view,
		eventTxRx: eventTxRx,
	}

	go m.run(ctx)

	m.view.menuList.Select(0)
	m.view.menuList.SetFocused(true)

	return m
}

func (m *MainController) run(ctx context.Context) {
	addr := common.Address{}
	var nodeConn *grpc.ClientConn

	workersConfirmationInProgress := map[string]struct{}{}
	workersConfirmed := map[string]struct{}{}

	workerStatusTimer := util.NewImmediateTicker(60 * time.Second)
	defer workerStatusTimer.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case ev := <-m.eventTxRx:
			switch event := ev.(type) {
			case *nodeConnectEvent:
				addr = crypto.PubkeyToAddress(event.PrivateKey.PublicKey)
				m.connectToNodeAsync(ctx, event.Addr, event.PrivateKey)
			case *nodeConnectionResultEvent:
				if event.Error != nil {
					m.view.currentNodeVLabel.StopProgress(event.Error.Error())
					continue
				}

				nodeConn = event.Conn
				m.onNodeConnected(ctx, event.Conn, addr)
				m.eventTxRx <- &workersListUpdateEvent{}
			case *workersListUpdateEvent:
				if nodeConn != nil {
					pos := m.view.workersView.Selected()
					node := sonm.NewMasterManagementClient(nodeConn)
					workers, err := node.WorkersList(ctx, &sonm.EthAddress{
						Address: addr.Bytes(),
					})
					if err != nil {
						return
					}

					m.view.workersView.Clear()

					for _, worker := range workers.GetWorkers() {
						workerItem := &workerItem{
							Addr:               worker.GetSlaveID().Unwrap(),
							ConfirmationStatus: Unconfirmed,
						}

						if _, ok := workersConfirmationInProgress[worker.GetSlaveID().Unwrap().Hex()]; ok {
							workerItem.ConfirmationStatus = InProgress
						} else if worker.Confirmed {
							workerItem.ConfirmationStatus = Confirmed
							workersConfirmed[worker.GetSlaveID().Unwrap().Hex()] = struct{}{}
						}
						m.view.workersView.AddItem(workerItem)
					}

					m.view.workersView.workersStatusBox.Append(tui.NewSpacer())
					m.view.workersView.Select(pos)
				} else {
					go func() {
						time.Sleep(1 * time.Second)
						m.eventTxRx <- &workersListUpdateEvent{}
					}()
				}
			case *workerConfirmEvent:
				workerAddr := common.HexToAddress(event.ID)
				if _, ok := workersConfirmed[event.ID]; ok {
					continue
				}

				if nodeConn != nil {
					node := sonm.NewMasterManagementClient(nodeConn)
					workersConfirmationInProgress[event.ID] = struct{}{}

					m.view.workersView.ReplaceItem(&workerItem{Addr: common.HexToAddress(event.ID), ConfirmationStatus: InProgress})

					go func() {
						_, err := node.WorkerConfirm(ctx, sonm.NewEthAddress(workerAddr))

						m.eventTxRx <- &workerConfirmDoneEvent{ID: event.ID, Error: err}
					}()
				}
			case *workerConfirmDoneEvent:
				m.eventTxRx <- &workersListUpdateEvent{}
				delete(workersConfirmationInProgress, event.ID)
			case *workersUpdateUptimeEvent:
				if nodeConn != nil {
					//node := sonm.NewWorkerManagementClient(nodeConn)
					//
					//for addr := range workersConfirmed {
					//	addr := common.HexToAddress(addr)
					//	md := metadata.MD{
					//		util.WorkerAddressHeader: []string{addr.Hex()},
					//	}
					//
					//	m.view.workersView.SetUptimeProgress(addr)
					//	go func() {
					//		ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
					//		defer cancel()
					//
					//		ctx = metadata.NewOutgoingContext(ctx, md)
					//		status, err := node.Status(ctx, &sonm.Empty{})
					//
					//		m.uiEventTxRx <- func() {
					//			if err != nil {
					//				m.view.workersView.ResetUptime(addr)
					//			} else {
					//				m.view.workersView.SetUptime(addr, status.GetUptime())
					//			}
					//		}
					//	}()
					//}
				}
			}
		case <-workerStatusTimer.C:
			m.eventTxRx <- &workersUpdateUptimeEvent{}
		}
	}
}

func (m *MainController) connectToNodeAsync(ctx context.Context, addr string, privateKey *ecdsa.PrivateKey) {
	m.view.currentNodeVLabel.RunProgress(ctx)

	fn := func() (*grpc.ClientConn, error) {
		_, TLSConfig, err := util.NewHitlessCertRotator(ctx, privateKey)
		if err != nil {
			return nil, err
		}

		credentials := auth.NewWalletAuthenticator(util.NewTLS(TLSConfig), crypto.PubkeyToAddress(privateKey.PublicKey))

		ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()

		return xgrpc.NewClient(ctx, addr, credentials, grpc.WithBlock())
	}

	go func() {
		conn, err := fn()
		if err != nil {
			m.eventTxRx <- &nodeConnectionResultEvent{Addr: addr, Error: err}
		} else {
			m.eventTxRx <- &nodeConnectionResultEvent{Addr: addr, Conn: conn}
		}
	}()
}

func (m *MainController) onNodeConnected(ctx context.Context, conn *grpc.ClientConn, addr common.Address) {
	m.view.currentNodeVLabel.StopProgress(conn.Target())

	m.view.currentAccountVLabel.SetText(addr.Hex())
	m.view.currentBalanceVLabel.SetTextAsync(ctx, func(ctx context.Context) string {
		node := sonm.NewTokenManagementClient(conn)
		balance, err := node.BalanceOf(ctx, &sonm.EthAddress{
			Address: addr.Bytes(),
		})
		if err != nil {
			return err.Error()
		}

		return balance.SideBalance.ToPriceString()
	})
	m.view.orderCountVLabel.SetTextAsync(ctx, func(ctx context.Context) string {
		node := sonm.NewMarketClient(conn)
		orders, err := node.GetOrders(ctx, &sonm.Count{})
		if err != nil {
			return err.Error()
		}

		return fmt.Sprintf("%d", len(orders.GetOrders()))
	})

	m.view.dealCountVLabel.SetTextAsync(ctx, func(ctx context.Context) string {
		node := sonm.NewDWHClient(conn)
		orders, err := node.GetDeals(ctx, &sonm.DealsRequest{
			Status:    sonm.DealStatus_DEAL_ACCEPTED,
			AnyUserID: &sonm.EthAddress{Address: addr.Bytes()},
			WithCount: true,
		})
		if err != nil {
			return err.Error()
		}

		return fmt.Sprintf("%d", orders.GetCount())
	})
}

func (m *MainController) SetAccount(privateKey *ecdsa.PrivateKey) {
	m.eventTxRx <- &nodeConnectEvent{Addr: "localhost:15030", PrivateKey: privateKey}
}

// ------------------------------------------------------------------------
type WelcomeView struct {
	*widgets.FocusBox

	accountsList     *widgets.List
	loginOtherButton *tui.Button
}

func NewWelcomeView() *WelcomeView {
	logoLabel := tui.NewLabel(logo)
	logoLabel.SetStyleName("logo")

	accountsList := widgets.NewList()
	loginOtherButton := tui.NewButton("[Login Other]")

	buttonsBox := tui.NewHBox(
		tui.NewSpacer(),
		//tui.NewPadder(1, 0, loginOtherButton),
		tui.NewSpacer(),
	)

	windowBox := tui.NewVBox(
		tui.NewPadder(0, 1, logoLabel),
		tui.NewPadder(0, 0, tui.NewLabel(welcomeText)),
		tui.NewPadder(0, 1, accountsList),
		buttonsBox,
	)

	wrapperBox := tui.NewVBox(
		tui.NewSpacer(),
		windowBox,
		tui.NewSpacer(),
		tui.NewSpacer(),
	)
	contentBox := tui.NewHBox(tui.NewSpacer(), wrapperBox, tui.NewSpacer())

	return &WelcomeView{
		FocusBox: widgets.NewFocusBox(contentBox, nil),

		accountsList:     accountsList,
		loginOtherButton: loginOtherButton,
	}
}

func (m *WelcomeView) Hide() {
	m.FocusBox.SetFocused(false)
}

type WelcomeController struct {
	OnLogin      *mp.Signal
	OnLoginOther *mp.Signal
}

func NewWelcomeController(view *WelcomeView, router *mp.Router, accounts map[common.Address]string) *WelcomeController {
	for account := range accounts {
		view.accountsList.AddItems(account.Hex())
	}

	focusChain := interactions.NewFocusChain()
	focusController := interactions.NewFocusController(focusChain)

	if view.accountsList.Length() > 0 {
		focusChain.AddWidget(view.accountsList)

		view.accountsList.OnKeyEventX = func(ev tui.KeyEvent) bool {
			switch ev.Key {
			case tui.KeyTab:
				focusController.FocusNextWidget()
				return true
			default:
				return false
			}
		}
	}

	onLogin := router.NewSignal()
	onLoginOther := router.NewSignal()

	view.accountsList.OnItemActivated(func(menu *tui.List) { onLogin.Emit(menu.SelectedItem()) })
	view.loginOtherButton.OnActivated(func(*tui.Button) { onLoginOther.Emit(struct{}{}) })

	focusChain.AddWidget(view.loginOtherButton)
	focusController.FocusDefaultWidget()

	view.SetFocusController(focusController)

	return &WelcomeController{
		OnLogin:      onLogin,
		OnLoginOther: onLoginOther,
	}
}

// ------------------------------------------------------------------------

type PasswordView struct {
	*widgets.FocusBox

	accountVLabel *tui.Label
	entry         *widgets.Entry
	unlockButton  *tui.Button
	cancelButton  *tui.Button
}

func NewPasswordView() *PasswordView {
	helpLabel := tui.NewLabel("Enter password for the account")
	helpLabel.SetStyleName("title")

	accountLabel := tui.NewLabel("Account:")
	accountLabel.SetStyleName("highlight")
	accountVLabel := tui.NewLabel("")
	passwordLabel := tui.NewLabel("Password:")
	passwordLabel.SetStyleName("highlight")
	passwordEntry := widgets.NewEntry()
	passwordEntry.SetEchoMode(tui.EchoModePassword)
	passwordEntry.SetSizeHint(image.Point{X: 32, Y: 1})

	unlockButton := tui.NewButton("[Unlock]")
	cancelButton := tui.NewButton("[Cancel]")

	contentBox := tui.NewVBox(
		tui.NewPadder(1, 1, helpLabel),
		tui.NewHBox(
			tui.NewVBox(tui.NewPadder(1, 0, accountLabel), tui.NewPadder(1, 0, passwordLabel)),
			tui.NewVBox(accountVLabel, passwordEntry),
			tui.NewSpacer(),
		),
		tui.NewHBox(tui.NewSpacer(), tui.NewPadder(1, 0, unlockButton), cancelButton),
	)

	box := tui.NewHBox(
		tui.NewSpacer(),
		tui.NewVBox(
			tui.NewSpacer(),
			contentBox,
			tui.NewSpacer(),
			tui.NewSpacer(),
		),
		tui.NewSpacer(),
	)

	return &PasswordView{
		FocusBox: widgets.NewFocusBox(box, nil),

		accountVLabel: accountVLabel,
		entry:         passwordEntry,
		unlockButton:  unlockButton,
		cancelButton:  cancelButton,
	}
}

func (m *PasswordView) SetFocused(focused bool) {
	m.entry.SetFocused(focused)
}

type PasswordController struct {
	view            *PasswordView
	focusController *interactions.FocusController

	OnSubmit *mp.Signal
	OnCancel *mp.Signal
}

func NewPasswordController(view *PasswordView, router *mp.Router) *PasswordController {
	focusChain := interactions.NewFocusChain()
	focusChain.AddWidget(view.entry)
	focusChain.AddWidget(view.unlockButton)
	focusChain.AddWidget(view.cancelButton)

	focusController := interactions.NewFocusController(focusChain)
	focusController.FocusDefaultWidget()

	onSubmit := router.NewSignal()
	onCancel := router.NewSignal()

	view.entry.OnSubmit(func(entry *tui.Entry) {
		onSubmit.Emit(entry.Text())
	})
	view.unlockButton.OnActivated(func(*tui.Button) {
		onSubmit.Emit(view.entry.Text())
	})
	view.cancelButton.OnActivated(func(*tui.Button) {
		onCancel.Emit(struct{}{})
	})

	view.SetFocusController(focusController)

	return &PasswordController{
		view:            view,
		focusController: focusController,

		OnSubmit: onSubmit,
		OnCancel: onCancel,
	}
}

func (m *PasswordController) Reset() {
	m.view.accountVLabel.SetText("")
	m.view.entry.SetText("")
	m.focusController.FocusDefaultWidget()
}

func (m *PasswordController) CurrentAccount() common.Address {
	return common.HexToAddress(m.view.accountVLabel.Text())
}

func (m *PasswordController) SetAccount(account common.Address) {
	m.view.accountVLabel.SetText(account.Hex())
}

// ------------------------------------------------------------------------

func exec() error {
	cfg, err := config.LoadConfig(config.DefaultConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			cfg = config.NewConfig()
		} else {
			return err
		}
	}

	ctx := context.Background()
	router := mp.NewRouter()

	welcomeView := NewWelcomeView()
	passwordView := NewPasswordView()
	loginView := views.NewLoginView()
	mainView := NewMainView(ctx, router)

	statusBar := tui.NewStatusBar("Select previously used account and press <Enter> to specify password or select <Login Other> button to login into other account.")
	statusBar.SetPermanentText(version.Version)

	root := tui.NewVBox(
		welcomeView,
		statusBar,
	)

	ui, err := tui.New(root)
	if err != nil {
		return err
	}

	// Controllers.

	welcomeController := NewWelcomeController(welcomeView, router, cfg.AccountPaths)
	passwordController := NewPasswordController(passwordView, router)
	loginController := views.NewLoginController(loginView, router)
	mainController := NewMainController(ctx, mainView)

	welcomeController.OnLogin.Connect(func(v interface{}) {
		passwordController.Reset()
		passwordController.SetAccount(common.HexToAddress(v.(string)))

		ui.SetWidget(tui.NewVBox(
			passwordView,
			statusBar,
		))
	})
	welcomeController.OnLoginOther.Connect(func(interface{}) {
		loginController.Reset()

		ui.SetWidget(tui.NewVBox(
			loginView,
			statusBar,
		))
		statusBar.SetText("Specify directory with keystore. <Tab> for completion, <Enter> for submitting")
	})

	passwordController.OnSubmit.Connect(func(v interface{}) {
		account := passwordController.CurrentAccount()
		password := v.(string)
		path, ok := cfg.AccountPaths[account]
		if !ok {
			statusBar.SetText(err.Error())
			return
		}

		keystore, err := accounts.NewMultiKeystore(accounts.NewKeystoreConfig(path), accounts.NewStaticPassPhraser(""))
		if err != nil {
			statusBar.SetText(err.Error())
			return
		}

		privateKey, err := keystore.GetKeyWithPass(account, password)
		if err != nil {
			statusBar.SetText(err.Error())
			return
		}

		mainController.SetAccount(privateKey)

		ui.SetWidget(tui.NewVBox(
			mainView,
			statusBar,
		))
	})
	passwordController.OnCancel.Connect(func(v interface{}) {
		ui.SetWidget(tui.NewVBox(
			welcomeView,
			statusBar,
		))
	})

	loginController.OnCancel.Connect(func(v interface{}) {
		ui.SetWidget(tui.NewVBox(
			welcomeView,
			statusBar,
		))
	})

	ui.SetTheme(DefaultTheme())
	ui.SetKeybinding("Ctrl+C", ui.Quit)

	go func() {
		for fn := range router.Rx() {
			ui.Update(fn)
		}
	}()

	if err := ui.Run(); err != nil {
		return err
	}

	return nil
}

func main() {
	if err := exec(); err != nil {
		os.Exit(1)
	}
}
