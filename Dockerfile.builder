# syntax = docker/dockerfile:experimental
FROM golang:1.21

RUN apt-get update

RUN mkdir -p -m 0600 ~/.ssh && ssh-keyscan -t rsa github.com >> ~/.ssh/known_hosts

RUN git config --global url."git@github.com:".insteadOf "https://github.com/"

WORKDIR /go/src/github.com/Mirantis/mcc

ENV GO111MODULE=on
ENV CGO_ENABLED=0

ADD go.mod go.sum ./

RUN --mount=type=ssh go mod download
