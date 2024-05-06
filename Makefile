build:
	@go build -o bin/tracker cmd/tracker/main.go

test: 
	@go test -v ./...

run: build
	@./bin/tracker