.PHONY: run build test fmt fmt-check vet lint depcheck check corectl docker-up docker-down tidy clean

APP := core-platform

run:
	go run ./cmd/server

build:
	go build -o bin/$(APP) ./cmd/server
	go build -o bin/corectl ./cmd/corectl

# The full gate. CI runs the same steps, so a green local run means green CI.
check: fmt-check vet depcheck test

fmt:
	gofmt -w .

fmt-check:
	@files="$$(gofmt -l .)"; \
	if [ -n "$$files" ]; then echo "gofmt needs to run on:"; echo "$$files"; exit 1; fi

vet:
	go vet ./...

lint: vet

# Enforces the module boundary: platform imports no module, modules import no
# module, contract imports neither.
depcheck:
	go run ./cmd/depcheck

test:
	go test ./...

corectl:
	go run ./cmd/corectl $(ARGS)

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down -v

tidy:
	go mod tidy

clean:
	rm -rf bin
