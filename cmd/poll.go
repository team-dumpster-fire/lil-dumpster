package cmd

import (
	"fmt"
	"log"
	"math/rand"
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
	count    int
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

					// first the tiebreaker button
					ret["pollButtonTiebreaker"] = func(s *discordgo.Session, interaction *discordgo.InteractionCreate) {
						log.Println("Button clicked: ", interaction.Message.ID, interaction.Member.User.Username)
						poll := parsePoll(interaction.Message.Content)

						ties, ok := poll.hasTie()
						if !ok {
							return
						}
						chosen := rand.Intn(len(ties))
						log.Printf("chose %d as a tiebreaker", chosen)

						poll.choices[chosen].count++
						poll.choices[chosen].mentions = append(poll.choices[chosen].mentions, s.State.User.Mention())

						// Build the buttons
						buttons := poll.buttons()

						s.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
							Type: discordgo.InteractionResponseUpdateMessage,
							Data: &discordgo.InteractionResponseData{
								Content: poll.serialize(),
								Flags:   interaction.Message.Flags,
								Components: []discordgo.MessageComponent{
									discordgo.ActionsRow{Components: buttons},
								},
							},
						})
					}

					// next the voting buttons
					for i := 0; i < 20; i++ {
						choiceN := i
						customID := fmt.Sprintf("pollButton%d", i)
						ret[customID] = func(s *discordgo.Session, interaction *discordgo.InteractionCreate) {
							pollMutex.Lock()
							defer pollMutex.Unlock()

							log.Println("Button clicked: ", interaction.Message.ID, interaction.Member.User.Username)
							poll := parsePoll(interaction.Message.Content)

							// Build the new user list
							var alreadyVoted bool
							newUsers := []string{}
							for _, user := range poll.choices[choiceN].mentions {
								// Skip the bot which might have a tiebreaker vote
								if user == s.State.User.Mention() {
									continue
								}

								// If the user already voted, they're un-voting
								if user == interaction.Member.Mention() {
									alreadyVoted = true
									continue
								}

								newUsers = append(newUsers, user)
							}

							// If the user is voting for the first time, add them
							if !alreadyVoted {
								newUsers = append(newUsers, interaction.Member.Mention())
							}

							poll.choices[choiceN].count = len(newUsers)
							poll.choices[choiceN].mentions = newUsers

							// Log the new poll string
							log.Println("New poll: ", poll)

							// Build the buttons
							buttons := poll.buttons()

							// And update the string on the server
							s.InteractionRespond(interaction.Interaction, &discordgo.InteractionResponse{
								Type: discordgo.InteractionResponseUpdateMessage,
								Data: &discordgo.InteractionResponseData{
									Content: poll.serialize(),
									Flags:   interaction.Message.Flags,
									Components: []discordgo.MessageComponent{
										discordgo.ActionsRow{Components: buttons},
									},
								},
							})
						}
					}

					return ret
				}(),
				Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
					// Build the poll
					poll := poll{prompt: "Poll:"}
					var choicesString string
					var flags discordgo.MessageFlags
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
					}

					// Build the buttons
					buttons := poll.buttons()

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

func (p *poll) buttons() []discordgo.MessageComponent {
	buttons := []discordgo.MessageComponent{}
	for i := range p.choices {
		buttons = append(buttons, discordgo.Button{
			CustomID: fmt.Sprintf("pollButton%d", i),
			Label:    fmt.Sprintf("%d", i+1),
		})
	}

	if _, tie := p.hasTie(); tie {
		buttons = append(buttons, discordgo.Button{
			CustomID: "pollButtonTiebreaker",
			Label:    "Tiebreaker!",
		})
	}

	return buttons
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
		c.count = int(count)

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

func (p *poll) hasTie() ([]int, bool) {
	var maxCount int
	maxIndexes := []int{}

	for i, choice := range p.choices {
		if choice.count == 0 {
			continue
		} else if choice.count > maxCount {
			maxCount = choice.count
			maxIndexes = []int{i}
		} else if choice.count == maxCount {
			maxIndexes = append(maxIndexes, i)
		}
	}

	return maxIndexes, len(maxIndexes) > 1
}
