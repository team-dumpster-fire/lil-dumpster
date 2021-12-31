package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type applicationCommand struct {
	Command      *discordgo.ApplicationCommand
	Autocomplete func(s *discordgo.Session, i *discordgo.InteractionCreate, o *discordgo.ApplicationCommandInteractionDataOption) []*discordgo.ApplicationCommandOptionChoice
	Handler      func(s *discordgo.Session, i *discordgo.InteractionCreate)
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
