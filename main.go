package main

import (
	"log"
	"os"
	"sync"

	"github.com/go-resty/resty/v2"
	"github.com/robfig/cron/v3"
)

const (
	appname = "endhouse"
)

func main() {
	path := "./schedule.yaml"

	if len(os.Args) >= 2 {
		path = os.Args[1]
	}

	data, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}

	config := &Config{}
	if err := config.Parse(data); err != nil {
		log.Fatal(err)
	}

	slacker := NewSlacker(os.ExpandEnv(config.Notifier.Slack))

	c := &Endhouse{
		Config:  config,
		Counter: make(map[string]int64),
		Cron:    cron.New(),
		Client:  resty.New(),
		Slacker: slacker,

		done: make(map[string]chan bool),
		lock: make(map[string]sync.Mutex),
	}

	c.Run()
	c.Wait()
}
