ARG GIT_COMMIT=unknown
FROM golang:1.24.3-alpine as builder
ARG GIT_COMMIT
RUN apk add --no-cache git
WORKDIR /siderite
COPY go.mod .
COPY go.sum .

# Get dependancies - will also be cached if we won't change mod/sum
RUN go mod download

# Build
COPY . .
RUN GIT_COMMIT=$GIT_COMMIT go build -ldflags "-X github.com/dip-software/siderite/cmd.GitCommit=${GIT_COMMIT}"

FROM alpine:latest
LABEL maintainer="andy.loafoe@gmail.com"
WORKDIR /app
COPY --from=builder /siderite/siderite /app
ENTRYPOINT ["/app/siderite","runner"]
