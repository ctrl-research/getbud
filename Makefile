.PHONY: db run web seed generate test test-db build docker clean

# Start postgres only (for local development)
db:
	docker compose up -d postgres

# Regenerate the sqlc store code from internal/store/queries/*.sql
generate:
	go tool sqlc generate

# Run the Go server against the compose postgres (:8081 to avoid clashing
# with other local apps on :8080)
run:
	GETBUD_DATABASE_URL=$${GETBUD_DATABASE_URL:-postgres://getbud:getbud@localhost:5433/getbud?sslmode=disable} \
	GETBUD_ADDR=$${GETBUD_ADDR:-:8081} \
	GETBUD_LOCAL_AUTH=$${GETBUD_LOCAL_AUTH:-true} \
		go run ./cmd/server

# Create/reset the local dev user (dev@getbud.local / getbud-dev).
# Sign in with it by running the server with GETBUD_LOCAL_AUTH=true.
seed:
	GETBUD_DATABASE_URL=$${GETBUD_DATABASE_URL:-postgres://getbud:getbud@localhost:5433/getbud?sslmode=disable} \
		go run ./cmd/server seed

# Vite dev server on :5173, proxying /api and /auth to :8080
web:
	cd web && npm run dev

test:
	go vet ./...
	go test ./...

# Like test, but includes postgres-backed store tests (needs `make db` running;
# creates the getbud_test database automatically)
test-db:
	go vet ./...
	GETBUD_TEST_DATABASE_URL=$${GETBUD_TEST_DATABASE_URL:-postgres://getbud:getbud@localhost:5433/getbud_test?sslmode=disable} \
		go test ./...

# Build the web UI and the server binary with the UI embedded
build:
	cd web && npm run build
	rm -rf internal/webui/dist
	cp -R web/dist internal/webui/dist
	go build -tags embedwebui -o bin/getbud ./cmd/server

docker:
	docker build -t getbud .

clean:
	rm -rf bin internal/webui/dist web/dist
