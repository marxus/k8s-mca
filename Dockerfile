#syntax=docker/dockerfile:1.7-labs

FROM golang:1.24-alpine AS build
WORKDIR /app
COPY --parents go.mod go.sum cmd conf pkg ./
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -tags=release -o mca cmd/mca/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=build /app/mca .
ENTRYPOINT ["./mca"]