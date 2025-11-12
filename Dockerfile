# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o upload-drive-script ./cmd/

FROM alpine:3.20
WORKDIR /app

COPY --from=builder /app/upload-drive-script .
COPY --from=builder /app/credentials.json .
RUN mkdir -p /app/upload
RUN apk add --no-cache ffmpeg

EXPOSE 3000

CMD ["./upload-drive-script"]
