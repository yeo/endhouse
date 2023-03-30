version ?= v1.3.0

build:
	GOOS=linux GOARCH=amd64 go build -o out/endhouse .

docker_build:
	docker build -t yeospace/endhouse:$(version) .

docker_tag:
	docker tag yeospace/endhouse:$(version) yeospace/endhouse:latest

docker_push:
	docker push yeospace/endhouse:$(version)
	docker push yeospace/endhouse:latest

docker_release: docker_build docker_tag docker_push
