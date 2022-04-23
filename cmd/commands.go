package cmd

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type applicationCommand struct {
	Command           *discordgo.ApplicationCommand
	Autocomplete      func(s *discordgo.Session, i *discordgo.InteractionCreate, o *discordgo.ApplicationCommandInteractionDataOption) []*discordgo.ApplicationCommandOptionChoice
	MessageComponents map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate)
	Handler           func(s *discordgo.Session, i *discordgo.InteractionCreate)
}

var applicationCommands = []applicationCommand{
	{
		Command: &discordgo.ApplicationCommand{
			Name:        "role-add",
			Description: "Adds a role to your user",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:         discordgo.ApplicationCommandOptionString,
					Name:         "role-name",
					Description:  "Name of the role to be added",
					Required:     true,
					Autocomplete: true,
				},
			},
		},
		Autocomplete: func(s *discordgo.Session, i *discordgo.InteractionCreate, o *discordgo.ApplicationCommandInteractionDataOption) []*discordgo.ApplicationCommandOptionChoice {
			ret := []*discordgo.ApplicationCommandOptionChoice{}

			switch o.Name {
			case "role-name":
				roles, err := s.GuildRoles(i.GuildID)
				if err != nil {
					return ret
				}

				for _, role := range roles {
					add := func() bool {
						// @ roles also show up here, skip 'em
						if strings.HasPrefix(role.Name, "@") {
							return false
						}

						// Skip roles already present on the user
						if i.Member != nil {
							for _, existRole := range i.Member.Roles {
								if existRole == role.ID {
									return false
								}
							}
						}

						// Check that the role name starts with the already-entered text
						return strings.HasPrefix(strings.ToLower(role.Name), strings.ToLower(o.StringValue()))
					}()

					if add {
						ret = append(ret, &discordgo.ApplicationCommandOptionChoice{Name: role.Name, Value: role.Name})
					}
				}
			}

			return ret
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			roleName := i.ApplicationCommandData().Options[0].StringValue()
			err := addRoleToUser(s, i.Interaction, roleName)
			if err != nil {
				log.Println("Could not handle role addition:", err)
				commandError(s, i.Interaction, err)
				return
			}

			err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("The %q role has been added to your user", roleName),
					Flags:   1 << 6, // Ephemeral, private
				},
			})
			if err != nil {
				log.Println("Could not respond to user message:", err)
				commandError(s, i.Interaction, err)
				return
			}
		},
	},
	{
		Command: &discordgo.ApplicationCommand{
			Name:        "role-remove",
			Description: "Removes a role from your user",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:         discordgo.ApplicationCommandOptionString,
					Name:         "role-name",
					Description:  "Name of the role to be removed",
					Required:     true,
					Autocomplete: true,
				},
			},
		},
		Autocomplete: func(s *discordgo.Session, i *discordgo.InteractionCreate, o *discordgo.ApplicationCommandInteractionDataOption) []*discordgo.ApplicationCommandOptionChoice {
			ret := []*discordgo.ApplicationCommandOptionChoice{}

			switch o.Name {
			case "role-name":
				roles, err := s.GuildRoles(i.GuildID)
				if err != nil {
					return ret
				}

				for _, roleID := range i.Member.Roles {
					var role *discordgo.Role
					for i := range roles {
						if roles[i].ID == roleID {
							role = roles[i]
						}
					}
					if role == nil {
						continue
					}

					add := func() bool {
						// @ roles also show up here, skip 'em
						if strings.HasPrefix(role.Name, "@") {
							return false
						}

						// Check that the role name starts with the already-entered text
						return strings.HasPrefix(strings.ToLower(role.Name), strings.ToLower(o.StringValue()))
					}()

					if add {
						ret = append(ret, &discordgo.ApplicationCommandOptionChoice{Name: role.Name, Value: role.Name})
					}
				}
			}

			return ret
		},
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			roleName := i.ApplicationCommandData().Options[0].StringValue()
			err := removeRoleFromUser(s, i.Interaction, roleName)
			if err != nil {
				log.Println("Could not handle role removal:", err)
				commandError(s, i.Interaction, err)
				return
			}

			err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf("The %q role has been removed from your user", roleName),
					Flags:   1 << 6, // Ephemeral, private
				},
			})
			if err != nil {
				log.Println("Could not respond to user message:", err)
				commandError(s, i.Interaction, err)
				return
			}
		},
	},
	{
		Command: &discordgo.ApplicationCommand{
			Name:        "poll",
			Description: "Submit a poll to the channel",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "choices",
					Description: "Comma-separated list of choices for presenting to users",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "prompt",
					Description: "Question to ask users",
					Required:    false,
				},
				{
					Type:        discordgo.ApplicationCommandOptionBoolean,
					Name:        "draft",
					Description: "If true, will only display the poll to you so that you may review the output",
					Required:    false,
				},
			},
		},
		MessageComponents: func() map[string]func(*discordgo.Session, *discordgo.InteractionCreate) {
			ret := map[string]func(*discordgo.Session, *discordgo.InteractionCreate){}
			for i := 0; i < 20; i++ {
				choiceN := i
				customID := fmt.Sprintf("pollButton%d", i)
				ret[customID] = func(s *discordgo.Session, interaction *discordgo.InteractionCreate) {
					pollMutex.Lock()
					defer pollMutex.Unlock()

					log.Println("Button clicked: ", interaction.Message.ID, interaction.Member.User.Username)
					poll := parsePoll(interaction.Message.Content)

					// Build the new user list, skipping the actioning user
					newUsers := []string{}
					for _, user := range poll.choices[choiceN].mentions {
						if user == interaction.Member.Mention() {
							continue
						}
						newUsers = append(newUsers, user)
					}

					if len(poll.choices[choiceN].mentions) != len(newUsers) {
						// If the user voted already (the list is N-1), decrement the count
						poll.choices[choiceN].count--
					} else {
						// If the user hasn't voted (N), increment the count and add the user
						poll.choices[choiceN].count++
						newUsers = append(newUsers, interaction.Member.Mention())
					}
					poll.choices[choiceN].mentions = newUsers

					// Log the new poll string
					log.Println("New poll: ", poll)

					// And update the string on the server
					s.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseUpdateMessage,
						Data: &discordgo.InteractionResponseData{
							Content:    poll.serialize(),
							Flags:      uint64(interaction.Message.Flags),
							Components: interaction.Message.Components,
						},
					})
				}
			}

			return ret
		}(),
		Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			poll := poll{prompt: "Poll:"}
			var choicesString string
			var flags uint64
			for _, opt := range i.ApplicationCommandData().Options {
				switch opt.Name {
				case "choices":
					choicesString = opt.StringValue()
				case "prompt":
					poll.prompt = strings.TrimSpace(opt.StringValue())
				case "draft":
					if opt.BoolValue() {
						flags = 1 << 6 // Ephemeral, private
					}
				}
			}

			// Build the buttons
			buttons := []discordgo.MessageComponent{}
			index := 0
			for _, choice := range strings.Split(choicesString, ",") {
				choice = strings.TrimSpace(choice)
				if len(choice) == 0 {
					continue
				}

				poll.choices = append(poll.choices, pollChoice{
					choice:   choice,
					count:    0,
					mentions: []string{},
				})

				buttons = append(buttons, discordgo.Button{
					CustomID: fmt.Sprintf("pollButton%d", index),
					Label:    fmt.Sprintf("%d", index+1),
				})
				index++
			}

			// Send the poll!
			err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: poll.serialize(),
					Flags:   flags,
					Components: []discordgo.MessageComponent{
						discordgo.ActionsRow{Components: buttons},
					},
				},
			})
			if err != nil {
				log.Println("Could not respond to user message:", err)
				commandError(s, i.Interaction, err)
				return
			}
		},
	},
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
