build:  
	@go mod tidy
	@go build -o bin/tracker cmd/tracker/main.go

build-prod:
	@go mod tidy
	@go build -ldflags="-s -w -X expense-tracker/config.BuildMode=release" -o bin/tracker-prod cmd/tracker/main.go

test: 
	@go test -v ./...

run: build
	@./bin/tracker

migration:
	@migrate create --ext sql -dir cmd/migrate/migrations $(filter-out $@,$(MAKECMDGOALS))

migrate-up:
	@go run cmd/migrate/main.go up

migrate-down:
	@go run cmd/migrate/main.go down

migrate-step:
	@go run cmd/migrate/main.go step $(n)

migrate-to:
	@go run cmd/migrate/main.go migrate $(v)

migrate-force:
	@go run cmd/migrate/main.go force $(v)