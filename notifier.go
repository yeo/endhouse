package main

import (
	"encoding/json"
	"log"

	"github.com/go-resty/resty/v2"
)

func NewSlacker(url string) *Slacker {
	return &Slacker{
		Client:     resty.New(),
		WebhookURL: url,
	}
}

type Slacker struct {
	Client     *resty.Client
	WebhookURL string
}

type MessageText struct {
	Type string `json:"type,omitempty"`
	Text string `json:"text,omitempty"`
}

type MessageBlock struct {
	Type    string       `json:"type,omitempty"`
	Text    *MessageText `json:"text,omitempty"`
	BlockID string       `json:"block_id"`

	Fields []MessageText `json:"fields,omitempty"`
}

type SlackMessage struct {
	Blocks []MessageBlock `json:"blocks"`
}

func (s *Slacker) Send(message string, extraBlocks []MessageBlock) {
	if s.WebhookURL == "" {
		log.Printf("slack webhook url isn't configured. print out body in stderr instead: %s", message)
		return
	}

	blocks := []MessageBlock{
		MessageBlock{
			Type: "section",
			Text: &MessageText{
				Type: "mrkdwn",
				Text: message,
			},
			BlockID: "text1",
		},
		MessageBlock{
			Type:    "divider",
			BlockID: "div1",
		},
	}

	body, _ := json.Marshal(SlackMessage{
		Blocks: append(blocks, extraBlocks...),
	})

	resp, err := s.Client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(body).
		Post(s.WebhookURL)

	if err != nil {
		log.Printf("error pushing to slack %s resp %s\n", err, resp)
	}
}
