// Package app provides the core TUI application shell for dts-cli.
package app

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/microsoft/durabletask-scheduler/cli/internal/api"
	"github.com/microsoft/durabletask-scheduler/cli/internal/config"
	"github.com/microsoft/durabletask-scheduler/cli/internal/ui/components"
)

// View represents a navigable view in the application.
type View interface {
	// Name returns the view identifier.
	Name() string
	// Primitive returns the tview primitive to display.
	Primitive() tview.Primitive
	// Init initializes the view (or refreshes data).
	Init(ctx context.Context)
	// Hints returns keybinding hints for the status bar.
	Hints() []components.KeyHint
	// Crumbs returns breadcrumb items for the navigation bar.
	Crumbs() []string
}

// App is the main TUI application.
type App struct {
	tviewApp      *tview.Application
	pages         *tview.Pages
	titleBar      *components.TitleBar
	crumbs        *components.Crumbs
	statusBar     *components.StatusBar
	mainLayout    *tview.Flex
	cmdInput      *components.CommandInput
	filterInput   *components.FilterInput
	cmdVisible    bool
	filterVisible bool

	Config *config.Config
	Client *api.Client

	viewStack []View
	mu        sync.Mutex

	// Auto-refresh
	refreshCancel   context.CancelFunc
	refreshTicker   *time.Ticker
	countdownCancel context.CancelFunc

	// Callbacks for creating views
	ViewFactory ViewFactory
}

// ViewFactory creates views by name.
type ViewFactory interface {
	CreateView(app *App, name string, params ...string) View
}

// New creates a new App instance.
func New(cfg *config.Config, client *api.Client) *App {
	a := &App{
		tviewApp:  tview.NewApplication(),
		pages:     tview.NewPages(),
		titleBar:  components.NewTitleBar(),
		crumbs:    components.NewCrumbs(),
		statusBar: components.NewStatusBar(),
		Config:    cfg,
		Client:    client,
	}

	// Command input (triggered by ':')
	a.cmdInput = components.NewCommandInput(
		func(cmd string) {
			a.hideCommandInput()
			a.handleCommand(cmd)
		},
		func() {
			a.hideCommandInput()
		},
	)

	// Filter input (triggered by '/')
	a.filterInput = components.NewFilterInput(
		func(filter string) {
			a.hideFilterInput()
			a.handleFilter(filter)
		},
		func() {
			a.hideFilterInput()
		},
	)

	// Main layout: titlebar | crumbs | content | statusbar
	a.mainLayout = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.titleBar, 3, 0, false).
		AddItem(a.crumbs, 1, 0, false).
		AddItem(a.pages, 0, 1, true).
		AddItem(a.statusBar, 1, 0, false)

	// Setup global key handler
	a.tviewApp.SetInputCapture(a.globalKeyHandler)

	if cfg.CurrentContext != "" {
		a.statusBar.SetContext(cfg.CurrentContext)
		if ctx, ok := cfg.Contexts[cfg.CurrentContext]; ok {
			a.titleBar.SetContext(ctx.URL, ctx.TaskHub)
		}
	}

	return a
}

// ShowSplash displays a splash screen that dismisses after 5 seconds or any key press.
// The onDone callback is invoked once the splash is dismissed (e.g. to navigate to the starting view).
func (a *App) ShowSplash(onDone func()) {
	splash := components.NewSplashScreen()
	a.tviewApp.SetRoot(splash, true)

	var once sync.Once

	transition := func() {
		a.tviewApp.SetRoot(a.mainLayout, true)
		a.tviewApp.SetInputCapture(a.globalKeyHandler)
		if onDone != nil {
			onDone()
		}
	}

	done := make(chan struct{})

	// Any key press dismisses the splash (runs on main goroutine).
	a.tviewApp.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		once.Do(func() {
			close(done)
			transition()
		})
		return nil
	})

	// Auto-dismiss after 5 seconds (runs on background goroutine).
	go func() {
		timer := time.NewTimer(5 * time.Second)
		defer timer.Stop()
		select {
		case <-timer.C:
			once.Do(func() {
				close(done)
				a.tviewApp.QueueUpdateDraw(transition)
			})
		case <-done:
		}
	}()
}

// Run starts the TUI application.
func (a *App) Run() error {
	return a.tviewApp.Run()
}

