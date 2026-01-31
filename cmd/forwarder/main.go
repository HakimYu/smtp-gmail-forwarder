package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/HakimYu/smtp-gmail-forwarder/internal/config"
	"github.com/HakimYu/smtp-gmail-forwarder/internal/gmail"
	smtpserver "github.com/HakimYu/smtp-gmail-forwarder/internal/smtp"
	"github.com/emersion/go-smtp"
)

func main() {
	configFile := flag.String("config", "config.yaml", "path to configuration file")
	flag.Parse()

	cfg, err := config.Load(*configFile)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	ctx := context.Background()

	gmailClient, err := gmail.NewClient(ctx, cfg.Gmail.CredentialsFile, cfg.Gmail.TokenFile)
	if err != nil {
		log.Fatalf("Failed to create Gmail client: %v", err)
	}

	handler := smtpserver.NewForwarderHandler(gmailClient)
	backend := smtpserver.NewBackend(handler)

	server := smtp.NewServer(backend)
	server.Addr = fmt.Sprintf("%s:%d", cfg.SMTP.Host, cfg.SMTP.Port)
	server.Domain = "localhost"
	server.ReadTimeout = 10 * time.Second
	server.WriteTimeout = 10 * time.Second
	server.MaxMessageBytes = 10 * 1024 * 1024
	server.MaxRecipients = 50
	server.AllowInsecureAuth = true

	log.Printf("Starting SMTP server on %s", server.Addr)

	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Fatalf("SMTP server error: %v", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	log.Println("Shutting down server...")
	if err := server.Close(); err != nil {
		log.Printf("Error closing server: %v", err)
	}
}
