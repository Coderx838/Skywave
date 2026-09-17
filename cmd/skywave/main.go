package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/skywave-chat/skywave/pkg/client"
	"github.com/skywave-chat/skywave/pkg/identity"
	"github.com/skywave-chat/skywave/pkg/ui"
)

func main() {
	serverURL := flag.String("server", "ws://localhost:8080/ws", "Skywave server address (ws://host:port/ws)")
	nickname := flag.String("nick", "", "Display name (saved to your anonymous account)")
	password := flag.String("password", "", "Server password (for private nodes)")
	reset := flag.Bool("reset-account", false, "Burn this machine's anonymous account and create a fresh one")
	flag.Parse()

	if *reset {
		_ = identity.Reset()
		fmt.Println("~ skywave: old identity burned. a new ghost will be born on next launch.")
	}

	// Anonymous account: auto-created on first run, persisted locally.
	// No email, no phone, no signup form. Just a stable ghost ID + a name you can change.
	id, isNew, err := identity.LoadOrCreate(*nickname)
	if err != nil {
		fmt.Fprintf(os.Stderr, "skywave: identity error: %v\n", err)
		os.Exit(1)
	}

	savedNick := id.Nickname
	if identity.IsGeneratedNick(savedNick) {
		savedNick = ""
	}
	if *nickname != "" {
		savedNick = *nickname
	}

	// If no user callsign has been confirmed yet, client does not broadcast auth until chosen in StageNamePrompt
	cli := client.NewClientWithAccount(*serverURL, savedNick, id.AccountID, *password)
	// Persist renames confirmed by the server.
	cli.OnNickChanged = func(_, newNick string) {
		id.Nickname = newNick
		_ = id.Save()
	}

	if err := cli.Connect(); err != nil {
		fmt.Fprintf(os.Stderr, "skywave: could not reach %s: %v\n", *serverURL, err)
		fmt.Fprintln(os.Stderr, "hint: start a node with  skywave-server  or check -server ws://<host>:<port>/ws")
		os.Exit(1)
	}

	model := ui.NewModel(cli, id.AccountID, isNew, savedNick)
	program := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := program.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "skywave: ui error: %v\n", err)
		os.Exit(1)
	}
}