// Stop gracefully stops the application.
func (a *App) Stop() {
	a.stopAutoRefresh()
	a.tviewApp.Stop()
}

// Navigate pushes a new view onto the view stack.
func (a *App) Navigate(view View) {
	a.mu.Lock()
	a.viewStack = append(a.viewStack, view)
	a.mu.Unlock()

	a.showView(view)
}

// Back pops the current view and returns to the previous one.
func (a *App) Back() {
	a.mu.Lock()
	if len(a.viewStack) <= 1 {
		a.mu.Unlock()
		return
	}
	a.viewStack = a.viewStack[:len(a.viewStack)-1]
	view := a.viewStack[len(a.viewStack)-1]
	a.mu.Unlock()

	a.showView(view)
}

// CurrentView returns the active view.
func (a *App) CurrentView() View {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.viewStack) == 0 {
		return nil
	}
	return a.viewStack[len(a.viewStack)-1]
}

// Refresh re-initializes the current view and resets the auto-refresh countdown.
func (a *App) Refresh() {
	view := a.CurrentView()
	if view != nil {
		go func() {
			ctx := context.Background()
			view.Init(ctx)
			a.tviewApp.Draw()
		}()
	}
	// Restart the countdown timer
	a.startAutoRefresh()
}

// Flash shows a temporary message in the status bar.
func (a *App) Flash(msg string, color tcell.Color) {
	a.statusBar.Flash(msg, color)
	go func() {
		time.Sleep(3 * time.Second)
		a.tviewApp.QueueUpdateDraw(func() {
			a.statusBar.ClearFlash()
		})
	}()
}

// FlashSuccess shows a green success message.
func (a *App) FlashSuccess(msg string) {
	a.Flash(msg, tcell.ColorGreen)
}

// FlashError shows a red error message.
func (a *App) FlashError(msg string) {
	a.Flash(msg, tcell.ColorRed)
}

// FlashInfo shows an informational message.
func (a *App) FlashInfo(msg string) {
	a.Flash(msg, tcell.ColorYellow)
}

// Pages returns the pages container for dialog overlays.
func (a *App) Pages() *tview.Pages {
	return a.pages
}

// SetTitleContext updates the title bar with endpoint and task hub.
func (a *App) SetTitleContext(url, taskHub string) {
	a.titleBar.SetContext(url, taskHub)
}

// TviewApp returns the underlying tview.Application.
func (a *App) TviewApp() *tview.Application {
	return a.tviewApp
}

// QueueUpdateDraw queues a UI update on the main thread.
func (a *App) QueueUpdateDraw(f func()) {
	a.tviewApp.QueueUpdateDraw(f)
}

// ShowConfirm shows a confirmation dialog.
func (a *App) ShowConfirm(title, message string, onConfirm func()) {
	a.hideCommandInput()
	a.hideFilterInput()
	components.ConfirmDialog(a.tviewApp, a.pages, title, message, onConfirm)
}

// NavigateToResource navigates to a resource view by name.
func (a *App) NavigateToResource(name string, params ...string) {
	if a.ViewFactory == nil {
		return
	}
	view := a.ViewFactory.CreateView(a, name, params...)
	if view != nil {
		// Clear stack and push new view (lateral navigation)
		a.mu.Lock()
		if len(a.viewStack) > 1 {
			a.viewStack = a.viewStack[:1] // Keep home
		}
		a.mu.Unlock()
		a.Navigate(view)
	}
}

func (a *App) showView(view View) {
	pageName := view.Name()

	a.pages.AddAndSwitchToPage(pageName, view.Primitive(), true)
	a.crumbs.SetCrumbs(view.Crumbs()...)
	a.statusBar.SetResource(view.Name())
	a.statusBar.SetHints(view.Hints())
	a.statusBar.SetFilter("")
	a.tviewApp.SetFocus(view.Primitive())

	// Initialize/refresh data
	go func() {
		ctx := context.Background()
		view.Init(ctx)
		a.tviewApp.Draw()
	}()

	// Setup auto-refresh
	a.startAutoRefresh()
}

func (a *App) showCommandInput() {
	if a.cmdVisible {
		return
	}
	a.cmdVisible = true
	a.cmdInput.SetText("")
	a.mainLayout.RemoveItem(a.statusBar)
	a.mainLayout.AddItem(a.cmdInput, 1, 0, true)
	a.tviewApp.SetFocus(a.cmdInput)
}

