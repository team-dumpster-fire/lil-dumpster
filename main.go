package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/go-redis/redis/v8"
	"github.com/team-dumpster-fire/lil-dumpster/cmd"
	"github.com/team-dumpster-fire/lil-dumpster/internal/state"
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

	commands := cmd.NewCommands(b, configureBackend(ctx))
	commands.AddHandlers()

	// Begin listening for events
	err = b.Open()
	if err != nil {
		log.Fatal("Could not connect to discord", err)
	}

	// Wait until the application is shutting down
	fmt.Println("Bot is now running. Check out Discord!")
	<-ctx.Done()
	b.Close()
}

func configureBackend(ctx context.Context) state.Backend {
	var store state.Backend = state.NewMemory()
	if host, ok := os.LookupEnv("REDIS_HOST"); ok {
		var port string
		if host, port, ok = strings.Cut(host, ":"); !ok {
			if port, ok = os.LookupEnv("REDIS_PORT"); !ok {
				port = "4646"
			}
		}

		addr := fmt.Sprintf("%s:%s", host, port)
		store = state.NewRedis(&redis.Options{Addr: addr})
		if err := store.Set(ctx, "client", "lil-dumpster"); err != nil {
			log.Fatalf("Unable to connect to Redis backend at %s: %s", addr, err)
		}
	}

	return store
}
