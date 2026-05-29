FROM golang:1.24-alpine AS build

WORKDIR /app

COPY go.mod ./
COPY internal ./internal
COPY services/chess-game-service ./services/chess-game-service
COPY config ./config

RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/chess-game-service ./services/chess-game-service

FROM alpine:3.20

WORKDIR /app

COPY --from=build /bin/chess-game-service /app/chess-game-service
COPY config ./config

EXPOSE 8080

CMD ["./chess-game-service"]