func (a *App) hideCommandInput() {
	if !a.cmdVisible {
		return
	}
	a.cmdVisible = false
	a.mainLayout.RemoveItem(a.cmdInput)
	a.mainLayout.AddItem(a.statusBar, 1, 0, false)
	view := a.CurrentView()
	if view != nil {
		a.tviewApp.SetFocus(view.Primitive())
	}
}

func (a *App) showFilterInput() {
	if a.filterVisible {
		return
	}
	a.filterVisible = true
	a.filterInput.SetText("")
	a.mainLayout.RemoveItem(a.statusBar)
	a.mainLayout.AddItem(a.filterInput, 1, 0, true)
	a.tviewApp.SetFocus(a.filterInput)
}

func (a *App) hideFilterInput() {
	if !a.filterVisible {
		return
	}
	a.filterVisible = false
	a.mainLayout.RemoveItem(a.filterInput)
	a.mainLayout.AddItem(a.statusBar, 1, 0, false)
	view := a.CurrentView()
	if view != nil {
		a.tviewApp.SetFocus(view.Primitive())
	}
}

func (a *App) handleCommand(cmd string) {
	cmd = strings.TrimSpace(cmd)
	parts := strings.Fields(cmd)
	if len(parts) == 0 {
		return
	}

	switch strings.ToLower(parts[0]) {
	case "q!", "quit!":
		a.Stop()
	case "q", "quit":
		a.ShowConfirm("Quit", "Are you sure you want to quit?", func() {
			a.Stop()
		})
	case "orchestrations", "orch":
		a.NavigateToResource("orchestrations")
	case "entities", "ent":
		a.NavigateToResource("entities")
	case "schedules", "sched":
		a.NavigateToResource("schedules")
	case "workers", "work":
		a.NavigateToResource("workers")
	case "agents", "ag":
		a.NavigateToResource("agents")
	case "home":
		a.NavigateToResource("home")
	case "help":
		a.NavigateToResource("help")
	case "ctx":
		if len(parts) >= 2 {
			a.switchContext(parts[1])
		}
	default:
		a.FlashError("Unknown command: " + cmd)
	}
}

func (a *App) handleFilter(filter string) {
	a.statusBar.SetFilter(filter)
	// The current view handles filtering via its own mechanism
	// This is dispatched via the view's filter handler
}

func (a *App) switchContext(name string) {
	ctx, ok := a.Config.Contexts[name]
	if !ok {
		a.FlashError("Unknown context: " + name)
		return
	}
	a.Config.CurrentContext = name
	a.Client = api.NewClient(ctx.URL, ctx.TaskHub, nil) // Will need auth re-init
	a.statusBar.SetContext(name)
	a.titleBar.SetContext(ctx.URL, ctx.TaskHub)
	a.FlashSuccess("Switched to context: " + name)
	_ = a.Config.Save()
}

func (a *App) startAutoRefresh() {
	a.stopAutoRefresh()
	interval := a.Config.Settings.RefreshInterval
	if interval <= 0 {
		return
	}

	a.statusBar.SetCountdown(interval)

	ctx, cancel := context.WithCancel(context.Background())
	a.refreshCancel = cancel

	// 1-second countdown ticker
	countdownCtx, countdownCancel := context.WithCancel(context.Background())
	a.countdownCancel = countdownCancel
	remaining := interval

	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-countdownCtx.Done():
				return
			case <-ticker.C:
				remaining--
				if remaining <= 0 {
					// Time to refresh
					view := a.CurrentView()
					if view != nil {
						view.Init(context.Background())
					}
					remaining = interval
				}
				a.tviewApp.QueueUpdateDraw(func() {
					a.statusBar.SetCountdown(remaining)
				})
			}
		}
	}()
}

func (a *App) stopAutoRefresh() {
	if a.refreshCancel != nil {
		a.refreshCancel()
		a.refreshCancel = nil
	}
	if a.countdownCancel != nil {
		a.countdownCancel()
		a.countdownCancel = nil
	}
	if a.refreshTicker != nil {
		a.refreshTicker.Stop()
		a.refreshTicker = nil
	}
	a.statusBar.SetCountdown(0)
}
