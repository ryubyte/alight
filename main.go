package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/progrium/darwinkit/macos"
	"github.com/progrium/darwinkit/macos/appkit"
	"github.com/progrium/darwinkit/objc"

	"github.com/ryubyte/codex-bar/internal/adapter/claude"
	"github.com/ryubyte/codex-bar/internal/adapter/codex"
	"github.com/ryubyte/codex-bar/internal/core"
	corestate "github.com/ryubyte/codex-bar/internal/core/state"
	"github.com/ryubyte/codex-bar/internal/icons"
	"github.com/ryubyte/codex-bar/internal/ui"
)

const defaultStartPort = 9876

func main() {
	macos.RunApp(didLaunch)
}

func didLaunch(app appkit.Application, delegate *appkit.ApplicationDelegate) {
	app.SetActivationPolicy(appkit.ApplicationActivationPolicyAccessory)

	registry := core.NewRegistry()
	registry.Register(codex.New())
	registry.Register(claude.New())

	machine := corestate.NewMachine()

	// Auto-detect free port
	port, err := core.FindFreePort(defaultStartPort)
	if err != nil {
		log.Fatalf("find free port: %v", err)
	}
	serverAddr := fmt.Sprintf("localhost:%d", port)
	portStr := fmt.Sprintf("%d", port)

	// Inject hooks for all registered adapters
	registry.InjectAll(portStr)

	// Setup cleanup on exit (no defer — didLaunch returns immediately)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigCh
		registry.CleanupAll()
		app.Terminate(nil)
	}()

	// Start HTTP server
	srv := core.NewServer(machine, registry, serverAddr)
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Printf("server error: %v", err)
		}
	}()

	// Status bar item
	item := appkit.StatusBar_SystemStatusBar().StatusItemWithLength(appkit.VariableStatusItemLength)
	objc.Retain(&item)
	btn := item.Button()
	btn.SetImage(icons.ForStatus(machine.Current()))
	btn.SetToolTip("Codex Bar idle")

	// Blink controller
	blink := ui.NewBlinkController(btn, machine)

	// Build menu
	menu := ui.BuildMenu(ui.MenuConfig{
		Machine: machine,
		Blink:   blink,
		OnQuit: func() {
			registry.CleanupAll()
			app.Terminate(nil)
		},
	})
	item.SetMenu(menu)
}
