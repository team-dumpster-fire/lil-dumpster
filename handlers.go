package main

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

func handleReady(s *discordgo.Session, event *discordgo.Ready) {
	for _, g := range event.Guilds {
		if err := manageRoles(s, g); err != nil {
			log.Println("Failed to watch guild:", err)
		}

		for _, cmd := range applicationCommands {
			if _, err := s.ApplicationCommandCreate(s.State.User.ID, g.ID, cmd.Command); err != nil {
				log.Printf("Unable to set application command %q: %s", cmd.Command.Name, err)
			}
		}
	}
}

func handleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	for _, cmd := range applicationCommands {
		if cmd.Command.Name == i.ApplicationCommandData().Name {
			switch i.Type {
			case discordgo.InteractionApplicationCommandAutocomplete:
				if cmd.Autocomplete == nil {
					return
				}

				for _, opt := range i.ApplicationCommandData().Options {
					if opt.Focused {
						choices := cmd.Autocomplete(s, i, opt)
						_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
							Type: discordgo.InteractionApplicationCommandAutocompleteResult,
							Data: &discordgo.InteractionResponseData{Choices: choices},
						})
					}
				}
			default:
				cmd.Handler(s, i)
			}
		}
	}
}
