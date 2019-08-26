FROM golang:1.12-alpine AS build
ADD . /src
WORKDIR /src
RUN apk add --no-cache git
RUN go mod download
RUN go build -ldflags "-X main.rev=$(git describe --always --tags)"

FROM alpine:latest
COPY --from=build /src/trello-calendar-proxy /
EXPOSE 8080
ENTRYPOINT ["/trello-calendar-proxy"]
