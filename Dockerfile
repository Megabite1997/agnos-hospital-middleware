# ---- build ------------------------------------------------------------------
FROM golang:1.24-alpine AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/server ./cmd/server \
 && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/mockhis ./cmd/mockhis

# ---- runtime ----------------------------------------------------------------
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata wget \
 && addgroup -S app && adduser -S -G app app
COPY --from=build /out/server /out/mockhis /usr/local/bin/
USER app
EXPOSE 8080
CMD ["server"]
