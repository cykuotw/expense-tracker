build:  
	@go mod tidy
	@go build -o bin/tracker cmd/tracker/main.go

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