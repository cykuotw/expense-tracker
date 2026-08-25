BUILD_DIR ?= bin
GOOS ?= linux
GOARCH ?= amd64
WORKER_TF_DIR := deployment/backend/serverless/worker/tf
WORKER_TF_PLAN ?= $(WORKER_TF_DIR)/worker.tfplan
BOOTSTRAP_TF_DIR := deployment/backend/serverless/bootstrap/tf
BOOTSTRAP_TF_PLAN ?= $(BOOTSTRAP_TF_DIR)/bootstrap.tfplan
SERVERLESS_SCRIPTS_DIR := deployment/backend/serverless/scripts
SERVERLESS_AWS_REGION = $(strip $(or $(AWS_REGION),$(AWS_DEFAULT_REGION),$(TF_VAR_aws_region),$(shell aws configure get region 2>/dev/null)))

.PHONY: \
	all \
	app \
	backend \
	bootstrap-build \
	bootstrap-configure \
	bootstrap-check-secret-boundary \
	bootstrap-invoke \
	bootstrap-tf-apply \
	bootstrap-tf-destroy \
	bootstrap-tf-init \
	bootstrap-tf-plan \
	bootstrap-update-code \
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
	lambda-concurrency-check \
	migrate-down \
	migrate-force \
	migrate-step \
	migrate-to \
	migrate-up \
	migration \
	phase-9-5-check \
	serverless-region-check \
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
	worker-activate \
	worker-check-boundaries \
	worker-check-authorizer \
	worker-check-secret-boundary \
	worker-check-static-authorizer \
	worker-configure \
	worker-google-exchange \
	worker-health \
	worker-tf-apply \
	worker-tf-destroy \
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
	@terraform -chdir=$(WORKER_TF_DIR) init -backend=false -input=false

worker-tf-plan: worker-build
	@TF_DIR="$(WORKER_TF_DIR)" $(SERVERLESS_SCRIPTS_DIR)/check-runtime-secret-boundary.sh
	@terraform -chdir=$(WORKER_TF_DIR) plan -refresh=false -input=false -out="$(abspath $(WORKER_TF_PLAN))"
	@TF_DIR="$(WORKER_TF_DIR)" $(SERVERLESS_SCRIPTS_DIR)/check-runtime-secret-boundary.sh

worker-tf-apply:
	@test -f "$(WORKER_TF_PLAN)" || { printf 'error: saved worker plan not found; run make worker-tf-plan first\n' >&2; exit 1; }
	@TF_DIR="$(WORKER_TF_DIR)" $(SERVERLESS_SCRIPTS_DIR)/check-runtime-secret-boundary.sh
	@terraform -chdir=$(WORKER_TF_DIR) show -json "$(abspath $(WORKER_TF_PLAN))" | jq -e '.variables.reserved_concurrency.value == 0' >/dev/null || { printf 'error: worker Terraform plans must keep reserved concurrency 0; use make worker-activate after bootstrap\n' >&2; exit 1; }
	@terraform -chdir=$(WORKER_TF_DIR) apply -input=false "$(abspath $(WORKER_TF_PLAN))"
	@rm -f "$(WORKER_TF_PLAN)"

worker-tf-destroy:
	@TF_DIR="$(WORKER_TF_DIR)" FUNCTION_OUTPUT=worker_function_name SECURITY_GROUP_OUTPUT=worker_security_group_id $(SERVERLESS_SCRIPTS_DIR)/destroy-lambda-stack.sh

worker-update-code:
	@./deployment/backend/serverless/worker/scripts/update-code.sh

worker-configure:
	@./deployment/backend/serverless/worker/scripts/configure-runtime.sh

worker-activate:
	@./deployment/backend/serverless/worker/scripts/activate.sh

worker-check-secret-boundary:
	@TF_DIR="$(WORKER_TF_DIR)" $(SERVERLESS_SCRIPTS_DIR)/check-runtime-secret-boundary.sh

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

bootstrap-build:
	@./deployment/backend/serverless/bootstrap/scripts/build-bootstrap.sh

serverless-region-check:
	@test -n "$(SERVERLESS_AWS_REGION)" || { printf 'error: set AWS_REGION, AWS_DEFAULT_REGION, TF_VAR_aws_region, or configure the AWS CLI region\n' >&2; exit 1; }

