.PHONY: build run test install clean

build:
	go build -o bin/sshtui ./cmd/sshtui

run: build
	./bin/sshtui

test:
	go vet ./...
	go test ./...

# pasang ke ~/.local/bin (pastikan ada di PATH)
install: build
	mkdir -p $(HOME)/.local/bin
	cp bin/sshtui $(HOME)/.local/bin/sshtui

clean:
	rm -rf bin
