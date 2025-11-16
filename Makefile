build:
	go mod tidy

	mkdir -p bin
	go clean -cache
	env CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o bin/code_review_service ./cmd/app/app.go

run:
	go run cmd/main.go
