.DEFAULT_GOAL := local-dev-all

.PHONY: go-doc
go-doc:
	go install golang.org/x/tools/cmd/godoc@latest
	godoc -http :8080

.PHONY: go-fmt
go-fmt:
	$(info Go formatting...)
	gofmt -d -s -w .

.PHONY: go-lint
go-lint:
	golangci-lint run

.PHONY: go-test
go-test:
	$(info Running tests...)
	go test ./...

.PHONY: local-dev-all
local-dev-all: go-fmt go-test go-lint
