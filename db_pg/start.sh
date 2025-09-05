#!/bin/bash

# Script de inicio rápido para DGI Service Database
# Uso: ./start.sh [start|stop|restart|status|logs|reset|backup|restore]

set -e

# Colores para output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Función para mostrar mensajes
print_message() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_header() {
    echo -e "${BLUE}================================${NC}"
    echo -e "${BLUE}  DGI Service Database Manager${NC}"
    echo -e "${BLUE}================================${NC}"
}

# Función para verificar Docker
check_docker() {
    if ! command -v docker &> /dev/null; then
        print_error "Docker no está instalado. Por favor instala Docker primero."
        exit 1
    fi
    
    if ! command -v docker-compose &> /dev/null; then
        print_error "Docker Compose no está instalado. Por favor instala Docker Compose primero."
        exit 1
    fi
    
    if ! docker info &> /dev/null; then
        print_error "Docker no está corriendo. Por favor inicia Docker primero."
        exit 1
    fi
}

# Función para verificar archivos necesarios
check_files() {
    local required_files=("docker-compose.yml" "postgresql.conf" "init/01-init-schema.sql" "init/02-seed-data.sql" "init/03-environment.sql" "init/04-verify-setup.sql")
    
    for file in "${required_files[@]}"; do
        if [[ ! -f "$file" ]]; then
            print_error "Archivo requerido no encontrado: $file"
            exit 1
        fi
    done
    
    print_message "Todos los archivos requeridos están presentes"
}

# Función para iniciar servicios
start_services() {
    print_message "Iniciando servicios DGI..."
    docker-compose up -d
    
    print_message "Esperando que PostgreSQL esté listo..."
    local max_attempts=30
    local attempt=1
    
    while [[ $attempt -le $max_attempts ]]; do
        if docker-compose exec -T postgres pg_isready -U dgi_user -d dgi_service &> /dev/null; then
            print_message "PostgreSQL está listo!"
            break
        fi
        
        if [[ $attempt -eq $max_attempts ]]; then
            print_error "PostgreSQL no se pudo iniciar en el tiempo esperado"
            docker-compose logs postgres
            exit 1
        fi
        
        echo -n "."
        sleep 2
        ((attempt++))
    done
    
    print_message "Verificando setup de la base de datos..."
    if docker-compose exec -T postgres psql -U dgi_user -d dgi_service -c "SELECT COUNT(*) FROM emitters;" &> /dev/null; then
        print_message "Base de datos configurada correctamente"
    else
        print_warning "La base de datos puede no estar completamente inicializada"
    fi
    
    print_message "Servicios iniciados exitosamente!"
    print_message "PostgreSQL: localhost:5432"
    print_message "Adminer: http://localhost:8080"
    print_message "Redis: localhost:6379"
}

# Función para detener servicios
stop_services() {
    print_message "Deteniendo servicios DGI..."
    docker-compose down
    print_message "Servicios detenidos"
}

# Función para reiniciar servicios
restart_services() {
    print_message "Reiniciando servicios DGI..."
    stop_services
    sleep 2
    start_services
}

# Función para mostrar estado
show_status() {
    print_message "Estado de los servicios:"
    docker-compose ps
    
    echo ""
    print_message "Uso de recursos:"
    docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}\t{{.BlockIO}}"
}

# Función para mostrar logs
show_logs() {
    print_message "Mostrando logs de PostgreSQL (Ctrl+C para salir):"
    docker-compose logs -f postgres
}

# Función para resetear base de datos
reset_database() {
    print_warning "Esta acción eliminará TODOS los datos de la base de datos"
    read -p "¿Estás seguro? (y/N): " -n 1 -r
    echo
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        print_message "Reseteando base de datos..."
        docker-compose down -v
        docker-compose up -d
        print_message "Base de datos reseteada"
    else
        print_message "Operación cancelada"
    fi
}

# Función para backup
backup_database() {
    local timestamp=$(date +"%Y%m%d_%H%M%S")
    local backup_file="dgi_backup_${timestamp}.sql"
    
    print_message "Creando backup: $backup_file"
    
    if docker-compose exec -T postgres pg_dump -U dgi_user -d dgi_service > "$backup_file"; then
        print_message "Backup creado exitosamente: $backup_file"
        print_message "Tamaño: $(du -h "$backup_file" | cut -f1)"
    else
        print_error "Error al crear backup"
        exit 1
    fi
}

# Función para restaurar backup
restore_database() {
    if [[ $# -eq 0 ]]; then
        print_error "Debes especificar el archivo de backup"
        print_message "Uso: $0 restore <archivo_backup.sql>"
        exit 1
    fi
    
    local backup_file="$1"
    
    if [[ ! -f "$backup_file" ]]; then
        print_error "Archivo de backup no encontrado: $backup_file"
        exit 1
    fi
    
    print_warning "Esta acción sobrescribirá la base de datos actual"
    read -p "¿Estás seguro? (y/N): " -n 1 -r
    echo
    
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        print_message "Restaurando backup: $backup_file"
        
        if docker-compose exec -T postgres psql -U dgi_user -d dgi_service < "$backup_file"; then
            print_message "Backup restaurado exitosamente"
        else
            print_error "Error al restaurar backup"
            exit 1
        fi
    else
        print_message "Operación cancelada"
    fi
}

# Función para conectar a la base de datos
connect_database() {
    print_message "Conectando a PostgreSQL..."
    docker-compose exec postgres psql -U dgi_user -d dgi_service
}

# Función para mostrar ayuda
show_help() {
    print_header
    echo "Uso: $0 [COMANDO]"
    echo ""
    echo "Comandos disponibles:"
    echo "  start     - Iniciar todos los servicios"
    echo "  stop      - Detener todos los servicios"
    echo "  restart   - Reiniciar todos los servicios"
    echo "  status    - Mostrar estado de los servicios"
    echo "  logs      - Mostrar logs de PostgreSQL"
    echo "  reset     - Resetear base de datos (elimina todos los datos)"
    echo "  backup    - Crear backup de la base de datos"
    echo "  restore   - Restaurar backup de la base de datos"
    echo "  connect   - Conectar a la base de datos PostgreSQL"
    echo "  help      - Mostrar esta ayuda"
    echo ""
    echo "Ejemplos:"
    echo "  $0 start                    # Iniciar servicios"
    echo "  $0 backup                   # Crear backup"
    echo "  $0 restore backup.sql       # Restaurar backup"
    echo ""
}

# Función principal
main() {
    check_docker
    check_files
    
    case "${1:-start}" in
        start)
            start_services
            ;;
        stop)
            stop_services
            ;;
        restart)
            restart_services
            ;;
        status)
            show_status
            ;;
        logs)
            show_logs
            ;;
        reset)
            reset_database
            ;;
        backup)
            backup_database
            ;;
        restore)
            restore_database "$2"
            ;;
        connect)
            connect_database
            ;;
        help|--help|-h)
            show_help
            ;;
        *)
            print_error "Comando desconocido: $1"
            show_help
            exit 1
            ;;
    esac
}

# Ejecutar función principal
main "$@"
