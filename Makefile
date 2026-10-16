.PHONY: build run test art review hardening test-terminal
build:
	go build -o bin/sea-battle .
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
	uv run --with pillow --with pyte tools/play_pty_review.py
	uv run --with pillow --with pyte tools/pty_review.py
