COVERAGE_FILE ?= coverage.out

# Get all directories in cmd/ as available modules
MODULES := $(notdir $(wildcard cmd/*))

# Help target - display usage information
.PHONY: help
help:
	@echo "Available commands:"
	@echo "  \033[36mmake run\033[0m   - Run the bot module locally (for development)"
	@echo "  \033[36mmake tidy\033[0m  - Download and clean dependencies"
	@echo "  \033[36mmake build\033[0m - Build all modules ($(MODULES))"
	@$(foreach mod,$(MODULES),echo "  \033[36mmake build_$(mod)\033[0m - Build $(mod) module";)
	@echo "  \033[36mmake test\033[0m  - Run all tests with coverage report"

# Быстрый запуск бота для локальной разработки
.PHONY: run
run:
	go run ./cmd/bot/main.go

# Загрузка и очистка зависимостей (go.mod / go.sum)
.PHONY: tidy
tidy:
	go mod tidy

.PHONY: build
build:
	@echo "Building all modules: $(MODULES)"
	@mkdir -p bin
	@$(foreach mod,$(MODULES),echo "Building module: $(mod)"; go build -o ./bin/$(mod) ./cmd/$(mod);)

# Convenience targets for building individual modules
.PHONY: $(addprefix build_,$(MODULES))
$(addprefix build_,$(MODULES)):
	@modulename=$(subst build_,,$@); \
	echo "Building module: $$modulename"; \
	mkdir -p bin; \
	go build -o ./bin/$$modulename ./cmd/$$modulename

## test: run all tests
.PHONY: test
test:
	@go test -coverpkg='./...' -count=1 -coverprofile='$(COVERAGE_FILE)' ./...
	@go tool cover -func='$(COVERAGE_FILE)' | grep ^total | tr -s '\t'
