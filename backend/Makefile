MIGRATE ?= $(shell go env GOPATH)/bin/migrate
MIGRATIONS_PATH ?= migrations
DATABASE_URL ?= postgres://ats_user:ats_password@localhost:5435/ats_db?sslmode=disable
BASE_URL ?= http://localhost:8000

.PHONY: migrate-install migrate-up migrate-down migrate-version migrate-force run test-auth test-jobs test-candidates

migrate-install:
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

migrate-up:
	@test -x "$(MIGRATE)" || (echo "Run: make migrate-install" && exit 1)
	$(MIGRATE) -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" up

migrate-down:
	@test -x "$(MIGRATE)" || (echo "Run: make migrate-install" && exit 1)
	$(MIGRATE) -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" down 1

migrate-version:
	@test -x "$(MIGRATE)" || (echo "Run: make migrate-install" && exit 1)
	$(MIGRATE) -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" version

migrate-force:
	@test -n "$(VERSION)" || (echo "VERSION is required, e.g. make migrate-force VERSION=1" && exit 1)
	@test -x "$(MIGRATE)" || (echo "Run: make migrate-install" && exit 1)
	$(MIGRATE) -path $(MIGRATIONS_PATH) -database "$(DATABASE_URL)" force $(VERSION)

run:
	go run ./cmd/api

test-auth:
	@chmod +x scripts/test-auth.sh
	BASE_URL="$(BASE_URL)" ./scripts/test-auth.sh

test-jobs:
	@chmod +x scripts/test-jobs.sh
	BASE_URL="$(BASE_URL)" ./scripts/test-jobs.sh

test-candidates:
	@chmod +x scripts/test-candidates.sh
	BASE_URL="$(BASE_URL)" ./scripts/test-candidates.sh
