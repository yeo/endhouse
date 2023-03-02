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

type SlackMessage struct {
	Text string `json:"text"`
}

func (s *Slacker) Send(message string) {
	if s.WebhookURL == "" {
		log.Printf("slack webhook url isn't configured. print out body in stderr instead: %s", message)
		return
	}

	body, _ := json.Marshal(SlackMessage{
		Text: message,
	})

	resp, err := s.Client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(body).
		Post(s.WebhookURL)

	if err == nil {
		log.Printf("slack resp: %s", resp)
	} else {
		log.Printf("error pushing to slack %s\n", err)
	}
}
