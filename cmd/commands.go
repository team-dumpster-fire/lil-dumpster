package cmd

import (
	"fmt"
	"log"

	"github.com/bwmarrin/discordgo"
	"github.com/team-dumpster-fire/lil-dumpster/internal/state"
)

type applicationCommand struct {
	Command           *discordgo.ApplicationCommand
	Autocomplete      func(s *discordgo.Session, i *discordgo.InteractionCreate, o *discordgo.ApplicationCommandInteractionDataOption) []*discordgo.ApplicationCommandOptionChoice
	MessageComponents map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate)
	Handler           func(s *discordgo.Session, i *discordgo.InteractionCreate)
	store             state.Backend
}

type commandRegistration func(store state.Backend) []applicationCommand

var fnRegisterCommands = []commandRegistration{}

type Commands struct {
	commands []applicationCommand
	s        *discordgo.Session
}

func NewCommands(session *discordgo.Session, store state.Backend) *Commands {
	ret := Commands{
		commands: []applicationCommand{},
		s:        session,
	}

	for _, fn := range fnRegisterCommands {
		ret.commands = append(ret.commands, fn(store)...)
	}

	return &ret
}

func (c *Commands) AddHandlers() {
	c.s.AddHandler(c.handleReady)
	c.s.AddHandler(c.handleCommand)
}

func (c *Commands) handleReady(s *discordgo.Session, event *discordgo.Ready) {
	for _, g := range event.Guilds {
		if err := manageRoles(s, g); err != nil {
			log.Println("Failed to watch guild:", err)
		}

		for _, cmd := range c.commands {
			log.Printf("Registering application command %q for bot user %q in guild %q", cmd.Command.Name, s.State.User.ID, g.ID)
			if _, err := s.ApplicationCommandCreate(s.State.User.ID, g.ID, cmd.Command); err != nil {
				log.Printf("Unable to set application command %q: %s", cmd.Command.Name, err)
			}
		}
	}
}

func (c *Commands) handleCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	for _, cmd := range c.commands {
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

func commandError(s *discordgo.Session, i *discordgo.Interaction, message error) {
	_ = s.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf(":warning: %s", message),
			Flags:   1 << 6, // Ephemeral, private
		},
	})
}
