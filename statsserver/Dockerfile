# Stats server image.
# Build from the project root:  docker build -f statsserver/Dockerfile .

# Stage 1: compile the Go program
FROM golang:1.27-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /stats-server ./statsserver

# Stage 2: small image that only contains the program
FROM alpine:3.20
WORKDIR /app
COPY --from=build /stats-server /app/stats-server
# The SQLite database file is stored in /data (a Docker volume)
RUN mkdir -p /data
EXPOSE 9000
CMD ["/app/stats-server"]
