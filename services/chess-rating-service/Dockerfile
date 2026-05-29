FROM golang:1.24-alpine AS build

WORKDIR /app

COPY go.mod ./
COPY internal ./internal
COPY services/chess-rating-service ./services/chess-rating-service
COPY config ./config

RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/chess-rating-service ./services/chess-rating-service

FROM alpine:3.20

WORKDIR /app

COPY --from=build /bin/chess-rating-service /app/chess-rating-service
COPY config ./config

EXPOSE 8081

CMD ["./chess-rating-service"]
