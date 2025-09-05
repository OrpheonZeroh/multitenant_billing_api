# DGI Service - Base de Datos PostgreSQL

Este directorio contiene la configuración completa de la base de datos PostgreSQL para el servicio DGI (Dirección General de Ingresos) de Panamá.

## 🚀 Inicio Rápido

### 1. Levantar la Base de Datos

```bash
# Desde el directorio db_pg/
docker-compose up -d
```

### 2. Verificar el Estado

```bash
# Ver logs de PostgreSQL
docker-compose logs postgres

# Ver logs de Adminer
docker-compose logs adminer

# Verificar que los servicios estén corriendo
docker-compose ps
```

### 3. Acceder a Adminer

- **URL**: http://localhost:8080
- **Sistema**: PostgreSQL
- **Servidor**: postgres
- **Usuario**: dgi_user
- **Contraseña**: dgi_password_2024
- **Base de datos**: dgi_service

## 📊 Estructura de la Base de Datos

### Tablas Principales

| Tabla | Descripción | Registros Iniciales |
|-------|-------------|---------------------|
| `emitters` | Empresas emisoras de documentos | 1 (HYPERNOVA LABS) |
| `api_keys` | Claves de API para integración | 2 (Test + Production) |
| `emitter_series` | Series de documentos por emisor | 5 series |
| `customers` | Catálogo de clientes | 3 clientes |
| `products` | Catálogo de productos/servicios | 5 productos |
| `invoices` | Documentos principales | 0 (vacía inicialmente) |
| `invoice_items` | Líneas de documentos | 0 (vacía inicialmente) |
| `email_logs` | Logs de envío de emails | 0 (vacía inicialmente) |
| `webhooks` | Cola de webhooks | 0 (vacía inicialmente) |
| `audit_logs` | Auditoría de cambios | 0 (vacía inicialmente) |

### Tablas de Catálogo

| Tabla | Descripción |
|-------|-------------|
| `cpbs_catalog` | Clasificación Panameña de Bienes y Servicios |
| `tax_rates` | Tasas de impuesto ITBMS |
| `payment_methods` | Métodos de pago |
| `system_config` | Configuración del sistema |

## 🔧 Configuración

### Variables de Entorno

```bash
# PostgreSQL
POSTGRES_USER=dgi_user
POSTGRES_PASSWORD=dgi_password_2024
POSTGRES_DB=dgi_service

# Redis (opcional)
REDIS_PORT=6379
```

### Puertos

| Servicio | Puerto | Descripción |
|----------|--------|-------------|
| PostgreSQL | 5432 | Base de datos principal |
| Adminer | 8080 | Interfaz web de administración |
| Redis | 6379 | Cache y sesiones |

### Volúmenes

- `postgres_data`: Datos persistentes de PostgreSQL
- `redis_data`: Datos persistentes de Redis
- `./init`: Scripts de inicialización
- `./postgresql.conf`: Configuración de PostgreSQL

## 📝 Scripts de Inicialización

Los scripts se ejecutan en orden automáticamente al levantar el contenedor:

1. **`01-init-schema.sql`**: Crea todas las tablas, tipos, índices y triggers
2. **`02-seed-data.sql`**: Inserta datos de catálogo iniciales
3. **`03-environment.sql`**: Configura el entorno y funciones auxiliares
4. **`04-verify-setup.sql`**: Verifica que todo esté configurado correctamente

## 🧪 Datos de Prueba

### Emisor de Ejemplo

- **Nombre**: HYPERNOVA LABS
- **RUC**: 2-155646463-86-0001
- **Código**: HYPE
- **Punto de Facturación**: 001

### API Keys de Prueba

| Nombre | Clave | Rate Limit |
|--------|-------|-------------|
| Test App | test_api_key_123 | 1000/min |
| Production App | prod_api_key_456 | 120/min |

### Clientes de Prueba

- Cliente Test 1 (cliente1@test.com)
- Cliente Test 2 (cliente2@test.com)
- Empresa Demo (demo@empresa.com)

### Productos de Prueba

- Radio para Auto Bluetooth ($150.00)
- Laptop Gaming 16GB RAM ($1,200.00)
- Servicio de Consultoría IT ($75.00)
- Software de Gestión ($299.00)
- Mantenimiento Preventivo ($120.00)

## 🔍 Verificación del Setup

### 1. Verificación Automática

```bash
# Conectar a la base de datos
docker-compose exec postgres psql -U dgi_user -d dgi_service

# Ejecutar verificación
\i /docker-entrypoint-initdb.d/04-verify-setup.sql
```

### 2. Verificación Manual

