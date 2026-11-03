FROM golang:1.26-alpine AS build
WORKDIR /src

# Dependencies are copied first so a source-only change reuses this layer.
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags "-s -w -X core-platform/platform.Commit=${COMMIT:-dev}" \
    -o /out/server ./cmd/server
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /out/corectl ./cmd/corectl

FROM alpine:3.19
RUN apk --no-cache add ca-certificates wget && \
    adduser -D -u 10001 app
WORKDIR /app

COPY --from=build /out/server /app/server
COPY --from=build /out/corectl /app/corectl
COPY config /app/config

# Uploads live under /app/data. The directory is created and handed to the
# runtime user here, because the process runs unprivileged and cannot create it
# itself. Without this the files module fails at startup with "permission denied"
# instead of at first upload, which is the better failure but a confusing one.
RUN mkdir -p /app/data/uploads && chown -R 10001:10001 /app/data

# Never copy .env or .env.example into the image: configuration comes from the
# environment at runtime.
USER 10001
EXPOSE 8080
HEALTHCHECK --interval=15s --timeout=3s --start-period=10s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8080/healthz || exit 1
ENTRYPOINT ["/app/server"]
