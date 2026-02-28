FROM golang:1.22-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /webstalk ./cmd/webstalk

FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata chromium

COPY --from=builder /webstalk /usr/local/bin/webstalk

ENV ROD_BROWSER=/usr/bin/chromium-browser

ENTRYPOINT ["webstalk"]
