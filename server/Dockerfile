FROM golang:1.27-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /game-server ./server

FROM alpine:3.20
WORKDIR /app
COPY --from=build /game-server /app/game-server
COPY words_dictionary.json /app/words_dictionary.json
EXPOSE 50051
CMD ["/app/game-server"]
