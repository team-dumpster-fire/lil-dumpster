package cmd

import "github.com/bwmarrin/discordgo"

type mockDiscordSession struct {
	mockUser               func(userID string) (st *discordgo.User, err error)
	mockInteractionRespond func(interaction *discordgo.Interaction, resp *discordgo.InteractionResponse) error
}

func (m *mockDiscordSession) User(userID string) (st *discordgo.User, err error) {
	return m.mockUser(userID)
}

func (m *mockDiscordSession) InteractionRespond(interaction *discordgo.Interaction, resp *discordgo.InteractionResponse) error {
	return m.mockInteractionRespond(interaction, resp)
}