bootstrap-tf-init: serverless-region-check
	@AWS_REGION="$(SERVERLESS_AWS_REGION)" TF_VAR_aws_region="$(SERVERLESS_AWS_REGION)" terraform -chdir=$(BOOTSTRAP_TF_DIR) init -backend=false -input=false

bootstrap-tf-plan: bootstrap-build serverless-region-check
	@AWS_REGION="$(SERVERLESS_AWS_REGION)" TF_VAR_aws_region="$(SERVERLESS_AWS_REGION)" REQUIRED_RESERVED_CONCURRENCY=4 $(SERVERLESS_SCRIPTS_DIR)/check-lambda-concurrency.sh
	@TF_DIR="$(BOOTSTRAP_TF_DIR)" $(SERVERLESS_SCRIPTS_DIR)/check-runtime-secret-boundary.sh
	@AWS_REGION="$(SERVERLESS_AWS_REGION)" TF_VAR_aws_region="$(SERVERLESS_AWS_REGION)" terraform -chdir=$(BOOTSTRAP_TF_DIR) plan -refresh=false -input=false -out="$(abspath $(BOOTSTRAP_TF_PLAN))"
	@TF_DIR="$(BOOTSTRAP_TF_DIR)" $(SERVERLESS_SCRIPTS_DIR)/check-runtime-secret-boundary.sh

bootstrap-tf-apply: serverless-region-check
	@test -f "$(BOOTSTRAP_TF_PLAN)" || { printf 'error: saved bootstrap plan not found; run make bootstrap-tf-plan first\n' >&2; exit 1; }
	@AWS_REGION="$(SERVERLESS_AWS_REGION)" TF_VAR_aws_region="$(SERVERLESS_AWS_REGION)" REQUIRED_RESERVED_CONCURRENCY=4 $(SERVERLESS_SCRIPTS_DIR)/check-lambda-concurrency.sh
	@TF_DIR="$(BOOTSTRAP_TF_DIR)" $(SERVERLESS_SCRIPTS_DIR)/check-runtime-secret-boundary.sh
	@AWS_REGION="$(SERVERLESS_AWS_REGION)" TF_VAR_aws_region="$(SERVERLESS_AWS_REGION)" terraform -chdir=$(BOOTSTRAP_TF_DIR) apply -input=false "$(abspath $(BOOTSTRAP_TF_PLAN))"
	@rm -f "$(BOOTSTRAP_TF_PLAN)"

bootstrap-tf-destroy: serverless-region-check
	@AWS_REGION="$(SERVERLESS_AWS_REGION)" TF_VAR_aws_region="$(SERVERLESS_AWS_REGION)" TF_DIR="$(BOOTSTRAP_TF_DIR)" FUNCTION_OUTPUT=bootstrap_function_name SECURITY_GROUP_OUTPUT=bootstrap_security_group_id $(SERVERLESS_SCRIPTS_DIR)/destroy-lambda-stack.sh

bootstrap-update-code: serverless-region-check
	@AWS_REGION="$(SERVERLESS_AWS_REGION)" TF_VAR_aws_region="$(SERVERLESS_AWS_REGION)" ./deployment/backend/serverless/bootstrap/scripts/update-code.sh

bootstrap-configure: serverless-region-check
	@AWS_REGION="$(SERVERLESS_AWS_REGION)" TF_VAR_aws_region="$(SERVERLESS_AWS_REGION)" ./deployment/backend/serverless/bootstrap/scripts/configure-runtime.sh

bootstrap-check-secret-boundary:
	@TF_DIR="$(BOOTSTRAP_TF_DIR)" $(SERVERLESS_SCRIPTS_DIR)/check-runtime-secret-boundary.sh

bootstrap-invoke: serverless-region-check
	@AWS_REGION="$(SERVERLESS_AWS_REGION)" TF_VAR_aws_region="$(SERVERLESS_AWS_REGION)" ./deployment/backend/serverless/bootstrap/scripts/invoke.sh

lambda-concurrency-check: serverless-region-check
	@AWS_REGION="$(SERVERLESS_AWS_REGION)" TF_VAR_aws_region="$(SERVERLESS_AWS_REGION)" REQUIRED_RESERVED_CONCURRENCY=4 $(SERVERLESS_SCRIPTS_DIR)/check-lambda-concurrency.sh

phase-9-5-check:
	@$(SERVERLESS_SCRIPTS_DIR)/check-phase-9-5.sh
