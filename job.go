package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
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

		c.done[j.Name] = make(chan bool)
		if j.Schedule.Every > 0 {
			log.Printf("found job %s schedule every: %d seconds\n", j.Name, j.Schedule.Every)
			go c.RepeatTask(j)
		} else {
			log.Printf("found job %s schedule at: %s\n", j.Name, j.Schedule.At)
			go c.ScheduleTask(j)
		}

		total += 1
	}

	c.Slacker.Send(fmt.Sprintf(`%s booted at %s. load %d jobs in %s`, appname, time.Now(), total, time.Now().Sub(t0)))
}

func (c *Endhouse) ScheduleTask(t *Task) {
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
	log.Printf("execute task %s at %s", t0)
	req := c.Client.R()

	for k, v := range t.Executor.Headers {
		req.SetHeader(k, v)
	}

	resp, err := req.Get(t.Executor.URL)

	if err != nil {
		log.Printf("error pushing to url %s\n", t.Executor.URL)
	}
	log.Printf("job %s perform in %s respond %s", t.Name, time.Now().Sub(t0), resp)
	c.Slacker.Send(fmt.Sprintf("job %s perform in %s respond %s", t.Name, time.Now().Sub(t0), resp))
}

func (c *Endhouse) Wait() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	done := make(chan bool, 1)

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

		done <- true
	}()

	<-done
}
