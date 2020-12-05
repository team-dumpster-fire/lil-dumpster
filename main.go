package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/net/context"
)

func main() {
	// Handle signal interrupts.
	var cancel func()
	ctx, cancel := context.WithCancel(context.Background())
	go signals(cancel)

	token := os.Getenv("DISCORD_TOKEN")
	if token == "" {
		panic("Please set a DISCORD_TOKEN environment variable to your bot token")
	}

	b, err := discordgo.New("Bot " + token)
	if err != nil {
		panic(err)
	}
	defer b.Close()

	// Register handlers
	b.AddHandler(handleReady)
	b.AddHandler(handleCommands)
	b.AddHandler(handleMessageReactionAdd)
	b.AddHandler(handleMessageReactionRemove)

	// Begin listening for events
	err = b.Open()
	if err != nil {
		log.Panic("Could not connect to discord", err)
	}
	log.Print("Bot is now running. Check out Discord!")

	// Wait until the application is shutting down
	<-ctx.Done()
}

func signals(cancel func()) {
	defer cancel()
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sig
}
