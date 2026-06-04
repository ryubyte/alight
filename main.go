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

	"github.com/ryubyte/codex-bar/internal/claudecfg"
	"github.com/ryubyte/codex-bar/internal/codexcfg"
	"github.com/ryubyte/codex-bar/internal/icons"
	"github.com/ryubyte/codex-bar/internal/server"
	"github.com/ryubyte/codex-bar/internal/state"
	"github.com/ryubyte/codex-bar/internal/ui"
)

const defaultStartPort = 9876

func main() {
	macos.RunApp(didLaunch)
}

func didLaunch(app appkit.Application, delegate *appkit.ApplicationDelegate) {
	app.SetActivationPolicy(appkit.ApplicationActivationPolicyAccessory)

	machine := state.NewMachine()

	// Auto-detect free port
	port, err := server.FindFreePort(defaultStartPort)
	if err != nil {
		log.Fatalf("find free port: %v", err)
	}
	serverAddr := fmt.Sprintf("localhost:%d", port)
	portStr := fmt.Sprintf("%d", port)

	// Inject hooks for Codex CLI
	injectCodexHooks(serverAddr)
	// Inject hooks for Claude Code
	injectClaudeHooks(portStr)

	// Setup cleanup on exit (no defer — didLaunch returns immediately)
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-sigCh
		cleanupAllHooks()
		app.Terminate(nil)
	}()

	// Start HTTP server
	srv := server.New(machine, serverAddr)
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
	btn.SetToolTip("Codex Bar ⚫ 空闲")

	// Blink controller
	blink := ui.NewBlinkController(btn, machine)

	// Build menu (also registers OnChange callback)
	menu := ui.BuildMenu(ui.MenuConfig{
		Machine:    machine,
		ServerAddr: serverAddr,
		Port:       port,
		Blink:      blink,
		OnQuit: func() {
			cleanupAllHooks()
			app.Terminate(nil)
		},
	})
	item.SetMenu(menu)
}

func injectCodexHooks(serverAddr string) {
	cfg, err := codexcfg.Read()
	if err != nil {
		log.Printf("warning: read codex config: %v", err)
	}
	cfg = codexcfg.Inject(cfg, serverAddr)
	if err := codexcfg.Write(cfg); err != nil {
		log.Printf("warning: write codex config: %v", err)
	}
}

func injectClaudeHooks(port string) {
	if err := claudecfg.Inject(port); err != nil {
		log.Printf("warning: inject claude hooks: %v", err)
	}
}

func cleanupAllHooks() {
	// Cleanup Codex CLI hooks
	cfg, err := codexcfg.Read()
	if err != nil {
		log.Printf("warning: read codex config for cleanup: %v", err)
	} else {
		cfg = codexcfg.Cleanup(cfg)
		if err := codexcfg.Write(cfg); err != nil {
			log.Printf("warning: write codex config for cleanup: %v", err)
		}
	}

	// Cleanup Claude Code hooks
	if err := claudecfg.Cleanup(); err != nil {
		log.Printf("warning: cleanup claude hooks: %v", err)
	}
}
