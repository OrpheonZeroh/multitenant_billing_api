# Makefile para DGI Service
.PHONY: help build run test clean deps lint docker-build docker-run

# Variables
BINARY_NAME=dgi-service
BINARY_UNIX=$(BINARY_NAME)_unix
MAIN_PATH=./cmd/dgi-service
BUILD_DIR=./build

# Colores para output
GREEN=\033[0;32m
YELLOW=\033[1;33m
RED=\033[0;31m
NC=\033[0m # No Color

# Comando por defecto
.DEFAULT_GOAL := help

help: ## Mostrar esta ayuda
	@echo "$(GREEN)DGI Service - Comandos disponibles:$(NC)"
	@echo ""
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(YELLOW)%-15s$(NC) %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""

deps: ## Instalar dependencias
	@echo "$(GREEN)Instalando dependencias...$(NC)"
	go mod download
	go mod tidy

build: ## Compilar el servicio
	@echo "$(GREEN)Compilando DGI Service...$(NC)"
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)

build-linux: ## Compilar para Linux
	@echo "$(GREEN)Compilando para Linux...$(NC)"
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_UNIX) $(MAIN_PATH)

run: ## Ejecutar el servicio
	@echo "$(GREEN)Ejecutando DGI Service...$(NC)"
	go run $(MAIN_PATH)

run-docker: ## Ejecutar con Docker Compose
	@echo "$(GREEN)Ejecutando con Docker Compose...$(NC)"
	cd db_pg && docker-compose up -d
	@echo "$(YELLOW)Esperando que PostgreSQL esté listo...$(NC)"
	@sleep 5
	@echo "$(GREEN)Ejecutando servicio...$(NC)"
	go run $(MAIN_PATH)

test: ## Ejecutar tests
	@echo "$(GREEN)Ejecutando tests...$(NC)"
	go test -v ./...

test-coverage: ## Ejecutar tests con coverage
	@echo "$(GREEN)Ejecutando tests con coverage...$(NC)"
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Coverage report generado: coverage.html$(NC)"

lint: ## Ejecutar linter
	@echo "$(GREEN)Ejecutando linter...$(NC)"
	golangci-lint run

format: ## Formatear código
	@echo "$(GREEN)Formateando código...$(NC)"
	go fmt ./...
	go vet ./...

clean: ## Limpiar archivos generados
	@echo "$(GREEN)Limpiando...$(NC)"
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	go clean

docker-build: ## Construir imagen Docker
	@echo "$(GREEN)Construyendo imagen Docker...$(NC)"
	docker build -t $(BINARY_NAME):latest .

docker-run: ## Ejecutar contenedor Docker
	@echo "$(GREEN)Ejecutando contenedor Docker...$(NC)"
	docker run -p 8080:8080 --env-file configs/.env $(BINARY_NAME):latest

db-start: ## Iniciar base de datos
	@echo "$(GREEN)Iniciando base de datos...$(NC)"
	cd db_pg && ./start.sh start

db-stop: ## Detener base de datos
	@echo "$(GREEN)Deteniendo base de datos...$(NC)"
	cd db_pg && ./start.sh stop

db-status: ## Estado de la base de datos
	@echo "$(GREEN)Estado de la base de datos:$(NC)"
	cd db_pg && ./start.sh status

db-logs: ## Logs de la base de datos
	@echo "$(GREEN)Logs de la base de datos:$(NC)"
	cd db_pg && ./start.sh logs

dev: ## Modo desarrollo completo
	@echo "$(GREEN)Iniciando modo desarrollo...$(NC)"
	@make db-start
	@sleep 3
	@make run

stop-dev: ## Detener modo desarrollo
	@echo "$(GREEN)Deteniendo modo desarrollo...$(NC)"
	@make db-stop

install-tools: ## Instalar herramientas de desarrollo
	@echo "$(GREEN)Instalando herramientas...$(NC)"
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/go-delve/delve/cmd/dlv@latest
	go install github.com/cosmtrek/air@latest

air: ## Ejecutar con hot reload (requiere air instalado)
	@echo "$(GREEN)Ejecutando con hot reload...$(NC)"
	air

# Comandos de verificación
check: format lint test ## Verificar código completo

# Comandos de despliegue
deploy: build-linux ## Preparar para despliegue
	@echo "$(GREEN)Build para despliegue completado en $(BUILD_DIR)/$(BINARY_UNIX)$(NC)"

# Comandos de base de datos
db-reset: ## Resetear base de datos
	@echo "$(RED)Reseteando base de datos...$(NC)"
	cd db_pg && ./start.sh reset

db-backup: ## Crear backup de la base de datos
	@echo "$(GREEN)Creando backup...$(NC)"
	cd db_pg && ./start.sh backup

# Comandos de monitoreo
monitor: ## Monitorear servicios
	@echo "$(GREEN)Estado de servicios:$(NC)"
	@make db-status
	@echo ""
	@echo "$(GREEN)Estado del servicio:$(NC)"
	@curl -s http://localhost:8081/health | jq . 2>/dev/null || echo "Servicio no está corriendo"

# Comandos de ayuda adicional
setup: ## Configuración inicial completa
	@echo "$(GREEN)Configuración inicial...$(NC)"
	@make install-tools
	@make deps
	@make db-start
	@echo "$(GREEN)Configuración completada!$(NC)"
	@echo "$(YELLOW)Ahora puedes ejecutar: make run$(NC)"

setup-inngest: ## Configurar Inngest paso a paso
	@echo "$(GREEN)Configurando Inngest...$(NC)"
	@./scripts/setup-inngest.sh

# Comandos de limpieza completa
clean-all: clean ## Limpieza completa
	@echo "$(GREEN)Limpiando todo...$(NC)"
	cd db_pg && docker-compose down -v
	docker system prune -f
	@echo "$(GREEN)Limpieza completada!$(NC)"
