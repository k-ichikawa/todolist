FROM golang:1.24-alpine

WORKDIR /app

ENV GOBIN=/usr/local/bin/
ENV PATH="${PATH}:${GOBIN}"
RUN go install github.com/air-verse/air@latest

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
RUN mkdir -p tmp

EXPOSE 8080