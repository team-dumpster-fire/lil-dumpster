package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/team-dumpster-fire/lil-dumpster/internal/state"
)

type (
	rotator struct {
		channel string
		store   state.Backend
		prefix  string
	}

	rotationUser struct {
		ID           string
		LastAssigned time.Time
	}

	rotation struct {
		Current int
		Users   []rotationUser
	}

	rotatorSession interface {
		User(userID string) (st *discordgo.User, err error)
	}
)

func init() {
	fnRegisterCommands = append(fnRegisterCommands, func(store state.Backend) []applicationCommand {
		return []applicationCommand{
			{
				Command: &discordgo.ApplicationCommand{
					Name:        "rotator",
					Description: "Display the current user in the channel rotation",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionBoolean,
							Name:        "announce",
							Description: "Post the response publicly for all to see",
						},
					},
				},
				Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
					var announce bool
					if len(i.ApplicationCommandData().Options) > 0 {
						announce = i.ApplicationCommandData().Options[0].BoolValue()
					}

					rot := newRotator(i.ChannelID, store)

					currentUser, err := rot.Current(context.TODO())
					if err != nil {
						log.Println("Could not look up current user:", err)
						commandError(s, i.Interaction, err)
						return
					}

					list, err := rot.ListFormatted(context.TODO(), s)
					if err != nil {
						log.Println("Could not render current list:", err)
						commandError(s, i.Interaction, err)
						return
					}

					var flags uint64
					if !announce {
						flags = 1 << 6 // Ephemeral, private
					}
					err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: fmt.Sprintf("%s\n\n%s is the current user as of <t:%d:R>", list, currentUser.resolve(s).Mention(), currentUser.LastAssigned.Unix()),
							Flags:   flags,
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

					if user == nil {
						log.Println("A user must be provided")
						commandError(s, i.Interaction, errors.New("a user must be provided"))
						return
					}

					rot := newRotator(i.ChannelID, store)
					if err := rot.AddUser(context.TODO(), *user); err != nil {
						log.Println("Could not add user to rotation:", err)
						commandError(s, i.Interaction, err)
						return
					}

					list, err := rot.ListFormatted(context.TODO(), s)
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

					rot := newRotator(i.ChannelID, store)
					if err := rot.RemoveUser(context.TODO(), user.ID); err != nil {
						log.Println("Could not remove user from rotation:", err)
						commandError(s, i.Interaction, err)
						return
					}

					list, err := rot.ListFormatted(context.TODO(), s)
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
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionBoolean,
							Name:        "reverse",
							Description: "Advance to the prior user in the rotation",
						},
					},
				},
				Handler: func(s *discordgo.Session, i *discordgo.InteractionCreate) {
					var reverse bool
					if len(i.ApplicationCommandData().Options) > 0 {
						reverse = i.ApplicationCommandData().Options[0].BoolValue()
					}

					rot := newRotator(i.ChannelID, store)
					user, err := rot.Advance(context.TODO(), reverse)
					if err != nil {
						log.Println("Could not advance rotation:", err)
						commandError(s, i.Interaction, err)
						return
					}

					list, err := rot.ListFormatted(context.TODO(), s)
					if err != nil {
						log.Println("Could not render current list:", err)
						commandError(s, i.Interaction, err)
						return
					}

					err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: fmt.Sprintf("%s\n\n%s is now assigned in the rotation!", list, user.resolve(s).Mention()),
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

func newRotator(channel string, store state.Backend) *rotator {
	return &rotator{
		channel: channel,
		store:   store,
		prefix:  fmt.Sprintf("rotator/%s/", channel),
	}
}

func (r *rotator) Current(ctx context.Context) (rotationUser, error) {
	data, err := r.getRotation(ctx)
	if err != nil {
		return rotationUser{}, fmt.Errorf("unable to get current rotation: %w", err)
	} else if len(data.Users) == 0 {
		return rotationUser{}, errors.New("no users currently in rotation")
	} else if data.Current >= len(data.Users) {
		return rotationUser{}, errors.New("current user out of range")
	}

	return data.Users[data.Current], nil
}

func (r *rotator) ListFormatted(ctx context.Context, s rotatorSession) (string, error) {
	data, err := r.getRotation(ctx)
	if err != nil {
		return "", fmt.Errorf("unable to get current rotation: %w", err)
	} else if len(data.Users) == 0 {
		return "", errors.New("no users currently in rotation")
	}

	ret := []string{}
	for i, user := range data.Users {
		add := user.resolve(s).Username
		if i == data.Current {
			add = "**" + add + "**"
		}

		ret = append(ret, add)
	}

	return "[ " + strings.Join(ret, " :fast_forward: ") + " ]", nil
}

func (r *rotator) Advance(ctx context.Context, reverse bool) (rotationUser, error) {
	data, err := r.getRotation(ctx)
	if err != nil {
		return rotationUser{}, fmt.Errorf("unable to get current rotation: %w", err)
	} else if len(data.Users) == 0 {
		return rotationUser{}, errors.New("no users currently in rotation")
	}

	if reverse {
		data.Current--
		if data.Current < 0 {
			data.Current = len(data.Users) - 1
		}
	} else {
		data.Current++
		if data.Current >= len(data.Users) {
			data.Current = 0
		}
	}

	data.Users[data.Current].LastAssigned = time.Now()
	return data.Users[data.Current], r.setRotation(ctx, data)
}

func (r *rotator) AddUser(ctx context.Context, user discordgo.User) error {
	data, err := r.getRotation(ctx)
	if err != nil {
		return fmt.Errorf("unable to get current rotation: %w", err)
	}

	for _, newID := range data.Users {
		if newID.ID == user.ID {
			return errors.New("user is already in the rotation")
		}
	}

	add := rotationUser{ID: user.ID}

	// If this is the first user to be added, they're automatically the current!
	if len(data.Users) == 0 {
		add.LastAssigned = time.Now()
	}

	data.Users = append(data.Users, add)
	return r.setRotation(ctx, data)
}

func (r *rotator) RemoveUser(ctx context.Context, id string) error {
	data, err := r.getRotation(ctx)
	if err != nil {
		return fmt.Errorf("unable to get current rotation: %w", err)
	}

	for i := range data.Users {
		if data.Users[i].ID == id {
			data.Users = append(data.Users[0:i], data.Users[i+1:]...)

			// If we removed the current user, then advance to the next user
			if i == data.Current {
				data.Current--
				if err := r.setRotation(ctx, data); err != nil {
					return err
				}
				_, err = r.Advance(ctx, false)
				return err
			}

			break
		}
	}

	return r.setRotation(ctx, data)
}

func (r *rotator) getRotation(ctx context.Context) (rotation, error) {
	data := rotation{}
	err := r.store.Get(ctx, r.prefix+"rotation", &data)
	if err != nil {
		log.Printf("Rotation not found for %q. Creating an empty one", r.prefix)
		err = r.store.Set(ctx, r.prefix+"rotation", &data)
	}

	return data, err
}

func (r *rotator) setRotation(ctx context.Context, data rotation) error {
	return r.store.Set(ctx, r.prefix+"rotation", &data)
}

func (u *rotationUser) resolve(s rotatorSession) *discordgo.User {
	user, err := s.User(u.ID)
	if err != nil {
		return &discordgo.User{Username: "Unknown"}
	}

	return user
}
