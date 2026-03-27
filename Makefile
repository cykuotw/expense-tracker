BUILD_DIR ?= bin
GOOS ?= linux
GOARCH ?= amd64

.PHONY: \
	all \
	app \
	backend \
	build \
	build-deploy-backend \
	build-frontend \
	build-prod \
	deploy \
	destroy \
	edge \
	frontend \
	help \
	infra \
	migrate-down \
	migrate-force \
	migrate-step \
	migrate-to \
	migrate-up \
	migration \
	run \
	test \
	tf-apply \
	tf-init \
	tf-plan

build:
	@go mod tidy
	@go build -o $(BUILD_DIR)/tracker ./backend/cmd/tracker

build-prod:
	@mkdir -p $(BUILD_DIR)
	@go mod tidy
	@GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 go build -ldflags="-s -w -X expense-tracker/backend/config.BuildMode=release" -o $(BUILD_DIR)/tracker ./backend/cmd/tracker

test:
	@go test -v ./...

run: build
	@./$(BUILD_DIR)/tracker

build-frontend:
	@cd frontend && pnpm run build

build-deploy-backend:
	@mkdir -p $(BUILD_DIR)
	@GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 go build -ldflags="-s -w -X expense-tracker/backend/config.BuildMode=release" -o $(BUILD_DIR)/tracker ./backend/cmd/tracker
	@GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 go build -o $(BUILD_DIR)/tracker-migrate ./backend/cmd/migrate
	@GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 go build -o $(BUILD_DIR)/tracker-db-bootstrap ./backend/cmd/db-bootstrap
	@GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=0 go build -o $(BUILD_DIR)/tracker-bootstrap-first-admin ./backend/cmd/bootstrap-first-admin

migration:
	@migrate create --ext sql -dir backend/cmd/migrate/migrations $(filter-out $@,$(MAKECMDGOALS))

migrate-up:
	@go run backend/cmd/migrate/main.go up

migrate-down:
	@go run backend/cmd/migrate/main.go down

migrate-step:
	@go run backend/cmd/migrate/main.go step $(n)

migrate-to:
	@go run backend/cmd/migrate/main.go migrate $(v)

migrate-force:
	@go run backend/cmd/migrate/main.go force $(v)

deploy:
	@./deployment/scripts/deploy.sh $(filter-out $@,$(MAKECMDGOALS))

all app infra frontend backend edge help:
	@:

destroy:
	@./deployment/scripts/destroy.sh

tf-init:
	@terraform -chdir=deployment/tf init -input=false

tf-plan:
	@terraform -chdir=deployment/tf plan

tf-apply:
	@terraform -chdir=deployment/tf apply
