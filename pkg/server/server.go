package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// Version is the node protocol version.
const Version = "2.0.0"

// Config defines server settings.
type Config struct {
	Host           string
	Port           int
	ServerName     string
	ServerPassword string
	MOTD           string
	MaxHistory     int
}

// Server wraps HTTP server and WebSocket Hub.
type Server struct {
	cfg     Config
	hub     *Hub
	started time.Time
}

// NewServer initializes a new Skywave server instance.
func NewServer(cfg Config) *Server {
	if cfg.ServerName == "" {
		cfg.ServerName = "Skywave Prime Node"
	}
	if cfg.MOTD == "" {
		cfg.MOTD = "Welcome to Skywave Prime. Anonymous, encrypted, blazing fast."
	}
	if cfg.MaxHistory <= 0 {
		cfg.MaxHistory = 50
	}

	hub := NewHub(cfg.ServerName, cfg.ServerPassword, cfg.MOTD, cfg.MaxHistory)
	return &Server{
		cfg:     cfg,
		hub:     hub,
		started: time.Now(),
	}
}

// Hub exposes the hub for health checks and tests.
func (s *Server) Hub() *Hub { return s.hub }

// Start boots the server listener and hub event loop.
func (s *Server) Start() error {
	go s.hub.Run()

	// Periodic presence heartbeat in logs (premium observability).
	go func() {
		t := time.NewTicker(5 * time.Minute)
		defer t.Stop()
		for range t.C {
			c, r, m := s.hub.Stats()
			log.Printf("[telemetry] clients=%d rooms=%d memberships=%d uptime=%s", c, r, m, time.Since(s.started).Round(time.Second))
		}
	}()

	mux := http.NewServeMux()

	// WebSocket chat endpoint.
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("[warn] websocket upgrade failed from %s: %v", r.RemoteAddr, err)
			return
		}

		client := NewClient(s.hub, conn)
		s.hub.Register(client)

		go client.WritePump()
		go client.ReadPump()
	})

	// Premium health + discovery endpoint.
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		clients, rooms, members := s.hub.Stats()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"status":       "ok",
			"product":      "skywave",
			"version":      Version,
			"server_name":  s.cfg.ServerName,
			"motd":         s.cfg.MOTD,
			"clients":      clients,
			"rooms":        rooms,
			"memberships":  members,
			"uptime_secs":  int64(time.Since(s.started).Seconds()),
			"private_node": s.cfg.ServerPassword != "",
			"time":         time.Now().Unix(),
		})
	})

	// Human-friendly landing page for anyone opening the node in a browser.
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintf(w, "skywave %s — %s\nconnect: ws://%s/ws\nhealth: /health\n", Version, s.cfg.ServerName, r.Host)
	})

	addr := fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
	httpServer := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
	}

	// Graceful shutdown handling.
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		printBanner(s.cfg, addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[fatal] server error: %v", err)
		}
	}()

	<-stopChan
	log.Println("\n[shutdown] draining connections gracefully...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return httpServer.Shutdown(ctx)
}

func printBanner(cfg Config, addr string) {
	privacy := "open relay"
	if cfg.ServerPassword != "" {
		privacy = "private (password locked)"
	}
	fmt.Println()
	fmt.Println("  ◈ ─ [ S I G N A L  C A R R I E R ] ─ ◈")
	fmt.Println("   ▄██████  █   █  █   █  █     █  ▄█████  █   █  ██████")
	fmt.Println("  ░██       █ ▄█▀   ▀█▄█▀  █ ▄ █ █ ░██  ██ █   █ ░██    ")
	fmt.Println("  ░██████   ███▀     ███   ███████ ░██████ █   █ ░█████ ")
	fmt.Println("       ██   █ ▀█▄    ███   ██ ▀ ██ ░██  ██ ░█ █░ ░██    ")
	fmt.Println("  ██████▀   █   █    ███   █     █ ░██  ██  ░█░  ░██████")
	fmt.Println("  E N C R Y P T E D   T E R M I N A L   R E L A Y   N O D E")
	fmt.Println(" ────────────────────────────────────────────────────────────")
	fmt.Printf("  ◈ NODE ID    : %s (v%s)\n", cfg.ServerName, Version)
	fmt.Printf("  ◈ WEBSOCKET  : ws://%s/ws\n", addr)
	fmt.Printf("  ◈ TELEMETRY  : http://%s/health\n", addr)
	fmt.Printf("  ◈ CARRIER    : %s · max_history=%d · motd=%q\n", privacy, cfg.MaxHistory, cfg.MOTD)
	fmt.Println(" ────────────────────────────────────────────────────────────")
	fmt.Println("  ● Awaiting incoming carrier connections...")
	fmt.Println()
}
