version ?= v1.3.0

build:
	GOOS=linux GOARCH=amd64 go build -o out/endhouse .

docker_build:
	docker build -t r.getopty.com/endhouse:$(version) .

docker_tag:
	docker tag r.getopty.com/endhouse:$(version) r.getopty.com/endhouse:latest

docker_push:
	docker push r.getopty.com/endhouse:$(version)
	docker push r.getopty.com/endhouse:latest

docker_release: docker_build docker_tag docker_push
