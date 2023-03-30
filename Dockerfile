FROM golang:1.20 as builder

WORKDIR /endhouse

COPY . ./
RUN \
  CGO_ENABLED=0 go build -o eh .


FROM ubuntu:jammy

WORKDIR /endhouse

RUN apt update && apt install ca-certificates -y

COPY --from=builder /endhouse/eh /bin/

CMD ["/bin/eh"]
