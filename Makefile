COVERAGE_FILE ?= coverage.out

# Get all directories in cmd/ as available modules
MODULES := $(notdir $(wildcard cmd/*))

# Help target - display usage information
.PHONY: help
help:
	@printf "Available commands:\n"
	@printf "  \033[36mmake run-bot\033[0m       - Run the bot module locally\n"
	@printf "  \033[36mmake run-scrapper\033[0m  - Run the scrapper module locally\n"
	@printf "  \033[36mmake run-agent\033[0m     - Run the agent module locally\n"
	@printf "  \033[36mmake run-report\033[0m    - Run the report module locally\n"
	@printf "  \033[36mmake tidy\033[0m          - Download and clean dependencies\n"
	@printf "  \033[36mmake mocks\033[0m         - Generate mocks from .mockery.yaml\n"
	@printf "  \033[36mmake build\033[0m         - Build all modules ($(MODULES))\n"
	@$(foreach mod,$(MODULES),printf "  \033[36mmake build_$(mod)\033[0m - Build $(mod) module\n";)
	@printf "  \033[36mmake test\033[0m          - Run all unit tests with coverage report\n"
	@printf "  \033[36mmake integration\033[0m   - Run integration tests\n"

# Быстрый запуск сервисов локально
.PHONY: run-bot
run-bot:
	go run ./cmd/bot

.PHONY: run-scrapper
run-scrapper:
	go run ./cmd/scrapper

.PHONY: run-agent
run-agent:
	go run ./cmd/agent

.PHONY: run-report
run-report:
	go run ./cmd/report

# Загрузка и очистка зависимостей (go.mod / go.sum)
.PHONY: tidy
tidy:
	go mod tidy

.PHONY: mocks
mocks:
	mockery

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
	@go tool cover -func='$(COVERAGE_FILE)' | grep "^total" | tr -s '\t'

## integration: run integration tests
.PHONY: integration
integration:
	@go test -tags=integration ./tests/integration/... -v
