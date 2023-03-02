# Endhouse

GCP Cloud Run or AWS Beanstalk has a concept where youd efine schedule
job in yaml file, Then they will parse and hit your app at those
URL to perform job, instead of relying on a background job. Now the job
will be process in a HTTP request to these.

End house works exactly the same way. It is a very simple job runner to
offload part of your job to Go and a framework of agnosic language job.

Endhouse is battle tested at [Opty](https://getopty.com) for our
clients.

endhouse is name after the novel of Agatha [Peril at End
House](https://en.wikipedia.org/wiki/Peril_at_End_House)

# Getting started

endhouse parses job definition in a schedule.yaml file so all we need is
ended  house binary itself and schedule file.

Create a schedule.yaml file(refer to below section for its syntax) and run
endhouse

```
# if schedule.yaml is in the current directory, simply invoke it
endhouse

# or point to the schedule somewhere else
endhouse /path/to/schedule.yaml
```

Or using docker

```
# we mount the current directory that has schedule.yaml into /endhouse
directory where we expect the file

docker run -v `pwd`:/endhouse yeospace/endhouse

# or specify path
docker run -v path/to/schedule.yaml:/some/where/config.yaml yeospace/endhouse /some/where/config.yaml
```

# Schedule.yaml syntax

Below are full document of schedule.yaml syntax

```
# http header to include in the request
headers:
  name: value
  # can also use env var
  apitoken: "$apitoken"

notifier:
  slack: slack-web-hook-url

tasks:
- name: shipment
  executor:
    url: a-url-to-be-hit
    method: POST|DELETE # default is get if not specify
    #headers:
    #  - a list of header, to be merge with the global headers to include
    #    in the request

    # beside url, can also run any arbitrary command with:
    # script: |
    #   any shell command or script here

  schedule:
    # does the task need to run when starting sheduler
    # If yes, the job will be fired once right on starting without
    # wainting until the next frequency
    run_on_start: true

    # define when to run, there are 2 syntan, using `every` or `at`
    # endhouse recond use `every` instead of `at`
    every: 5 # in second

    # or using cron syntax
    #at: '* * * * *'
```

# Custom job type

By default endhouse support only two job type. **http** and **shell**
job. To expend its functionality. endhouse follow a design similar to
Terraform where you run a daemon and it communicate with the main
process. A daemon so the job task can also be track and communicate back

