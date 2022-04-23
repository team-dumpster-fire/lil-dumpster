package cmd

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/team-dumpster-fire/lil-dumpster/internal/state"
)

type poll struct {
	prompt  string
	choices []pollChoice
}

type pollChoice struct {
	choice   string
	count    int64
	mentions []string
}

var pollMutex sync.Mutex

func init() {
	fnRegisterCommands = append(fnRegisterCommands, func(store state.Backend) []applicationCommand {
		return []applicationCommand{
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
	})
}

func parsePoll(msg string) poll {
	// * Choice (0)
	// 1. Another Choice (1, @taiidani, @anyone)
	replacer := regexp.MustCompile(`\d\. (.+) \((\d)(.*)\)$`)

	// ["prompt", "choices"]
	msgParts := strings.SplitN(msg, "\n", 2)
	ret := poll{prompt: strings.TrimSpace(msgParts[0]), choices: []pollChoice{}}

	// ["* choice0", "* choice1", "* choice2"]
	for _, choice := range strings.Split(strings.TrimSpace(msgParts[1]), "\n") {
		// ["* choice (#count, user1, user2)", "choice", "#count", ", user1, user2"]
		lineParts := replacer.FindStringSubmatch(choice)
		if len(lineParts) < 4 {
			log.Println("Failed to parse choice: ", choice)
			continue
		}

		c := pollChoice{
			choice:   strings.TrimSpace(lineParts[1]),
			mentions: []string{},
		}

		// ["user1", "user2"]
		for _, mention := range strings.Split(lineParts[3], ", ") {
			mention = strings.TrimSpace(mention)
			if len(mention) == 0 {
				continue
			}
			c.mentions = append(c.mentions, mention)
		}

		count, err := strconv.ParseInt(lineParts[2], 10, 64)
		if err != nil {
			log.Println("Could not parse count: ", lineParts[2])
		}
		c.count = count

		ret.choices = append(ret.choices, c)
	}

	return ret
}

func (p *poll) serialize() string {
	msg := strings.Builder{}
	msg.WriteString(strings.TrimSpace(p.prompt) + "\n")

	for i, choice := range p.choices {
		var mentionsSeparator string
		if len(choice.mentions) > 0 {
			mentionsSeparator = ", "
		}

		str := fmt.Sprintf(
			"%d. %s (%d%s)\n",
			i+1,
			strings.TrimSpace(choice.choice),
			choice.count,
			mentionsSeparator+strings.Join(choice.mentions, mentionsSeparator),
		)
		msg.WriteString(str)
	}

	return msg.String()
}
