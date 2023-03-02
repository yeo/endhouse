# Endhouse

Endhouse is a very simple job runner to offload parse of your job to Go
and a framework of agnosic language job.

Endhouse is battle tested at [Opty](https://getopty.com) for our
clients.

# Getting start

Create a schedule.yaml file and run endhouse

``
endhouse schedule.yaml`
```

Or using docker

```
docker run --rm getopty/
```

# Custom job type

By default endhouse support only two job type. **http** and **shell**
job. To expend its functionality. endhouse follow a design similar to
Terraform where you run a daemon and it communicate with the main
process. A daemon so the job task can also be track and communicate back
