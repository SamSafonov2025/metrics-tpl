.PHONY: proto
proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/metrics.proto

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  proto  - Generate Go code from proto files"
	@echo ""
	@echo "Requirements:"
	@echo "  - protoc (install from https://grpc.io/docs/protoc-installation/)"
	@echo "  - protoc-gen-go (go install google.golang.org/protobuf/cmd/protoc-gen-go@latest)"
	@echo "  - protoc-gen-go-grpc (go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest)"
