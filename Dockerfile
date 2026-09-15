FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /out/api ./server/cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /out/migrate ./server/tools/migrate

FROM alpine:3.22
RUN addgroup -S app && adduser -S app -G app
WORKDIR /app
# The sqlite default DSN writes var/site.db relative to CWD — give the
# non-root user a writable dir so the image also runs without Postgres.
RUN mkdir -p /app/var && chown app:app /app/var
COPY --from=build /out/api /usr/local/bin/api
COPY --from=build /out/migrate /usr/local/bin/migrate
COPY db ./db
USER app
EXPOSE 8080
CMD ["api"]
