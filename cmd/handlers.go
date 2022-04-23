package cmd

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

func HandleReady(s *discordgo.Session, event *discordgo.Ready) {
	for _, g := range event.Guilds {
		if err := manageRoles(s, g); err != nil {
			log.Println("Failed to watch guild:", err)
		}

		for _, cmd := range applicationCommands {
			log.Printf("Registering application command %q for bot user %q in guild %q", cmd.Command.Name, s.State.User.ID, g.ID)
			if _, err := s.ApplicationCommandCreate(s.State.User.ID, g.ID, cmd.Command); err != nil {
				log.Printf("Unable to set application command %q: %s", cmd.Command.Name, err)
			}
		}
	}
}

func HandleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	for _, cmd := range applicationCommands {
		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			if cmd.Command.Name == i.ApplicationCommandData().Name {
				cmd.Handler(s, i)
			}
		case discordgo.InteractionApplicationCommandAutocomplete:
			if cmd.Autocomplete != nil && cmd.Command.Name == i.ApplicationCommandData().Name {
				for _, opt := range i.ApplicationCommandData().Options {
					if opt.Focused {
						choices := cmd.Autocomplete(s, i, opt)
						_ = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
							Type: discordgo.InteractionApplicationCommandAutocompleteResult,
							Data: &discordgo.InteractionResponseData{Choices: choices},
						})
					}
				}
			}
		case discordgo.InteractionMessageComponent:
			if cmd.MessageComponents != nil {
				log.Println(i.MessageComponentData().CustomID)
				for customID, fn := range cmd.MessageComponents {
					if customID == i.MessageComponentData().CustomID {
						fn(s, i)
					}
				}
			}
		default:
			log.Println("Unknown interaction type encountered: ", i.Type)
		}
	}
}
