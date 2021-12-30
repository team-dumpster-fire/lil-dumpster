package main

import (
	"log"
	"strings"

	"github.com/bwmarrin/discordgo"
)

const helpText = `Hello there! I'm a happy helpful dumpster fire ready to assist you in this server. Check out the #roles channel for a message from me with some instructions! Have a wonderful day!
`

func handleReady(s *discordgo.Session, event *discordgo.Ready) {
	if err := s.UpdateListeningStatus("!lil-dumpster"); err != nil {
		log.Println("Failed to update listening status:", err)
	}

	for _, g := range event.Guilds {
		if err := manageRoles(s, g); err != nil {
			log.Println("Failed to watch guild:", err)
		}
	}
}

func handleCommands(s *discordgo.Session, m *discordgo.MessageCreate) {
	if !strings.HasPrefix(m.Content, "!lil-dumpster ") || m.Author.Bot {
		return
	}

	args := strings.Split(m.Content, " ")[1:]
	if len(args) == 0 {
		return
	}

	log.Println(args)

	switch args[0] {
	default:
		if _, err := s.ChannelMessageSend(m.ChannelID, helpText); err != nil {
			log.Println("Failed to send channel message:", err)
		}
	}
}

func handleMessageReactionAdd(s *discordgo.Session, event *discordgo.MessageReactionAdd) {
	for _, watchedChannel := range watchedChannels {
		if event.GuildID == watchedChannel.guild.ID &&
			event.ChannelID == watchedChannel.channel.ID &&
			event.MessageID == watchedChannel.message.ID {

			if err := addRoleToUser(s, watchedChannel, event.MessageReaction); err != nil {
				log.Println("Could not handle message reaction addition:", err)
			}
			break
		}
	}
}

func handleMessageReactionRemove(s *discordgo.Session, event *discordgo.MessageReactionRemove) {
	for _, watchedChannel := range watchedChannels {
		if event.GuildID == watchedChannel.guild.ID &&
			event.ChannelID == watchedChannel.channel.ID &&
			event.MessageID == watchedChannel.message.ID {

			if err := removeRoleFromUser(s, watchedChannel, event.MessageReaction); err != nil {
				log.Println("Could not handle message reaction removal:", err)
			}
			break
		}
	}
}
