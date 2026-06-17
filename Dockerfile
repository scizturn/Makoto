FROM golang:1.24-alpine AS build

WORKDIR /src

COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o /out/makoto ./cmd/makoto

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata
RUN addgroup -S makoto && adduser -S makoto -G makoto

WORKDIR /app
COPY --from=build /out/makoto /usr/local/bin/makoto
COPY templates ./templates
COPY db/migrations ./db/migrations

USER makoto

ENTRYPOINT ["makoto"]