```sql
-- Verificar extensiones
SELECT * FROM pg_extension WHERE extname IN ('uuid-ossp', 'pgcrypto');

-- Verificar tipos enumerados
SELECT typname FROM pg_type WHERE typname IN ('document_status', 'email_status', 'document_type');

-- Verificar tablas
SELECT tablename FROM pg_tables WHERE schemaname = 'public' ORDER BY tablename;

-- Verificar datos de catálogo
SELECT COUNT(*) as emitters FROM emitters;
SELECT COUNT(*) as series FROM emitter_series;
SELECT COUNT(*) as customers FROM customers;
SELECT COUNT(*) as products FROM products;
```

## 🚨 Troubleshooting

### Problemas Comunes

#### 1. Contenedor no inicia

```bash
# Ver logs detallados
docker-compose logs postgres

# Verificar puertos disponibles
netstat -tulpn | grep :5432
netstat -tulpn | grep :8080

# Reiniciar servicios
docker-compose down
docker-compose up -d
```

#### 2. Error de conexión a la base de datos

```bash
# Verificar que PostgreSQL esté corriendo
docker-compose exec postgres pg_isready -U dgi_user

# Verificar credenciales
docker-compose exec postgres psql -U dgi_user -d dgi_service -c "SELECT 1;"
```

#### 3. Scripts de inicialización fallan

```bash
# Ver logs de inicialización
docker-compose logs postgres | grep "docker-entrypoint-initdb.d"

# Ejecutar scripts manualmente
docker-compose exec postgres psql -U dgi_user -d dgi_service -f /docker-entrypoint-initdb.d/01-init-schema.sql
```

#### 4. Problemas de permisos

```bash
# Verificar permisos de archivos
ls -la init/
ls -la postgresql.conf

# Corregir permisos si es necesario
chmod 644 init/*.sql
chmod 644 postgresql.conf
```

### Logs y Debugging

```bash
# Ver logs en tiempo real
docker-compose logs -f postgres

# Ver logs de Adminer
docker-compose logs -f adminer

# Ver logs de Redis
docker-compose logs -f redis

# Conectar a PostgreSQL para debugging
docker-compose exec postgres psql -U dgi_user -d dgi_service
```

## 📈 Monitoreo y Mantenimiento

### Funciones Útiles

```sql
-- Obtener estadísticas de la base de datos
SELECT * FROM get_database_stats();

-- Validar integridad de datos
SELECT * FROM validate_data_integrity();

-- Generar reporte de KPIs
SELECT * FROM generate_kpi_report();

-- Limpiar logs antiguos (mantener últimos 90 días)
SELECT cleanup_old_logs(90);
```

### Vistas Útiles

```sql
-- Dashboard de emisores
SELECT * FROM v_emitter_dashboard;

-- Resumen de invoices
SELECT * FROM v_invoice_summary;
```

## 🔒 Seguridad

### Recomendaciones de Producción

1. **Cambiar contraseñas por defecto**
2. **Configurar firewall para limitar acceso**
3. **Usar SSL/TLS para conexiones**
4. **Implementar backup automático**
5. **Configurar monitoreo de logs**

### Usuarios y Permisos

```sql
-- Crear usuario específico para la aplicación
CREATE USER dgi_app WITH PASSWORD 'strong_password_here';

-- Otorgar permisos mínimos necesarios
GRANT CONNECT ON DATABASE dgi_service TO dgi_app;
GRANT USAGE ON SCHEMA public TO dgi_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO dgi_app;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO dgi_app;
```

## 📚 Recursos Adicionales

### Documentación

- [PostgreSQL 15 Documentation](https://www.postgresql.org/docs/15/)
- [Docker Compose Reference](https://docs.docker.com/compose/)
- [Adminer Documentation](https://www.adminer.org/)

### Comandos Útiles

```bash
# Backup de la base de datos
docker-compose exec postgres pg_dump -U dgi_user dgi_service > backup.sql

# Restaurar backup
docker-compose exec -T postgres psql -U dgi_user -d dgi_service < backup.sql

# Ver estadísticas de contenedores
docker stats

# Limpiar recursos no utilizados
docker system prune -f
```

## 🤝 Contribución

Para reportar problemas o sugerir mejoras:

1. Verificar que el problema no esté en la sección de troubleshooting
2. Revisar los logs del contenedor
3. Proporcionar información del entorno (OS, Docker version, etc.)
4. Incluir pasos para reproducir el problema

---

**Estado**: ✅ Configuración completa lista para desarrollo
**Última actualización**: Diciembre 2024
**Versión**: 1.0.0
