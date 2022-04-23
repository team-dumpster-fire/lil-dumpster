package cmd

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/team-dumpster-fire/lil-dumpster/internal/state"
)

type rotator struct {
	s       discordRotatorSession
	channel string
	store   state.Backend
	prefix  string
}

type rotation struct {
	Current         int
	CurrentAssigned time.Time
	Users           []string
}

func init() {
	fnRegisterCommands = append(fnRegisterCommands, func(store state.Backend) []applicationCommand {
		return []applicationCommand{
			{
				Command: &discordgo.ApplicationCommand{
					Name:        "rotator",
					Description: "Display the current user in the channel rotation",
					Options:     []*discordgo.ApplicationCommandOption{},
				},
				Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
					rot := newRotator(s, i.ChannelID, store)

					currentUser, currentAssigned, err := rot.Current()
					if err != nil {
						log.Println("Could not look up current user:", err)
						commandError(s, i.Interaction, err)
						return
					}

					list, err := rot.ListFormatted()
					if err != nil {
						log.Println("Could not render current list:", err)
						commandError(s, i.Interaction, err)
						return
					}

					err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: fmt.Sprintf("%s\n\n%s is the current user as of %s", list, currentUser.Mention(), currentAssigned.Format("2006-01-02")),
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
					Name:        "rotator-add",
					Description: "Add a user to the channel rotation",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "username",
							Description: "Name of the user to be added",
							Required:    true,
						},
					},
				},
				Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
					user := i.ApplicationCommandData().Options[0].UserValue(s)

					rot := newRotator(s, i.ChannelID, store)
					if err := rot.AddUser(user.ID); err != nil {
						log.Println("Could not add user to rotation:", err)
						commandError(s, i.Interaction, err)
						return
					}

					list, err := rot.ListFormatted()
					if err != nil {
						log.Println("Could not render current list:", err)
						commandError(s, i.Interaction, err)
						return
					}

					err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: fmt.Sprintf("%s\n\n%s has been added to the rotation", list, user.Mention()),
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
					Name:        "rotator-remove",
					Description: "Remove a person from the channel rotation",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionUser,
							Name:        "username",
							Description: "Name of the user to be added",
							Required:    true,
						},
					},
				},
				Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
					user := i.ApplicationCommandData().Options[0].UserValue(s)

					rot := newRotator(s, i.ChannelID, store)
					if err := rot.RemoveUser(user.ID); err != nil {
						log.Println("Could not remove user from rotation:", err)
						commandError(s, i.Interaction, err)
						return
					}

					list, err := rot.ListFormatted()
					if err != nil {
						log.Println("Could not render current list:", err)
						commandError(s, i.Interaction, err)
						return
					}

					err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: fmt.Sprintf("%s\n\n%s has been removed from the rotation", list, user.Mention()),
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
					Name:        "rotator-advance",
					Description: "Advance the channel rotation to the next user",
					Options:     []*discordgo.ApplicationCommandOption{},
				},
				Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
					rot := newRotator(s, i.ChannelID, store)
					user, err := rot.Advance()
					if err != nil {
						log.Println("Could not advance rotation:", err)
						commandError(s, i.Interaction, err)
						return
					}

					list, err := rot.ListFormatted()
					if err != nil {
						log.Println("Could not render current list:", err)
						commandError(s, i.Interaction, err)
						return
					}

					err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: fmt.Sprintf("%s\n\n%s is now assigned in the rotation!", list, user.Mention()),
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

type discordRotatorSession interface {
	User(userID string) (st *discordgo.User, err error)
}

func newRotator(s discordRotatorSession, channel string, store state.Backend) *rotator {
	return &rotator{
		s:       s,
		channel: channel,
		store:   store,
		prefix:  fmt.Sprintf("rotator/%s/", channel),
	}
}

