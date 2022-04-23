package cmd

import (
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/team-dumpster-fire/lil-dumpster/internal/state"
)

var errUnauthorizedRole = errors.New("not authorized to manage this role")

func init() {
	fnRegisterCommands = append(fnRegisterCommands, func(store state.Backend) []applicationCommand {
		return []applicationCommand{
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
	})
}

func manageRoles(s *discordgo.Session, guild *discordgo.Guild) error {
	const rolesChannelName = "roles"

	rolesChannel, err := findChannel(s, guild.ID, rolesChannelName)
	if err != nil {
		return fmt.Errorf("could not find %s channel: %w", rolesChannelName, err)
	}

	messages, err := s.ChannelMessagesPinned(rolesChannel.ID)
	if err != nil {
		return fmt.Errorf("could not find pinned channel messages: %w", err)
	}

	// Filter to only messages by the bot
	var activeMessage *discordgo.Message
	for _, m := range messages {
		if m.Author.ID != s.State.User.ID {
			continue
		}

		activeMessage = m
	}

	messageText := buildRolesMessage()
	if activeMessage == nil {
		// No messages found! Post and pin an initial message
		activeMessage, err = s.ChannelMessageSend(rolesChannel.ID, messageText)
		if err != nil {
			return fmt.Errorf("could not post a new channel message: %w", err)
		}

		if err := s.ChannelMessagePin(rolesChannel.ID, activeMessage.ID); err != nil {
			return fmt.Errorf("could not pin the new channel message: %w", err)
		}
	}

	if activeMessage.Content != messageText {
		// Update the message text to match expected
		_, err = s.ChannelMessageEdit(rolesChannel.ID, activeMessage.ID, messageText)
		if err != nil {
			return fmt.Errorf("could not update the message text: %w", err)
		}
	}

	return nil
}

func addRoleToUser(s *discordgo.Session, i *discordgo.Interaction, roleName string) error {
	role, err := findRoleForName(s, i.GuildID, roleName)
	if err != nil {
		return fmt.Errorf("could not find role: %w", err)
	}

	if i.Member == nil {
		return fmt.Errorf("user not set. Have you sent this command from within a server channel?")
	}

	fmt.Printf("Adding role %s to user %v in guild %s\n", role.Name, i.Member.User.Username, i.GuildID)
	err = s.GuildMemberRoleAdd(i.GuildID, i.Member.User.ID, role.ID)
	if err != nil && strings.Contains(err.Error(), "50013") {
		return errUnauthorizedRole
	}
	return err
}

func removeRoleFromUser(s *discordgo.Session, i *discordgo.Interaction, roleName string) error {
	role, err := findRoleForName(s, i.GuildID, roleName)
	if err != nil {
		return fmt.Errorf("could not find role: %w", err)
	}

	if i.Member == nil {
		return fmt.Errorf("user not set. Have you sent this command from within a server channel?")
	}

	fmt.Printf("Removing role %s from user %s in guild %s\n", role.Name, i.Member.User.Username, i.GuildID)
	err = s.GuildMemberRoleRemove(i.GuildID, i.Member.User.ID, role.ID)
	if err != nil && strings.Contains(err.Error(), "50013") {
		return errUnauthorizedRole
	}
	return err
}

func findRoleForName(s *discordgo.Session, guildID string, name string) (*discordgo.Role, error) {
	roles, err := s.GuildRoles(guildID)
	if err != nil {
		return nil, fmt.Errorf("could not enumerate guild roles: %w", err)
	}

	for _, role := range roles {
		if strings.EqualFold(role.Name, name) {
			return role, nil
		}
	}

	return nil, fmt.Errorf("could not find a guild role for the configured role '%s'", name)
}

func findChannel(s *discordgo.Session, guildID, channelName string) (*discordgo.Channel, error) {
	channels, err := s.GuildChannels(guildID)
	if err != nil {
		return nil, fmt.Errorf("could not enumerate guild channels: %w", err)
	}

	for _, channel := range channels {
		if channel.Name == channelName {
			return channel, nil
		}
	}

	return nil, fmt.Errorf("couldn't find a channel called '" + channelName + "'")
}

func buildRolesMessage() string {
	return `Hello! I've registered /slash commands in this server for managing user roles. Please use the /role-add and /role-remove commands to manage your roles.

If you'd like to see more roles in this server, send a message to one of the helpful server administrators and they can help out. Have a wonderful day!
	`
}
