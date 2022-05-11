FROM golang:1.18

RUN apt-get update && apt-get install -y golint

WORKDIR /go/src/github.com/Mirantis/mcc

ENV GO111MODULE=on

ADD go.mod go.sum ./

RUN go mod download
