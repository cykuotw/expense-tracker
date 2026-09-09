BUILD_DIR ?= bin
GOOS ?= linux
GOARCH ?= amd64
CI_ENV_FILE ?= backend/.env.ci
GOVULNCHECK_VERSION ?= v1.1.4
.PHONY: \
	all \
	app \
	backend \
	build \
	build-deploy-backend \
	build-frontend \
	build-prod \
	check \
	check-backend \
	check-components \
	check-db \
	check-frontend \
	check-go-static \
	check-python \
	check-terraform \
	deploy \
	deploy-serverful \
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

check:
	@CI_ENV_FILE="$(CI_ENV_FILE)" ./deployment/ci/ci-db.sh run make --no-print-directory check-components

check-components: check-backend check-python check-terraform check-frontend

check-db:
	@CI_ENV_FILE="$(CI_ENV_FILE)" ./deployment/ci/ci-db.sh prepare

check-go-static:
	@files="$$(gofmt -l $$(git ls-files '*.go'))"; test -z "$$files" || { printf 'Go files need formatting:\n%s\n' "$$files" >&2; exit 1; }
	@go mod tidy -diff
	@go vet ./...
	@go run golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION) ./...

check-backend: check-db check-go-static
	@CI_ENV_FILE="$(CI_ENV_FILE)" ./deployment/ci/ci-db.sh test ./...

check-python:
	@uv run --python 3.14 python -m unittest discover -s deployment/serverless/tests -p 'test_*.py'

check-terraform:
	@terraform -chdir=deployment/serverless/infrastructure/tf fmt -check -recursive
	@terraform -chdir=deployment/serverless/infrastructure/tf init -backend=false -input=false
	@terraform -chdir=deployment/serverless/infrastructure/tf validate

check-frontend:
	@test "$$(node --version)" = "v$$(cat frontend/.node-version)" || { echo "Node must match frontend/.node-version" >&2; exit 1; }
	@expected="$$(node -p "require('./frontend/package.json').packageManager.replace(/^pnpm@/, '')")"; test "$$(pnpm --version)" = "$$expected" || { echo "pnpm must match frontend/package.json" >&2; exit 1; }
	@pnpm --dir frontend install --frozen-lockfile
	@pnpm --dir frontend run lint
	@pnpm --dir frontend run test:run
	@pnpm --dir frontend run build

run: build
	@./$(BUILD_DIR)/tracker

build-frontend:
	@pnpm --dir frontend install --frozen-lockfile
	@pnpm --dir frontend run build

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

deploy-serverful:
	@./deployment/serverful/scripts/deploy.sh $(filter-out $@,$(MAKECMDGOALS))

all app infra frontend backend edge help:
	@:

destroy:
	@./deployment/serverful/scripts/destroy.sh

tf-init:
	@terraform -chdir=deployment/serverful/infrastructure/tf init -input=false

tf-plan:
	@terraform -chdir=deployment/serverful/infrastructure/tf plan

tf-apply:
	@terraform -chdir=deployment/serverful/infrastructure/tf apply

deploy:
	@python_path="$$(uv python find 3.14)"; "$$python_path" deployment/serverless/deploy.py --action "$(if $(ACTION),$(ACTION),auto)" --scope "$(if $(SCOPE),$(SCOPE),all)"