func (r *rotator) Current() (*discordgo.User, time.Time, error) {
	data, err := r.getRotation()
	if err != nil {
		return &discordgo.User{}, time.Now(), fmt.Errorf("unable to get current rotation: %w", err)
	} else if len(data.Users) == 0 {
		return &discordgo.User{}, data.CurrentAssigned, errors.New("no users currently in rotation")
	} else if data.Current >= len(data.Users) {
		return &discordgo.User{}, data.CurrentAssigned, errors.New("current user out of range")
	}

	user, err := r.resolveUser(data.Users[data.Current])
	if err != nil {
		return user, data.CurrentAssigned, fmt.Errorf("could not resolve current user: %w", err)
	}

	return user, data.CurrentAssigned, nil
}

func (r *rotator) Peek() (*discordgo.User, error) {
	data, err := r.getRotation()
	if err != nil {
		return &discordgo.User{}, fmt.Errorf("unable to get current rotation: %w", err)
	} else if len(data.Users) == 0 {
		return &discordgo.User{}, errors.New("no users currently in rotation")
	}

	next := data.Current + 1
	if next >= len(data.Users) {
		next = 0
	}

	return r.resolveUser(data.Users[next])
}

func (r *rotator) ListFormatted() (string, error) {
	data, err := r.getRotation()
	if err != nil {
		return "", fmt.Errorf("unable to get current rotation: %w", err)
	} else if len(data.Users) == 0 {
		return "", errors.New("no users currently in rotation")
	}

	ret := []string{}
	for i, id := range data.Users {
		user, err := r.resolveUser(id)
		if err != nil {
			user = &discordgo.User{ID: id, Username: id}
		}

		add := user.Username
		if i == data.Current {
			add = "**" + add + "**"
		}

		ret = append(ret, add)
	}

	return "[ " + strings.Join(ret, " => ") + " ]", nil
}

func (r *rotator) Advance() (*discordgo.User, error) {
	data, err := r.getRotation()
	if err != nil {
		return &discordgo.User{}, fmt.Errorf("unable to get current rotation: %w", err)
	} else if len(data.Users) == 0 {
		return &discordgo.User{}, errors.New("no users currently in rotation")
	}

	data.Current++
	data.CurrentAssigned = time.Now()
	var ret *discordgo.User
	for ret == nil {
		if data.Current >= len(data.Users) {
			data.Current = 0
		}

		ret, err = r.resolveUser(data.Users[data.Current])
		if err != nil {
			return ret, fmt.Errorf("could not get user %q: %w", data.Users[data.Current], err)
		}
	}

	return ret, r.setRotation(data)
}

func (r *rotator) AddUser(id string) error {
	data, err := r.getRotation()
	if err != nil {
		return fmt.Errorf("unable to get current rotation: %w", err)
	} else if _, err = r.resolveUser(id); err != nil {
		return fmt.Errorf("unable to resolve user id: %w", err)
	}

	for _, newID := range data.Users {
		if newID == id {
			return errors.New("user is already in the rotation")
		}
	}

	data.Users = append(data.Users, id)

	// If this is the first user to be added, they're automatically the current!
	if len(data.Users) == 1 {
		data.CurrentAssigned = time.Now()
	}

	return r.setRotation(data)
}

func (r *rotator) RemoveUser(id string) error {
	data, err := r.getRotation()
	if err != nil {
		return fmt.Errorf("unable to get current rotation: %w", err)
	}

	for i := range data.Users {
		if data.Users[i] == id {
			data.Users = append(data.Users[0:i], data.Users[i+1:]...)

			// If we removed the current user, then the next user just got assigned
			if i == data.Current {
				data.CurrentAssigned = time.Now()
			}
			break
		}
	}

	return r.setRotation(data)
}

func (r *rotator) resolveUser(id string) (*discordgo.User, error) {
	return r.s.User(id)
}

func (r *rotator) getRotation() (rotation, error) {
	data := rotation{}
	err := r.store.Get(r.prefix+"rotation", &data)
	if err != nil {
		log.Printf("Rotation not found for %q. Creating an empty one", r.prefix)
		err = r.store.Set(r.prefix+"rotation", &data)
	}

	return data, err
}

func (r *rotator) setRotation(data rotation) error {
	return r.store.Set(r.prefix+"rotation", &data)
}
