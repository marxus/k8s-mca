#syntax=docker/dockerfile:1.7-labs
ARG BUILDSTAGE=build

# Stage for CI pre-built binaries
FROM scratch AS prebuilt
ARG TARGETARCH
WORKDIR /app
COPY mca-${TARGETARCH} ./mca

# Stage for local builds
FROM golang:1.24-alpine3.22 AS build
WORKDIR /app
COPY --parents go.mod go.sum cmd conf pkg ./
RUN go mod download
RUN CGO_ENABLED=0 go build -tags=release -o mca cmd/mca/main.go

FROM ${BUILDSTAGE} AS binary
FROM alpine:3.22
WORKDIR /app
COPY --from=binary /app/mca ./mca

USER 999
ENTRYPOINT ["./mca"]