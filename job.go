package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/robfig/cron/v3"
	"github.com/rs/xid"
	"gopkg.in/yaml.v3"
)

type JobSchedule struct {
	RunOnStart *bool  `yaml:"run_on_start"`
	Every      int64  `yaml:"every"`
	At         string `yaml:"at"`
}

type Executor struct {
	URL     string            `yaml:"url"`
	Method  string            `yaml:"method"`
	Headers map[string]string `yaml:"headers"`
}

type Task struct {
	Name     string       `json:"name" yaml:"name"`
	Executor *Executor    `json:"executor"`
	Schedule *JobSchedule `yaml:"schedule"`
}

type Notifier struct {
	Slack string `yaml:"slack"`
}

type Config struct {
	Headers map[string]string `yaml:"headers"`
	Tasks   []*Task           `yaml:"tasks"`

	Notifier *Notifier `yaml:"notifier"`
}

func (c *Config) Parse(data []byte) error {
	return yaml.Unmarshal(data, c)
}

type Endhouse struct {
	Config  *Config
	Client  *resty.Client
	Counter map[string]int64
	Cron    *cron.Cron

	Slacker *Slacker

	done map[string]chan bool
	lock map[string]sync.Mutex
}

func (c *Endhouse) Run() {
	t0 := time.Now()
	total := 0
	for i, j := range c.Config.Tasks {
		// TODO: Slack error?
		if j.Executor == nil {
			log.Printf("job %s has no executor")
		}

		if j.Schedule == nil || (j.Schedule.Every == 0 && j.Schedule.At == "") {
			log.Printf("job %s has no schedule or schedule is invalid", j.Name)
			continue
		}

		if j.Name == "" {
			c.Config.Tasks[i].Name = xid.New().String()
			j = c.Config.Tasks[i]
			log.Printf("job has no name, generate random name: %s")
		}

		// expand env into header or url
		if j.Executor.URL != "" {
			j.Executor.URL = os.ExpandEnv(j.Executor.URL)
		}

		if j.Executor.Headers == nil {
			j.Executor.Headers = make(map[string]string)
		}

		for k, v := range c.Config.Headers {
			if j.Executor.Headers[k] == "" {
				j.Executor.Headers[k] = v
			}
		}

		for k, v := range j.Executor.Headers {
			j.Executor.Headers[k] = os.ExpandEnv(v)
		}

		if j.Schedule.RunOnStart != nil && *j.Schedule.RunOnStart {
			// perform this job once
			go c.ExecuteTask(j, time.Now())
		}

		c.done[j.Name] = make(chan bool, 1)
		c.lock[j.Name] = sync.Mutex{}

		if j.Schedule.Every > 0 {
			log.Printf("found job %s endpoint: %s schedule every: %d seconds\n", j.Name, j.Executor.URL, j.Schedule.Every)
			go c.RepeatTask(j)
		} else {
			log.Printf("found job %s endpoint: %s schedule at: %s\n", j.Name, j.Executor.URL, j.Schedule.At)
			go c.ScheduleTask(j)
		}

		total += 1
	}

	sections := []MessageBlock{
		MessageBlock{
			Type: "Section",
			Fields: []MessageText{
				MessageText{
					Type: "mrkdwn",
					Text: "*Boot time*",
				},

				MessageText{
					Type: "mrkdwn",
					Text: "*Job loaded*",
				},

				MessageText{
					Type: "mrkdwn",
					Text: fmt.Sprintf("%s", time.Now().Sub(t0)),
				},

				MessageText{
					Type: "mrkdwn",
					Text: fmt.Sprintf("%d", total),
				},
			},
		},
	}

	c.Slacker.Send(fmt.Sprintf(`*%s* booted at %s`, appname, time.Now()), sections)

	c.Cron.Start()
}

func (c *Endhouse) ScheduleTask(t *Task) {
	c.Cron.AddFunc(t.Schedule.At, func() {
		c.ExecuteTask(t, time.Now())
	})
}

func (c *Endhouse) RepeatTask(t *Task) {
	ticker := time.NewTicker(time.Duration(t.Schedule.Every) * time.Second)
	for {
		select {
		case <-c.done[t.Name]:
			return
		case at := <-ticker.C:
			c.ExecuteTask(t, at)
		}
	}

}

func (c *Endhouse) ExecuteTask(t *Task, t0 time.Time) {
	log.Printf("execute task %s at %s", t.Name, t0)
	lock := c.lock[t.Name]

	lock.Lock()
	defer lock.Unlock()

	req := c.Client.R()

	for k, v := range t.Executor.Headers {
		req.SetHeader(k, v)
	}

	resp, err := req.Get(t.Executor.URL)

	if err != nil {
		log.Printf("job %s performed in %s respond status %d body=(%s)", t.Name, time.Now().Sub(t0), resp.StatusCode(), resp)

		c.Slacker.Send(fmt.Sprintf(":bangbang: error pushing to url %s: %s. Resp %s\n", t.Executor.URL, err, resp), []MessageBlock{})
	} else {
		log.Printf("job %s performed in %s respond status %d body=(%s)", t.Name, time.Now().Sub(t0), resp.StatusCode(), resp)

		sections := []MessageBlock{
			MessageBlock{
				Type: "section",
				Fields: []MessageText{
					MessageText{
						Type: "mrkdwn",
						Text: fmt.Sprintf("*Job*\n%s", t.Name),
					},

					MessageText{
						Type: "mrkdwn",
						Text: "*Result*\nSucceed",
					},
				},
			},

			MessageBlock{
				Type: "section",
				Fields: []MessageText{
					MessageText{
						Type: "mrkdwn",
						Text: fmt.Sprintf("*Duration*\n%s", time.Now().Sub(t0)),
					},

					MessageText{
						Type: "mrkdwn",
						Text: fmt.Sprintf("*Status*\n%d", resp.StatusCode()),
					},
				},
			},
		}

		c.Slacker.Send(fmt.Sprintf(":ship: job *%s* finished with resp:\n\n>%s\n", t.Name, resp), sections)
	}
}

func (c *Endhouse) Wait() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	doneAll := make(chan bool, 1)

	go func() {
		// This goroutine executes a blocking receive for
		// signals. When it gets one it'll print it out
		// and then notify the program that it can finish.
		sig := <-sigs
		log.Printf("catch signal %s. Quit", sig)

		for k, v := range c.done {
			log.Printf("send done signal to job: %s", k)
			v <- true
		}

		doneAll <- true
	}()

	// wail until other go routing quit
	<-doneAll
}
