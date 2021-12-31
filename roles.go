package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

var errUnauthorizedRole = errors.New("not authorized to manage this role")

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
