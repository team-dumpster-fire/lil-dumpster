package cmd

import (
	"context"
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
		r       rotation
		want    rotationUser
		wantErr bool
	}{
		{
			name: "found",
			r: rotation{
				Current: 0,
				Users: []rotationUser{
					{ID: "123"},
				},
			},
			want: rotationUser{ID: "123"},
		},
		{
			name: "found+1",
			r: rotation{
				Current: 1,
				Users: []rotationUser{
					{ID: "123"},
					{ID: "456"},
				},
			},
			want: rotationUser{ID: "456"},
		},
		{
			name: "empty",
			r: rotation{
				Current: 0,
				Users:   []rotationUser{},
			},
			want:    rotationUser{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &rotator{
				store: state.NewMemory(),
			}
			r.store.Set(context.Background(), "rotation", tt.r)

			got, err := r.Current(context.Background())
			if (err != nil) != tt.wantErr {
				t.Errorf("rotator.Current() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("rotator.Current() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_rotator_ListFormatted(t *testing.T) {
	tests := []struct {
		name    string
		r       rotation
		s       rotatorSession
		want    string
		wantErr bool
	}{
		{
			name: "basic",
			r: rotation{
				Current: 0,
				Users: []rotationUser{
					{ID: "123"},
					{ID: "456"},
				},
			},
			s: &mockDiscordSession{
				mockUser: func(userID string, opt ...discordgo.RequestOption) (st *discordgo.User, err error) {
					if userID == "123" || userID == "456" {
						return &discordgo.User{ID: userID, Username: userID}, nil
					}
					return nil, errors.New("unexpexted user requested")
				},
			},
			want: "[ **123** :fast_forward: 456 ]",
		},
		{
			name: "empty",
			r: rotation{
				Current: 0,
				Users:   []rotationUser{},
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &rotator{
				store: state.NewMemory(),
			}
			r.store.Set(context.Background(), "rotation", tt.r)

			got, err := r.ListFormatted(context.Background(), tt.s)
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
		reverse bool
		r       rotation
		want    rotationUser
		wantErr bool
	}{
		{
			name: "basic",
			r: rotation{
				Current: 0,
				Users: []rotationUser{
					{ID: "123", LastAssigned: time.Now()},
					{ID: "456"},
					{ID: "789"},
				},
			},
			want: rotationUser{ID: "456"},
		},
		{
			name: "overflow",
			r: rotation{
				Current: 2,
				Users: []rotationUser{
					{ID: "123"},
					{ID: "456"},
					{ID: "789", LastAssigned: time.Now()},
				},
			},
			want: rotationUser{ID: "123"},
		},
		{
			name: "reverse",
			r: rotation{
				Current: 1,
				Users: []rotationUser{
					{ID: "123"},
					{ID: "456", LastAssigned: time.Now()},
					{ID: "789"},
				},
			},
			reverse: true,
			want:    rotationUser{ID: "123"},
		},
		{
			name: "reverse-overflow",
			r: rotation{
				Current: 0,
				Users: []rotationUser{
					{ID: "123", LastAssigned: time.Now()},
					{ID: "456"},
					{ID: "789"},
				},
			},
			reverse: true,
			want:    rotationUser{ID: "789"},
		},
		{
			name: "empty",
			r: rotation{
				Current: 0,
				Users:   []rotationUser{},
			},
			want:    rotationUser{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &rotator{
				store: state.NewMemory(),
			}
			r.store.Set(context.Background(), "rotation", tt.r)

			got, err := r.Advance(context.Background(), tt.reverse)
			if (err != nil) != tt.wantErr {
				t.Errorf("rotator.Advance() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			got.LastAssigned = time.Time{}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("rotator.Advance() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_rotator_AddUser(t *testing.T) {
	type args struct {
		user discordgo.User
	}
	tests := []struct {
		name      string
		r         rotation
		args      args
		wantState rotation
		wantErr   bool
	}{
		{
			name: "basic",
			r: rotation{
				Current: 0,
				Users: []rotationUser{
					{ID: "123", LastAssigned: time.Now()},
					{ID: "456"},
				},
			},
			args: args{discordgo.User{ID: "789"}},
			wantState: rotation{
				Current: 0,
				Users: []rotationUser{
					{ID: "123", LastAssigned: time.Now()},
					{ID: "456"},
					{ID: "789"},
				},
			},
		},
		{
			name: "first",
			r: rotation{
				Current: 0,
				Users:   []rotationUser{},
			},
			args: args{discordgo.User{ID: "123"}},
			wantState: rotation{
				Current: 0,
				Users: []rotationUser{
					{ID: "123", LastAssigned: time.Now()},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &rotator{
				store: state.NewMemory(),
			}
			r.store.Set(context.Background(), "rotation", tt.r)

			if err := r.AddUser(context.Background(), tt.args.user); (err != nil) != tt.wantErr {
				t.Errorf("rotator.AddUser() error = %v, wantErr %v", err, tt.wantErr)
			}

			gotState := rotation{}
			r.store.Get(context.Background(), "rotation", &gotState)
			if len(gotState.Users) != len(tt.wantState.Users) {
				t.Fatalf("rotator.AddUser() users = %d, want %d", len(gotState.Users), len(tt.wantState.Users))
			}

			for i := range gotState.Users {
				if gotState.Users[i].LastAssigned.IsZero() != tt.wantState.Users[i].LastAssigned.IsZero() {
					t.Errorf("rotator.AddUser() time at %d = %v, wantTime %v", i, gotState.Users[i].LastAssigned.IsZero(), tt.wantState.Users[i].LastAssigned.IsZero())
				}
				gotState.Users[i].LastAssigned = time.Time{}
				tt.wantState.Users[i].LastAssigned = time.Time{}
			}

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
		r         rotation
		args      args
		wantState rotation
		wantErr   bool
	}{
		{
			name: "basic",
			r: rotation{
				Current: 0,
				Users: []rotationUser{
					{ID: "123", LastAssigned: time.Now()},
					{ID: "456"},
				},
			},
			args: args{id: "123"},
			wantState: rotation{
				Current: 0,
				Users: []rotationUser{
					{ID: "456", LastAssigned: time.Now()},
				},
			},
		},
		{
			name: "remove-last-user",
			r: rotation{
				Current: 1,
				Users: []rotationUser{
					{ID: "123"},
					{ID: "456", LastAssigned: time.Now()},
				},
			},
			args: args{id: "456"},
			wantState: rotation{
				Current: 0,
				Users: []rotationUser{
					{ID: "123", LastAssigned: time.Now()},
				},
			},
		},
		{
			name: "remove-inactive-user",
			r: rotation{
				Current: 0,
				Users: []rotationUser{
					{ID: "123", LastAssigned: time.Now()},
					{ID: "456"},
				},
			},
			args: args{id: "456"},
			wantState: rotation{
				Current: 0,
				Users: []rotationUser{
					{ID: "123", LastAssigned: time.Now()},
				},
			},
		},
		{
			name: "remove-non-user",
			r: rotation{
				Current: 0,
				Users: []rotationUser{
					{ID: "123"},
					{ID: "456"},
				},
			},
			args: args{id: "789"},
			wantState: rotation{
				Current: 0,
				Users: []rotationUser{
					{ID: "123"},
					{ID: "456"},
				},
			},
		},
		{
			name: "empty",
			r: rotation{
				Current: 0,
				Users:   []rotationUser{},
			},
			wantState: rotation{
				Current: 0,
				Users:   []rotationUser{},
			},
			args: args{id: "123"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &rotator{
				store: state.NewMemory(),
			}
			r.store.Set(context.Background(), "rotation", tt.r)

			if err := r.RemoveUser(context.Background(), tt.args.id); (err != nil) != tt.wantErr {
				t.Errorf("rotator.RemoveUser() error = %v, wantErr %v", err, tt.wantErr)
			}

			gotState := rotation{}
			r.store.Get(context.Background(), "rotation", &gotState)
			if len(gotState.Users) != len(tt.wantState.Users) {
				t.Fatalf("rotator.RemoveUser() users = %d, want %d", len(gotState.Users), len(tt.wantState.Users))
			}

			for i := range gotState.Users {
				if gotState.Users[i].LastAssigned.IsZero() != tt.wantState.Users[i].LastAssigned.IsZero() {
					t.Errorf("rotator.RemoveUser() time at %d = %v, wantTime %v", i, gotState.Users[i].LastAssigned.IsZero(), tt.wantState.Users[i].LastAssigned.IsZero())
				}
				gotState.Users[i].LastAssigned = time.Time{}
				tt.wantState.Users[i].LastAssigned = time.Time{}
			}

			if !reflect.DeepEqual(tt.wantState, gotState) {
				t.Errorf("rotator.RemoveUser() state = %v, wantState %v", gotState, tt.wantState)
			}
		})
	}
}
