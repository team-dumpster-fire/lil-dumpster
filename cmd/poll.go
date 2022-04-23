package cmd

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"sync"
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
