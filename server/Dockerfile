# Game server image.
# Build from the project root:  docker build -f server/Dockerfile .

# Stage 1: compile the Go program
FROM golang:1.27-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /game-server ./server

# Stage 2: small image that only contains the program and the dictionary
FROM alpine:3.20
WORKDIR /app
COPY --from=build /game-server /app/game-server
COPY words_dictionary.json /app/words_dictionary.json
EXPOSE 50051
CMD ["/app/game-server"]
