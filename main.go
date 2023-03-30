package main

import (
	"errors"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

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

	c.Client.
		SetTimeout(5 * time.Minute).
		// Set retry count to non zero to enable retries
		SetRetryCount(3).
		SetRetryWaitTime(180 * time.Second).
		// MaxWaitTime can be overridden as well.
		// Default is 2 seconds.
		SetRetryMaxWaitTime(300 * time.Second).
		// SetRetryAfter sets callback to calculate wait time between retries.
		// Default (nil) implies exponential backoff with jitter
		SetRetryAfter(func(client *resty.Client, resp *resty.Response) (time.Duration, error) {
			return 0, errors.New("quota exceeded")
		}).
		AddRetryCondition(
			// RetryConditionFunc type is for retry condition function
			// input: non-nil Response OR request execution error
			func(r *resty.Response, err error) bool {
				return r.StatusCode() == http.StatusTooManyRequests
			},
		)

	c.Run()
	c.Wait()
}
