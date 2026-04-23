FROM golang:1.26 AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/server ./cmd/server
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/migrate ./cmd/migrate

FROM gcr.io/distroless/base-debian12
WORKDIR /app

ENV HTTP_PORT=8080
ENV APP_ENV=cloud

COPY --from=build /out/server /app/server
COPY --from=build /out/migrate /app/migrate
COPY --from=build /src/internal/platform/migrate/migrations /app/migrations

EXPOSE 8080
ENTRYPOINT ["/app/server"]
