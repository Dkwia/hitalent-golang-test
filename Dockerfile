FROM golang:1.22-alpine AS builder
WORKDIR /app
RUN apk add --no-cache git
COPY go.mod .
RUN go mod download
COPY ../../Downloads/Telegram%20Desktop/organization-api .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/server ./cmd/api

FROM alpine:3.20
WORKDIR /app
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/server /app/server
COPY migrations /app/migrations
EXPOSE 8080
CMD ["/app/server"]
