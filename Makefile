.PHONY: lint fmt fmt-check test test-coverage vet all ci

# Путь к бинарникам (если используешь go install)
GOBIN := $(shell go env GOPATH)/bin

# Бинарники
GOLANGCI_LINT := $(GOBIN)/golangci-lint
GOIMPORTS := $(GOBIN)/goimports

# Убедимся, что golangci-lint установлен (совместимо с Go 1.24+)
$(GOLANGCI_LINT):
	@echo "Installing golangci-lint v2.4.0..."
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | \
	GOBIN=$(GOBIN) sh -s -- -b $(GOBIN) v2.4.0

# Убедимся, что goimports установлен
$(GOIMPORTS):
	@echo "Installing goimports..."
	@go install golang.org/x/tools/cmd/goimports@latest

# Lint: проверка стиля и ошибок
lint: $(GOLANGCI_LINT)
	@echo "Running golangci-lint..."
	$(GOLANGCI_LINT) run --fix --config .golangci-lint.yaml ./...

# fmt: автоматическое форматирование
fmt: $(GOIMPORTS)
	@echo "Running go fmt and goimports..."
	@find . -name "*.go" -not -path "./vendor/*" -exec gofmt -s -w {} \;
	@$(GOIMPORTS) -local github.com/shuldan/framework -w $$(find . -name "*.go" -not -path "./vendor/*")

# fmt-check: проверить, нет ли неотформатированных файлов (для CI)
fmt-check: $(GOIMPORTS)
	@echo "Checking code formatting..."
	@gofmt -s -l . | grep -v vendor | grep .go && echo "❌ Unformatted files found" && exit 1 || echo "✅ All files are formatted"
	@$(GOIMPORTS) -local github.com/shuldan/framework -l . | grep -v vendor && echo "❌ Unformatted imports" && exit 1 || echo "✅ Imports are clean"

# vet: базовая проверка Go
vet:
	@echo "Running go vet..."
	@go vet ./...

# test: запуск тестов
test:
	@echo "Running tests..."
	@go test -race ./... -count=1

# test-coverage: с отчётом о покрытии
test-coverage:
	@echo "Running tests with coverage..."
	@go test ./... -coverprofile=coverage.out -covermode=atomic
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# all: полная проверка (для CI)
all: fmt-check vet lint test

# ci: полная проверка для CI
ci: fmt-check vet lint test-coverage
	@echo "✅ All CI checks passed."

# Убедимся, что после всех проверок нет изменённых файлов (например, gofmt)
verify-no-changes:
	@git diff --exit-code || (echo "❌ Code changes detected after checks. Run 'make fmt' and commit again."; exit 1)