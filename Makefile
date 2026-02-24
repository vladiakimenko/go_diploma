# Makefile для Blog API

# Переменные
APP_NAME := blog-api
MAIN_PATH := cmd/api/main.go
BUILD_DIR := build
DOCKER_COMPOSE := docker-compose

# Цвета для вывода
GREEN := \033[0;32m
YELLOW := \033[0;33m
RED := \033[0;31m
NC := \033[0m # No Color

.PHONY: help
help: ## Показать справку
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "$(GREEN)%-15s$(NC) %s\n", $$1, $$2}'

.PHONY: run
run: ## Запустить приложение
	@echo "$(GREEN)Starting application...$(NC)"
	go run $(MAIN_PATH)

.PHONY: build
build: ## Собрать бинарный файл
	@echo "$(GREEN)Building application...$(NC)"
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(APP_NAME) $(MAIN_PATH)
	@echo "$(GREEN)Build completed: $(BUILD_DIR)/$(APP_NAME)$(NC)"

.PHONY: test
test: ## Запустить тесты
	@echo "$(GREEN)Running tests...$(NC)"
	go test -v ./...

.PHONY: fmt
fmt: ## Форматировать код
	@echo "$(GREEN)Formatting code...$(NC)"
	go fmt ./...
	@echo "$(GREEN)Formatting completed$(NC)"

.PHONY: deps
deps: ## Скачать зависимости
	@echo "$(GREEN)Downloading dependencies...$(NC)"
	go mod download
	go mod tidy
	@echo "$(GREEN)Dependencies downloaded$(NC)"

.PHONY: docker-up
docker-up: ## Запустить все Docker сервисы
	@echo "$(GREEN)Starting Docker services...$(NC)"
	$(DOCKER_COMPOSE) up -d
	@echo "$(GREEN)Waiting for PostgreSQL to be ready...$(NC)"
	@sleep 5
	@echo "$(GREEN)All services started$(NC)"

.PHONY: docker-down
docker-down: ## Остановить Docker контейнеры
	@echo "$(YELLOW)Stopping Docker containers...$(NC)"
	$(DOCKER_COMPOSE) down
	@echo "$(GREEN)Docker containers stopped$(NC)"

.PHONY: db-shell
db-shell: ## Подключиться к PostgreSQL через psql
	@echo "$(GREEN)Connecting to PostgreSQL...$(NC)"
	docker exec -it blog_postgres psql -U ${POSTGRES_USER} -d ${POSTGRES_DB}

.PHONY: dev
dev: docker-up ## Запустить в режиме разработки (БД + приложение)
	@echo "$(GREEN)Starting development environment...$(NC)"
	@trap '$(MAKE) docker-down' INT TERM; \
	$(MAKE) run

.PHONY: clean
clean: ## Очистить артефакты сборки
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	@rm -rf $(BUILD_DIR)
	@echo "$(GREEN)Clean completed$(NC)"

# Цель по умолчанию
.DEFAULT_GOAL := help
