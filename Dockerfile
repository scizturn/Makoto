FROM golang:1.22-alpine AS build

WORKDIR /src

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN go build -o /out/makoto ./cmd/makoto

FROM alpine:3.20

RUN addgroup -S makoto && adduser -S makoto -G makoto

WORKDIR /app
COPY --from=build /out/makoto /usr/local/bin/makoto

USER makoto

ENTRYPOINT ["makoto"]
