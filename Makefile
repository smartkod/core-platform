.PHONY: run build test clean docker-up docker-down mod-tidy

run:
go run ./cmd/server/main.go

build:
go build -o bin/core-platform ./cmd/server/main.go

test:
go test ./... -v

clean:
rmdir /s /q bin 2>nul
del /q core-platform.exe 2>nul

docker-up:
docker-compose up -d

docker-down:
docker-compose down

mod-tidy:
go mod tidy
go mod download

dev: mod-tidy docker-up run

# Windows için özel komutlar
run-win:
go run ./cmd/server/main.go

build-win:
go build -o bin/core-platform.exe ./cmd/server/main.go
