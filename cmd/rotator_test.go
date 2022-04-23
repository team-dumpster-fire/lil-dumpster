package cmd

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/team-dumpster-fire/lil-dumpster/internal/state"
)

func Test_rotator_Current(t *testing.T) {
	tests := []struct {
		name    string
		s       discordRotatorSession
		r       rotation
		want    *discordgo.User
		want1   time.Time
		wantErr bool
	}{
		{
			name: "found",
			r: rotation{
				Current: 0,
				Users:   []string{"123"},
			},
			s: &mockDiscordSession{
				mockUser: func(userID string) (st *discordgo.User, err error) {
					if userID == "123" {
						return &discordgo.User{ID: userID}, nil
					}
					return nil, errors.New("unexpected user requested")
				},
			},
			want: &discordgo.User{ID: "123"},
		},
		{
			name: "found+1",
			r: rotation{
				Current: 1,
				Users:   []string{"123", "456"},
			},
			s: &mockDiscordSession{
				mockUser: func(userID string) (st *discordgo.User, err error) {
					if userID == "123" || userID == "456" {
						return &discordgo.User{ID: userID}, nil
					}
					return nil, errors.New("unexpected user requested")
				},
			},
			want: &discordgo.User{ID: "456"},
		},
		{
			name: "empty",
			r: rotation{
				Current: 0,
				Users:   []string{},
			},
			want:    &discordgo.User{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &rotator{
				s:     tt.s,
				store: state.NewMemory(),
			}
			r.store.Set("rotation", tt.r)

			got, got1, err := r.Current()
			if (err != nil) != tt.wantErr {
				t.Errorf("rotator.Current() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("rotator.Current() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t.Errorf("rotator.Current() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_rotator_Peek(t *testing.T) {
	tests := []struct {
		name    string
		s       discordRotatorSession
		r       rotation
		want    *discordgo.User
		wantErr bool
	}{
		{
			name: "basic",
			r: rotation{
				Current: 0,
				Users:   []string{"123", "456"},
			},
			s: &mockDiscordSession{
				mockUser: func(userID string) (st *discordgo.User, err error) {
					if userID == "123" || userID == "456" {
						return &discordgo.User{ID: userID}, nil
					}
					return nil, errors.New("unexpected user requested")
				},
			},
			want: &discordgo.User{ID: "456"},
		},
		{
			name: "overflow",
			r: rotation{
				Current: 1,
				Users:   []string{"123", "456"},
			},
			s: &mockDiscordSession{
				mockUser: func(userID string) (st *discordgo.User, err error) {
					if userID == "123" || userID == "456" {
						return &discordgo.User{ID: userID}, nil
					}
					return nil, errors.New("unexpected user requested")
				},
			},
			want: &discordgo.User{ID: "123"},
		},
		{
			name: "empty",
			r: rotation{
				Current: 0,
				Users:   []string{},
			},
			want:    &discordgo.User{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &rotator{
				s:     tt.s,
				store: state.NewMemory(),
			}
			r.store.Set("rotation", tt.r)

			got, err := r.Peek()
			if (err != nil) != tt.wantErr {
				t.Errorf("rotator.Peek() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("rotator.Peek() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_rotator_ListFormatted(t *testing.T) {
	tests := []struct {
		name    string
		s       discordRotatorSession
		r       rotation
		want    string
		wantErr bool
	}{
		{
			name: "basic",
			r: rotation{
				Current: 0,
				Users:   []string{"123", "456"},
			},
			s: &mockDiscordSession{
				mockUser: func(userID string) (st *discordgo.User, err error) {
					if userID == "123" || userID == "456" {
						return &discordgo.User{ID: userID, Username: userID}, nil
					}
					return nil, errors.New("unexpected user requested")
				},
			},
			want: "[ **123** => 456 ]",
		},
		{
			name: "empty",
			r: rotation{
				Current: 0,
				Users:   []string{},
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &rotator{
				s:     tt.s,
				store: state.NewMemory(),
			}
			r.store.Set("rotation", tt.r)

			got, err := r.ListFormatted()
			if (err != nil) != tt.wantErr {
				t.Errorf("rotator.ListFormatted() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("rotator.ListFormatted() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_rotator_Advance(t *testing.T) {
	tests := []struct {
		name    string
		s       discordRotatorSession
		r       rotation
		want    *discordgo.User
		wantErr bool
	}{
		{
			name: "basic",
			r: rotation{
				Current: 0,
				Users:   []string{"123", "456"},
			},
			s: &mockDiscordSession{
				mockUser: func(userID string) (st *discordgo.User, err error) {
					if userID == "123" || userID == "456" {
						return &discordgo.User{ID: userID}, nil
					}
					return nil, errors.New("unexpected user requested")
				},
			},
			want: &discordgo.User{ID: "456"},
		},
		{
			name: "overflow",
			r: rotation{
				Current: 1,
				Users:   []string{"123", "456"},
			},
			s: &mockDiscordSession{
				mockUser: func(userID string) (st *discordgo.User, err error) {
					if userID == "123" || userID == "456" {
						return &discordgo.User{ID: userID}, nil
					}
					return nil, errors.New("unexpected user requested")
				},
			},
			want: &discordgo.User{ID: "123"},
		},
		{
			name: "empty",
			r: rotation{
				Current: 0,
				Users:   []string{},
			},
			want:    &discordgo.User{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &rotator{
				s:     tt.s,
				store: state.NewMemory(),
			}
			r.store.Set("rotation", tt.r)

			got, err := r.Advance()
			if (err != nil) != tt.wantErr {
				t.Errorf("rotator.Advance() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("rotator.Advance() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_rotator_AddUser(t *testing.T) {
	type args struct {
		id string
	}
	tests := []struct {
		name      string
		s         discordRotatorSession
		r         rotation
		args      args
		wantState rotation
		wantErr   bool
	}{
		{
			name: "basic",
			r: rotation{
				Current: 0,
				Users:   []string{"123", "456"},
			},
			s: &mockDiscordSession{
				mockUser: func(userID string) (st *discordgo.User, err error) {
					if userID == "123" || userID == "456" || userID == "789" {
						return &discordgo.User{ID: userID}, nil
					}
					return nil, errors.New("unexpected user requested")
				},
			},
			args: args{id: "789"},
			wantState: rotation{
				Current: 0,
				Users:   []string{"123", "456", "789"},
			},
		},
		{
			name: "unknown-user",
			r: rotation{
				Current: 0,
				Users:   []string{},
			},
			s: &mockDiscordSession{
				mockUser: func(userID string) (st *discordgo.User, err error) {
					return nil, errors.New("unknown user")
				},
			},
			wantState: rotation{
				Current: 0,
				Users:   []string{},
			},
			args:    args{id: "789"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &rotator{
				s:     tt.s,
				store: state.NewMemory(),
			}
			r.store.Set("rotation", tt.r)

			if err := r.AddUser(tt.args.id); (err != nil) != tt.wantErr {
				t.Errorf("rotator.AddUser() error = %v, wantErr %v", err, tt.wantErr)
			}

			gotState := rotation{}
			r.store.Get("rotation", &gotState)
			gotState.CurrentAssigned = time.Time{}
			if !reflect.DeepEqual(tt.wantState, gotState) {
				t.Errorf("rotator.AddUser() state = %v, wantState %v", gotState, tt.wantState)
			}
		})
	}
}

func Test_rotator_RemoveUser(t *testing.T) {
	type args struct {
		id string
	}
	tests := []struct {
		name      string
		s         discordRotatorSession
		r         rotation
		args      args
		wantState rotation
		wantErr   bool
	}{
		{
			name: "basic",
			r: rotation{
				Current: 0,
				Users:   []string{"123", "456"},
			},
			args: args{id: "123"},
			wantState: rotation{
				Current: 0,
				Users:   []string{"456"},
			},
		},
		{
			name: "remove-non-user",
			r: rotation{
				Current: 0,
				Users:   []string{"123", "456"},
			},
			args: args{id: "789"},
			wantState: rotation{
				Current: 0,
				Users:   []string{"123", "456"},
			},
		},
		{
			name: "empty",
			r: rotation{
				Current: 0,
				Users:   []string{},
			},
			wantState: rotation{
				Current: 0,
				Users:   []string{},
			},
			args: args{id: "123"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &rotator{
				s:     tt.s,
				store: state.NewMemory(),
			}
			r.store.Set("rotation", tt.r)

			if err := r.RemoveUser(tt.args.id); (err != nil) != tt.wantErr {
				t.Errorf("rotator.RemoveUser() error = %v, wantErr %v", err, tt.wantErr)
			}

			gotState := rotation{}
			r.store.Get("rotation", &gotState)
			gotState.CurrentAssigned = time.Time{}
			if !reflect.DeepEqual(tt.wantState, gotState) {
				t.Errorf("rotator.RemoveUser() state = %v, wantState %v", gotState, tt.wantState)
			}
		})
	}
}
