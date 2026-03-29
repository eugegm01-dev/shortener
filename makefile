.PHONY: build test run docker-up docker-down migrate

build:
    go build -o bin/shortener ./cmd/shortener/

test:
    go test ./... -v

run: build
    ./bin/shortener

docker-up:
    docker-compose up --build

docker-down:
    docker-compose down

migrate:
    @echo "Running migrations..."
    @for file in migrations/*.up.sql; do \
        echo "Applying $$file..."; \
        psql postgres://shortener:shortener@localhost:5432/shortener?sslmode=disable -f $$file; \
    done

lint:
    golangci-lint run

bench:
    go test -bench=. -benchmem ./...

coverage:
    go test ./... -coverprofile=coverage.out
    go tool cover -html=coverage.out