build:
	CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o bin/agenda ./cmd/main.go
