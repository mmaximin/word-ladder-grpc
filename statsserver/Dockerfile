FROM golang:1.27-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /stats-server ./statsserver

FROM alpine:3.20
WORKDIR /app
COPY --from=build /stats-server /app/stats-server
RUN mkdir -p /data
EXPOSE 9000
CMD ["/app/stats-server"]
