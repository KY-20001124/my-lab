.PHONY: fmt vet test check check-architecture

fmt:
	go fmt ./...

vet:
	go vet ./...

test:
	go test ./... -count=1

check-architecture:
	bash scripts/check-architecture.sh

check: vet test check-architecture
