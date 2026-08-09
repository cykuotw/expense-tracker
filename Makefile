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
	tf-plan \
	postgres-setup \
	postgres-tf-apply \
	postgres-tf-init \
	postgres-tf-plan \
	postgres-tf-destroy \
	worker-build \
	worker-check-boundaries \
	worker-check-authorizer \
	worker-check-static-authorizer \
	worker-configure \
	worker-google-exchange \
	worker-health \
	worker-tf-apply \
	worker-tf-init \
	worker-tf-plan \
	worker-update-code

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
	@./deployment/backend/serverful/scripts/deploy.sh $(filter-out $@,$(MAKECMDGOALS))

all app infra frontend backend edge help:
	@:

destroy:
	@./deployment/backend/serverful/scripts/destroy.sh

tf-init:
	@terraform -chdir=deployment/backend/serverful/tf init -input=false

tf-plan:
	@terraform -chdir=deployment/backend/serverful/tf plan

tf-apply:
	@terraform -chdir=deployment/backend/serverful/tf apply

postgres-tf-init:
	@terraform -chdir=deployment/backend/serverless/postgres/tf init -backend=false -input=false

postgres-tf-plan:
	@terraform -chdir=deployment/backend/serverless/postgres/tf plan

postgres-tf-apply:
	@terraform -chdir=deployment/backend/serverless/postgres/tf apply

postgres-setup:
	@./deployment/backend/serverless/postgres/scripts/setup-postgres.sh

postgres-tf-destroy:
	@terraform -chdir=deployment/backend/serverless/postgres/tf destroy

worker-build:
	@./deployment/backend/serverless/worker/scripts/build-worker.sh

worker-tf-init:
	@terraform -chdir=deployment/backend/serverless/worker/tf init -backend=false -input=false

worker-tf-plan: worker-build
	@terraform -chdir=deployment/backend/serverless/worker/tf plan -input=false

worker-tf-apply: worker-build
	@terraform -chdir=deployment/backend/serverless/worker/tf apply

worker-update-code:
	@./deployment/backend/serverless/worker/scripts/update-code.sh

worker-configure:
	@./deployment/backend/serverless/worker/scripts/configure-runtime.sh

worker-check-authorizer:
	@./deployment/backend/serverless/worker/scripts/check-google-authorizer.sh

worker-check-static-authorizer:
	@./deployment/backend/serverless/worker/scripts/check-static-authorizer.sh

worker-check-boundaries:
	@./deployment/backend/serverless/worker/scripts/check-password-boundary.sh

worker-health:
	@./deployment/backend/serverless/worker/scripts/check-health.sh

worker-google-exchange:
	@./deployment/backend/serverless/worker/scripts/check-google-exchange.sh
