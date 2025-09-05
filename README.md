# Multitenant Billing API

API de facturación electrónica multitenant para Panamá con integración DGI, generación de PDF/XML, y envío de emails.

## 🚀 Características

- **Facturación Electrónica**: Generación de facturas, notas de crédito y débito
- **Multitenant**: Soporte para múltiples emisores con API keys
- **Generación de Documentos**: PDF estilizado y XML para DGI
- **Almacenamiento Híbrido**: Local + Supabase Storage
- **Envío de Emails**: Integración con Resend API
- **Workflows**: Integración con Inngest para procesos asíncronos
- **Base de Datos**: PostgreSQL con migraciones automáticas
- **Autenticación**: API Keys con hash SHA-256
- **Rate Limiting**: Control de velocidad de requests
- **Logging**: Logs estructurados con logrus

## 🏗️ Arquitectura

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Frontend      │    │   DGI Service   │    │   PostgreSQL    │
│   (Cliente)     │◄──►│   (Go + Gin)    │◄──►│   (Base de      │
│                 │    │                 │    │    Datos)       │
└─────────────────┘    └─────────────────┘    └─────────────────┘
                                │
                                ▼
                       ┌─────────────────┐
                       │   Supabase      │
                       │   (Storage +    │
                       │    Auth)        │
                       └─────────────────┘
```

## 📋 Requisitos

- Go 1.21+
- PostgreSQL 15+
- Docker & Docker Compose
- Git

## 🛠️ Instalación

### Desarrollo Local

1. **Clonar el repositorio:**
   ```bash
   git clone git@github.com:OrpheonZeroh/multitenant_billing_api.git
   cd multitenant_billing_api
   ```

2. **Configurar variables de entorno:**
   ```bash
   cp .env.example .env
   # Editar .env con tus credenciales
   ```

3. **Iniciar base de datos:**
   ```bash
   docker-compose up -d
   ```

4. **Ejecutar migraciones:**
   ```bash
   make migrate-up
   ```

5. **Ejecutar el servicio:**
   ```bash
   go run cmd/dgi-service/main.go
   ```

### Docker

```bash
docker-compose up -d
```

## 🔧 Configuración

### Variables de Entorno

```bash
# Servidor
SERVER_PORT=8081
SERVER_BASE_URL=http://localhost:8081
GIN_MODE=release

# Base de Datos
DB_HOST=localhost
DB_PORT=5432
DB_NAME=dgi_service
DB_USER=dgi_user
DB_PASSWORD=your_password
DB_SSLMODE=disable

# Resend API
RESEND_API_KEY=your_resend_api_key

# Supabase
SUPABASE_URL=your_supabase_url
SUPABASE_ANON_KEY=your_supabase_anon_key
SUPABASE_SERVICE_ROLE=your_supabase_service_role
SUPABASE_STORAGE_ENDPOINT=your_storage_endpoint
SUPABASE_STORAGE_REGION=your_storage_region
SUPABASE_ACCESS_KEY_ID=your_access_key_id
SUPABASE_SECRET_ACCESS_KEY=your_secret_access_key
```

## 📚 API Endpoints

### Públicos (Sin Autenticación)

- `GET /health` - Health check
- `POST /v1/emitters` - Registrar emisor
- `POST /v1/invoices` - Crear factura
- `GET /v1/invoices/:id` - Obtener factura
- `GET /v1/invoices/:id/files` - Obtener archivos de factura
- `POST /v1/invoices/:id/email` - Reenviar email
- `GET /v1/series` - Obtener series disponibles
- `GET /v1/files/invoices/:id` - Descarga pública de archivos

### Protegidos (Con API Key)

- `POST /v1/customers` - Crear cliente
- `POST /v1/products` - Crear producto
- `POST /v1/emitters/:id/series` - Crear serie
- `POST /v1/emitters/:id/apikeys` - Crear API key
- `GET /v1/emitters/:id/dashboard` - Dashboard del emisor

## 🔑 Autenticación

El servicio usa API Keys para autenticación. Incluye el header:

```bash
X-API-Key: your_api_key_here
```

## �� Modelos de Datos

### Emisor
```json
{
  "id": "uuid",
  "name": "string",
  "ruc": "string",
  "email": "string",
  "phone": "string",
  "address": "string"
}
```

### Factura
```json
{
  "id": "uuid",
  "document_number": "string",
  "document_type": "invoice|credit_note|debit_note",
  "customer": "Customer",
  "items": "Item[]",
  "payment": "Payment",
  "totals": "Totals"
}
```

## 🚀 Deploy en Railway

1. **Instalar Railway CLI:**
   ```bash
   npm install -g @railway/cli
   ```

2. **Login a Railway:**
   ```bash
   railway login
   ```

3. **Deploy:**
   ```bash
   ./deploy-railway.sh
   ```

4. **Configurar variables de entorno en Railway Dashboard**

## 🧪 Testing

```bash
# Ejecutar tests
make test

# Test de endpoints
./test-endpoints.sh
```

## 📝 Scripts Disponibles

- `make dev` - Ejecutar en modo desarrollo
- `make build` - Compilar la aplicación
- `make test` - Ejecutar tests
- `make migrate-up` - Ejecutar migraciones
- `make migrate-down` - Revertir migraciones
- `make docker-build` - Construir imagen Docker
- `make docker-run` - Ejecutar contenedor

## 🔍 Logs

El servicio genera logs estructurados en formato JSON:

```json
{
  "level": "info",
  "msg": "Invoice created successfully",
  "invoice_id": "uuid",
  "document_number": "0000000001",
  "time": "2025-09-02T21:30:00Z"
}
```

## 📈 Monitoreo

- **Health Check**: `GET /health`
- **Métricas**: Logs estructurados
- **Alertas**: Configurables via Railway

## 🤝 Contribución

1. Fork el proyecto
2. Crea una rama para tu feature (`git checkout -b feature/AmazingFeature`)
3. Commit tus cambios (`git commit -m 'Add some AmazingFeature'`)
4. Push a la rama (`git push origin feature/AmazingFeature`)
5. Abre un Pull Request

## 📄 Licencia

Este proyecto está bajo la Licencia MIT. Ver `LICENSE` para más detalles.

## 🆘 Soporte

Para soporte, contacta a:
- Email: jadamson382@gmail.com
- GitHub Issues: [Crear un issue](https://github.com/OrpheonZeroh/multitenant_billing_api/issues)

## 🗺️ Roadmap

- [ ] Integración completa con DGI
- [ ] Dashboard web
- [ ] API de webhooks
- [ ] Soporte para múltiples países
- [ ] Integración con sistemas de contabilidad
- [ ] Mobile app

---

**Desarrollado con ❤️ por Hypernova Labs**
