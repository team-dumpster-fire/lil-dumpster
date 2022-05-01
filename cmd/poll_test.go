package cmd

import (
	"reflect"
	"testing"

	"github.com/bwmarrin/discordgo"
)

func Test_parsePoll(t *testing.T) {
	tests := []struct {
		name string
		msg  string
		want poll
	}{
		{
			name: "new poll",
			msg: `Poll:
1. One (0)
2. Two (0)
3. Three (0)
`,
			want: poll{
				prompt: "Poll:",
				choices: []pollChoice{
					{choice: "One", count: 0, mentions: []string{}},
					{choice: "Two", count: 0, mentions: []string{}},
					{choice: "Three", count: 0, mentions: []string{}},
				},
			},
		},
		{
			name: "full poll",
			msg: `What number do you choose?
1. One (2, @user1, @user2)
2. Two (0)
3. Three (1, @user3)
`,
			want: poll{
				prompt: "What number do you choose?",
				choices: []pollChoice{
					{choice: "One", count: 2, mentions: []string{"@user1", "@user2"}},
					{choice: "Two", count: 0, mentions: []string{}},
					{choice: "Three", count: 1, mentions: []string{"@user3"}},
				},
			},
		},
		{
			name: "fun with spaces",
			msg: `  What number do you choose?
1. One  (2, @user1, @user2)
2.  Two (0)
3. Three   (1, @user3)
`,
			want: poll{
				prompt: "What number do you choose?",
				choices: []pollChoice{
					{choice: "One", count: 2, mentions: []string{"@user1", "@user2"}},
					{choice: "Two", count: 0, mentions: []string{}},
					{choice: "Three", count: 1, mentions: []string{"@user3"}},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := parsePoll(tt.msg); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("parsePoll() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func Test_poll_serialize(t *testing.T) {
	type fields struct {
		prompt  string
		choices []pollChoice
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "new poll",
			fields: fields{
				prompt: "Poll:",
				choices: []pollChoice{
					{choice: "One", count: 0, mentions: []string{}},
					{choice: "Two", count: 0, mentions: []string{}},
					{choice: "Three", count: 0, mentions: []string{}},
				},
			},
			want: `Poll:
1. One (0)
2. Two (0)
3. Three (0)
`,
		},
		{
			name: "full poll",
			fields: fields{
				prompt: "What number do you choose?",
				choices: []pollChoice{
					{choice: "One", count: 2, mentions: []string{"@user1", "@user2"}},
					{choice: "Two", count: 0, mentions: []string{}},
					{choice: "Three", count: 1, mentions: []string{"@user3"}},
				},
			},
			want: `What number do you choose?
1. One (2, @user1, @user2)
2. Two (0)
3. Three (1, @user3)
`,
		},
		{
			name: "fun with spaces",
			fields: fields{
				prompt: " What number do you choose? ",
				choices: []pollChoice{
					{choice: "One ", count: 2, mentions: []string{"@user1", "@user2"}},
					{choice: " Two", count: 0, mentions: []string{}},
					{choice: "Three  ", count: 1, mentions: []string{"@user3"}},
				},
			},
			want: `What number do you choose?
1. One (2, @user1, @user2)
2. Two (0)
3. Three (1, @user3)
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &poll{
				prompt:  tt.fields.prompt,
				choices: tt.fields.choices,
			}
			if got := p.serialize(); got != tt.want {
				t.Errorf("poll.serialize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_poll_hasTie(t *testing.T) {
	type fields struct {
		prompt  string
		choices []pollChoice
	}
	tests := []struct {
		name    string
		fields  fields
		want    []int
		wantTie bool
	}{
		{
			name: "no votes",
			want: []int{},
		},
		{
			name: "no tie",
			fields: fields{
				choices: []pollChoice{
					{choice: "foo", count: 1},
					{choice: "bar", count: 0},
					{choice: "baz", count: 2},
				},
			},
			want: []int{2},
		},
		{
			name: "no tie",
			fields: fields{
				choices: []pollChoice{
					{choice: "foo", count: 1},
					{choice: "bar", count: 0},
					{choice: "baz", count: 1},
				},
			},
			want:    []int{0, 2},
			wantTie: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &poll{
				prompt:  tt.fields.prompt,
				choices: tt.fields.choices,
			}
			got, got1 := p.hasTie()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("poll.hasTie() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.wantTie {
				t.Errorf("poll.hasTie() got1 = %v, want %v", got1, tt.wantTie)
			}
		})
	}
}

func Test_poll_buttons(t *testing.T) {
	type fields struct {
		prompt  string
		choices []pollChoice
	}
	tests := []struct {
		name   string
		fields fields
		want   []discordgo.MessageComponent
	}{
		{
			name: "no choices",
			want: []discordgo.MessageComponent{},
		},
		{
			name: "no tie",
			fields: fields{
				choices: []pollChoice{
					{choice: "foo", count: 1},
					{choice: "bar", count: 0},
					{choice: "baz", count: 2},
				},
			},
			want: []discordgo.MessageComponent{
				discordgo.Button{CustomID: "pollButton0", Label: "1"},
				discordgo.Button{CustomID: "pollButton1", Label: "2"},
				discordgo.Button{CustomID: "pollButton2", Label: "3"},
			},
		},
		{
			name: "no tie",
			fields: fields{
				choices: []pollChoice{
					{choice: "foo", count: 1},
					{choice: "bar", count: 0},
					{choice: "baz", count: 1},
				},
			},
			want: []discordgo.MessageComponent{
				discordgo.Button{CustomID: "pollButton0", Label: "1"},
				discordgo.Button{CustomID: "pollButton1", Label: "2"},
				discordgo.Button{CustomID: "pollButton2", Label: "3"},
				discordgo.Button{CustomID: "pollButtonTiebreaker", Label: "Tiebreaker!"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &poll{
				prompt:  tt.fields.prompt,
				choices: tt.fields.choices,
			}
			if got := p.buttons(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("poll.buttons() = %v, want %v", got, tt.want)
			}
		})
	}
}
