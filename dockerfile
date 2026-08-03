# syntax=docker/dockerfile:1

FROM golang:1.25-alpine

RUN apk update && apk upgrade && apk add --update alpine-sdk && \
    apk add --update --no-cache bash git openssh make cmake dcron libcap github-cli

WORKDIR /app

COPY . .

RUN make install

RUN chmod +x scripts/entry.sh

CMD scripts/entry.sh
