package scaffold

// templates holds every generated file. Keys are slash-separated paths
// relative to the new project root.
//
// Keep these files boring. A generated project should look like something a
// careful developer wrote by hand, not like generated code.
var templates = map[string]string{

	"go.mod": `module {{.ModulePath}}

go {{.GoVersion}}

require {{.CoreModulePath}} v{{.CoreVersion}}
{{- if .CoreReplacePath}}

// Local development. Remove this block (or delete go.work) before releasing;
// see PRD.md 4.3.
replace {{.CoreModulePath}} => {{.CoreReplacePath}}
{{- end}}
`,

	".env.example": `# Copy to .env and fill in. Never commit .env.
APP_ENV=development
SERVER_HOST=0.0.0.0
SERVER_PORT=8080

# Host ports used by docker compose. Change these if another project already
# owns them.
APP_PORT=8080
DB_PORT_HOST=55432

DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME={{.AppName}}
DB_SSLMODE=disable
DB_MAX_OPEN_CONNS=25
DB_SLOW_QUERY=200ms

HTTP_BODY_LIMIT=8M
HTTP_REQUEST_TIMEOUT=30s
HTTP_SHUTDOWN_TIMEOUT=10s

LOG_LEVEL=info
LOG_FORMAT=json

# Required in production. Comma-separated, never "*".
CORS_ALLOWED_ORIGINS=
{{- if .HasIdentity}}

# Identity. Required in production; at least 32 characters, and the two values
# must differ.
JWT_ACCESS_SECRET=
JWT_REFRESH_SECRET=
JWT_ACCESS_TTL=15m
JWT_REFRESH_TTL=168h
{{- end}}
`,

	".gitignore": `# Secrets
.env
.env.*
!.env.example

# Build output
bin/
*.exe
*.test
*.out

# Local workspace link to the core repo
go.work
go.work.sum

# Data
uploads/
*.log
`,

	"Makefile": `APP := {{.AppName}}

.PHONY: run build test fmt fmt-check vet lint check docker-up docker-down tidy clean

run:
	go run ./cmd/server

build:
	go build -o bin/$(APP) ./cmd/server

test:
	go test ./...

fmt:
	gofmt -w cmd internal

# vendor/ is generated upstream and is not ours to reformat.
fmt-check:
	@files="$$(gofmt -l cmd internal)"; \
	if [ -n "$$files" ]; then echo "gofmt needs to run on:"; echo "$$files"; exit 1; fi

vet:
	go vet ./cmd/... ./internal/...

lint: vet

check: fmt-check vet test

docker-up:
	docker compose up --build

docker-down:
	docker compose down -v

clean:
	rm -rf bin

tidy:
	go mod tidy
`,

	"Dockerfile": `# Build relies on vendor/ so the image never needs network access and never
# needs the core-platform source to be present in the build context.
# corectl new runs ` + "`go mod vendor`" + ` for you.
#
# The Go image tag matches the go directive in go.mod; corectl keeps them in
# sync so a local build and a container build cannot disagree.
FROM golang:{{.GoVersion}}-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
COPY vendor ./vendor
COPY . .
RUN CGO_ENABLED=0 go build -mod=vendor -trimpath -ldflags "-s -w" -o /out/app ./cmd/server

FROM alpine:3.19
RUN adduser -D -u 10001 app
WORKDIR /app
COPY --from=build /out/app /app/app
USER 10001
EXPOSE 8080
HEALTHCHECK --interval=15s --timeout=3s --start-period=10s --retries=3 \
  CMD wget -qO- http://127.0.0.1:8080/healthz || exit 1
ENTRYPOINT ["/app/app"]
`,

	"docker-compose.yml": `# Host ports are overridable so a project never fails to start because another
# one already owns 8080. Set APP_PORT / DB_PORT_HOST in .env if needed.
services:
  postgres:
    image: postgres:15-alpine
    restart: unless-stopped
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
      POSTGRES_DB: {{.AppName}}
    ports:
      - "${DB_PORT_HOST:-55432}:5432"
    volumes:
      - pgdata:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 5s
      timeout: 3s
      retries: 10

  app:
    build: .
    restart: unless-stopped
    ports:
      - "${APP_PORT:-8080}:8080"
    environment:
      APP_ENV: development
      SERVER_HOST: 0.0.0.0
      SERVER_PORT: 8080
      DB_HOST: postgres
      DB_PORT: 5432
      DB_USER: postgres
      DB_PASSWORD: postgres
      DB_NAME: {{.AppName}}
      DB_SSLMODE: disable
      LOG_LEVEL: info
      LOG_FORMAT: json
{{- if .HasIdentity}}
      # Identity secrets. Production refuses to start without them.
      JWT_ACCESS_SECRET: ${JWT_ACCESS_SECRET:-0123456789abcdef0123456789abcdef}
      JWT_REFRESH_SECRET: ${JWT_REFRESH_SECRET:-fedcba9876543210fedcba9876543210}
{{- end}}
    depends_on:
      postgres:
        condition: service_healthy

volumes:
  pgdata:
`,

	"config/config.yaml": `env: development

server:
  host: 0.0.0.0
  port: 8080

db:
  host: localhost
  port: 5432
  user: postgres
  name: {{.AppName}}
  sslmode: disable
  max_open_conns: 25
  slow_query: 200ms

http:
  body_limit: 8M
  request_timeout: 30s
  shutdown_timeout: 10s

log:
  level: info
  format: json

security:
  allowed_origins: []
{{- if .HasIdentity}}

# Identity secrets may also come from JWT_ACCESS_SECRET / JWT_REFRESH_SECRET.
identity:
  access_ttl: 15m
  refresh_ttl: 168h
  password_min_length: 10
  cookie_secure: false
{{- end}}
`,

	"config/modules.yaml": `# Operational module states. A module must also be compiled into the binary.
modules:
{{- range .Modules}}
  {{.}}: ENABLED
{{- end}}
{{- if not .Modules}}
  {}
{{- end}}
`,

	"cmd/server/main.go": `package main

import (
	"log"

	"{{.CoreModulePath}}/contract/core"
{{- range .Modules}}
	"{{$.CoreModulePath}}/mod/{{.}}"
{{- end}}
	"{{.CoreModulePath}}/platform"
)

// version is stamped at build time: -ldflags "-X main.version=v0.1.0".
var version = "0.1.0-dev"

func main() {
	err := platform.Run(core.App{
		Name:    "{{.AppName}}",
		Version: version,
		Modules: []core.Module{
{{- range .Modules}}
			{{.}}.New(),
{{- end}}
		},
		ConfigPath: "config/config.yaml",
	})
	if err != nil {
		log.Fatal(err)
	}
}
`,

	"internal/modules/example/module.go": `// Package example is a placeholder module. It shows the shape of a module and
// can be deleted once the first real one exists.
package example

import (
	"context"
	"net/http"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"

	"{{.CoreModulePath}}/contract/core"
)

// Module is the example module.
type Module struct{}

// New builds the module.
func New() *Module { return &Module{} }

func (m *Module) Name() string           { return "example" }
func (m *Module) Version() string        { return "0.1.0" }
func (m *Module) Dependencies() []string { return nil }

func (m *Module) Migrate(context.Context, *gorm.DB) error { return nil }

func (m *Module) RegisterRoutes(g *echo.Group, _ core.Deps) {
	g.GET("/example/ping", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"pong": "true"})
	})
}

func (m *Module) Start(context.Context) error { return nil }
func (m *Module) Stop(context.Context) error  { return nil }
`,

	"internal/modules/example/module_test.go": `package example

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"

	"{{.CoreModulePath}}/contract/core"
)

func TestPing(t *testing.T) {
	e := echo.New()
	m := New()
	m.RegisterRoutes(e.Group("/api/v1"), core.Deps{})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/example/ping", nil)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}
`,

	"README.md": `# {{.AppName}}

Built on the core platform at level **{{.Level}}**.

## Run

` + "```bash" + `
cp .env.example .env
docker compose up          # app + postgres
` + "```" + `

Health:

- ` + "`GET /healthz`" + ` — process is alive
- ` + "`GET /readyz`" + ` — database is reachable
- ` + "`GET /version`" + ` — build metadata

## Layout

` + "```" + `
cmd/server          entry point (bootstrap only)
internal/modules    application modules
config              config.yaml and modules.yaml
migrations          application migrations
` + "```" + `

## Next

` + "```bash" + `
corectl add module <name>   # scaffold a new domain module
corectl doctor              # check config, migrations, invariants
corectl upgrade             # apply a new core version
` + "```" + `
`,
}
