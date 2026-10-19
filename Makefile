.PHONY: build run test art review hardening test-terminal
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null | sed 's/^v//')
build:
	go build -ldflags "-X main.version=$(or $(VERSION),dev)" -o bin/sea-battle .
run: build
	./bin/sea-battle
test:
	go test ./...
	go vet ./...
art:
	uv run --with pillow tools/make_art.py
review: build
	uv run --with pillow tools/review.py

hardening:
	go test -race ./...
	go vet ./...
	go test -race ./internal/peer ./internal/session ./internal/netsetup -run 'Test(StressRepeated|EndpointUpdates|ShotCommitted|QueuedIntent|RapidStart|CloseJoins|LostRenewal|ExpiredLease|ParentCancellation|LocalInterface|SameClientResume)' -count=5
	go test ./internal/peer -run '^$$' -fuzz '^FuzzInvitation$$' -fuzztime=100000x -parallel=2
	go test ./internal/peer -run '^$$' -fuzz '^FuzzWireMessages$$' -fuzztime=100000x -parallel=2
	go test ./internal/peer -run '^$$' -fuzz '^FuzzCommandsPreserveRules$$' -fuzztime=100000x -parallel=2
test-terminal: build
	uv run --with pillow --with pyte tools/solo_pty_review.py
	uv run --with pillow --with pyte tools/play_pty_review.py
	uv run --with pillow --with pyte tools/pty_review.py

.PHONY: preview-results
preview-results: build
	./bin/sea-battle --preview-results

.PHONY: test-packaging
test-packaging:
	python3 -m unittest discover -s packaging/release -p 'test_*.py'
	python3 -m unittest discover -s packaging/homebrew -p 'test_*.py'
	python3 -m unittest discover -s packaging/itch -p 'test_*.py'
