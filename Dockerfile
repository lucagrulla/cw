FROM golang:1.15-alpine AS build_base

RUN apk add --no-cache git

# Set the Current Working Directory inside the container
WORKDIR /tmp/cw

# We want to populate the module cache based on the go.{mod,sum} files.
COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

# Unit tests
RUN CGO_ENABLED=0 go test -v

# Build the Go app
RUN go build -o ./out/cw .


# Start fresh from a smaller image
FROM alpine:3
RUN apk add ca-certificates

COPY --from=build_base /tmp/cw/out/cw /app/cw

ENTRYPOINT ["/app/cw"]
