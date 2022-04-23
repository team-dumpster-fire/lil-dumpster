package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
)

func main() {
	// Handle signal interrupts.
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	defer cancel()

	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		log.Fatal("Please set a DISCORD_TOKEN environment variable to your bot token")
	}

	b, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal(err)
	}
	defer b.Close()

	// Register handlers
	b.AddHandler(handleReady)
	b.AddHandler(handleCommand)

	// Begin listening for events
	err = b.Open()
	if err != nil {
		log.Fatal("Could not connect to discord", err)
	}

	// Wait until the application is shutting down
	log.Print("Bot is now running. Check out Discord!")
	<-ctx.Done()
	b.Close()
}
