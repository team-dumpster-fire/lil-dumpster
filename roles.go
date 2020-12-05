package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/bwmarrin/discordgo"
)

type (
	role struct {
		Name string
		Game string
	}

	watchedChannel struct {
		guild   *discordgo.Guild
		channel *discordgo.Channel
		message *discordgo.Message
	}
)

var emojiRoleMap = map[string]role{
	"among_us":       {Name: "Imposter", Game: "Among Us"},
	"borderlands":    {Name: "Vault Hunter", Game: "Borderlands"},
	"destiny":        {Name: "Guardian", Game: "Destiny"},
	"jackbox":        {Name: "Jackal", Game: "Jackbox Party Pack"},
	"sea_of_thieves": {Name: "Pirate", Game: "Sea of Thieves"},
	"speedrunners":   {Name: "Speed Runner", Game: "SpeedRunners"},
}

var watchedChannels []watchedChannel

func manageRoles(s *discordgo.Session, guild *discordgo.Guild) error {
	const watchedChannelName = "roles"

	rolesChannel, err := findChannel(s, guild.ID, watchedChannelName)
	if err != nil {
		return fmt.Errorf("could not find %s channel: %w", watchedChannelName, err)
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
		activeMessage, err = s.ChannelMessageEdit(rolesChannel.ID, activeMessage.ID, messageText)
		if err != nil {
			return fmt.Errorf("could not update the message text: %w", err)
		}
	}

	watchedChannels = append(watchedChannels, watchedChannel{
		guild:   guild,
		channel: rolesChannel,
		message: activeMessage,
	})
	return nil
}

func addRoleToUser(s *discordgo.Session, watch watchedChannel, reaction *discordgo.MessageReaction) error {
	user, err := s.User(reaction.UserID)
	if err != nil {
		return fmt.Errorf("could not find user record that added reaction: %w", err)
	}

	role, err := findRoleForReaction(s, watch.guild.ID, reaction)
	if err != nil {
		return fmt.Errorf("could not find role for reaction: %w", err)
	}

	fmt.Printf("Adding role %s to user %s in guild %s\n", role.Name, user.Username, watch.guild.Name)
	return s.GuildMemberRoleAdd(watch.guild.ID, user.ID, role.ID)
}

func removeRoleFromUser(s *discordgo.Session, watch watchedChannel, reaction *discordgo.MessageReaction) error {
	user, err := s.User(reaction.UserID)
	if err != nil {
		return fmt.Errorf("could not find user record that added reaction: %w", err)
	}

	role, err := findRoleForReaction(s, watch.guild.ID, reaction)
	if err != nil {
		return fmt.Errorf("could not find role for reaction: %w", err)
	}

	fmt.Printf("Removing role %s from user %s in guild %s\n", role.Name, user.Username, watch.guild.Name)
	return s.GuildMemberRoleRemove(watch.guild.ID, user.ID, role.ID)
}

func findRoleForReaction(s *discordgo.Session, guildID string, reaction *discordgo.MessageReaction) (*discordgo.Role, error) {
	roleConfig, ok := emojiRoleMap[reaction.Emoji.Name]
	if !ok {
		return nil, fmt.Errorf("could not map given reaction '%s' to a role", reaction.Emoji.Name)
	}

	roles, err := s.GuildRoles(guildID)
	if err != nil {
		return nil, fmt.Errorf("could not enumerate guild roles: %w", err)
	}

	for _, role := range roles {
		if role.Name == roleConfig.Name {
			return role, nil
		}
	}

	return nil, fmt.Errorf("could not find a guild role for the configured role '%s'", roleConfig.Name)
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
	const roleMessageText = `Hello! I'm watching this message for reactions, and will add or remove you from channel roles based on how you react. This lets you indicate your preferences in this server for the content you want to see!

I support the following reactions:
%s

If you'd like to see more roles, send a message to one of the helpful server administrators and they can help out. Have a wonderful day!
`

	var roleList []string

	// :among_us: - Among Us (Imposter role)
	// :destiny: - Destiny / Destiny 2 (Guardian role)
	// :pirate_flag: - Sea of Thieves (Pirate role)
	for emojiName, role := range emojiRoleMap {
		roleList = append(roleList, fmt.Sprintf(":%s: - %s (%s role)", emojiName, role.Game, role.Name))
	}
	sort.Strings(roleList)

	return fmt.Sprintf(roleMessageText, strings.Join(roleList, "\n"))
}
