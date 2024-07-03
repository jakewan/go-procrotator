.DEFAULT_GOAL := local-dev-all

.PHONY: go-doc
go-doc:
	go install golang.org/x/tools/cmd/godoc@latest
	godoc -http :8080

.PHONY: go-fmt
go-fmt:
	$(info Go formatting...)
	gofmt -d -s -w .

.PHONY: go-mod-tidy
go-mod-tidy:
	$(info Tidying module...)
	go mod tidy

.PHONY: go-staticcheck
go-staticcheck:
	$(info Running go-staticcheck...)
	go install honnef.co/go/tools/cmd/staticcheck@latest
	staticcheck -go 1.22 ./...

.PHONY: go-staticcheck-no-install
go-staticcheck-no-install:
	$(info Running go-staticcheck...)
	staticcheck -go 1.22 ./...

.PHONY: go-test
go-test:
	$(info Running tests...)
	go test ./...

.PHONY: go-update-deps
go-update-deps:
	$(info Updating dependencies...)
	go get -u ./...

.PHONY: go-vet
go-vet:
	$(info Vetting...)
	go vet ./...

.PHONY: local-dev-all
local-dev-all: go-fmt go-test go-vet go-staticcheck
