package main

import (
	"flag"
	"log"
	"os"
	"strconv"

	"github.com/skywave-chat/skywave/pkg/server"
)

func main() {
	defaultPort := 8080
	if envPort := os.Getenv("PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil {
			defaultPort = p
		}
	}

	port := flag.Int("port", defaultPort, "Port to listen on (default 8080 or $PORT)")
	host := flag.String("host", "0.0.0.0", "Host address to bind")
	name := flag.String("name", "Skywave Prime Node", "Display name of this server node")
	password := flag.String("password", "", "Set a server password to make this instance private")
	motd := flag.String("motd", "Welcome to Skywave Prime. Anonymous, encrypted, blazing fast.", "Server Message of the Day")
	history := flag.Int("history", 50, "Max messages stored in RAM per room (0 for zero-log ephemeral mode)")
	flag.Parse()

	cfg := server.Config{
		Host:           *host,
		Port:           *port,
		ServerName:     *name,
		ServerPassword: *password,
		MOTD:           *motd,
		MaxHistory:     *history,
	}

	srv := server.NewServer(cfg)

	if err := srv.Start(); err != nil {
		log.Fatalf("Server shutdown with error: %v", err)
	}
}
