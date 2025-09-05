#!/bin/bash

# Script para configurar Inngest para DGI Service
# Colores para output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}🚀 Configurando Inngest para DGI Service${NC}"
echo ""

# 1. Verificar que el usuario tenga una cuenta en Inngest
echo -e "${YELLOW}📋 Paso 1: Verificar cuenta de Inngest${NC}"
echo "Asegúrate de tener una cuenta en https://inngest.com"
echo "Si no tienes cuenta, créala ahora y luego presiona Enter"
read -p "Presiona Enter cuando tengas tu cuenta lista..."

# 2. Crear nueva aplicación
echo ""
echo -e "${YELLOW}📋 Paso 2: Crear nueva aplicación${NC}"
echo "1. Ve al dashboard de Inngest"
echo "2. Haz clic en 'New App'"
echo "3. Nombre: dgi-service"
echo "4. Descripción: Servicio de documentos fiscales para Panamá"
echo "5. Presiona 'Create App'"
read -p "Presiona Enter cuando hayas creado la aplicación..."

# 3. Obtener credenciales
echo ""
echo -e "${YELLOW}📋 Paso 3: Obtener credenciales${NC}"
echo "En el dashboard de tu aplicación, necesitarás:"
echo ""
echo "🔑 Event Key:"
echo "   - Ve a 'Settings' > 'API Keys'"
echo "   - Copia el 'Event Key'"
echo ""
echo "🔐 Signing Key:"
echo "   - En la misma página, copia el 'Signing Key'"
echo ""
echo "🆔 App ID:"
echo "   - En 'Settings' > 'General', copia el 'App ID'"
read -p "Presiona Enter cuando tengas las credenciales..."

# 4. Configurar variables de entorno
echo ""
echo -e "${YELLOW}📋 Paso 4: Configurar variables de entorno${NC}"
echo "Ahora actualiza tu archivo .env con las credenciales:"
echo ""

# Leer credenciales del usuario
read -p "Event Key: " EVENT_KEY
read -p "Signing Key: " SIGNING_KEY
read -p "App ID: " APP_ID

# Validar que no estén vacías
if [ -z "$EVENT_KEY" ] || [ -z "$SIGNING_KEY" ] || [ -z "$APP_ID" ]; then
    echo -e "${RED}❌ Error: Todas las credenciales son requeridas${NC}"
    exit 1
fi

# 5. Crear archivo .env actualizado
echo ""
echo -e "${YELLOW}📋 Paso 5: Actualizar archivo .env${NC}"

# Verificar si existe .env
if [ -f ".env" ]; then
    echo "Archivo .env encontrado. Actualizando credenciales..."
    
    # Actualizar credenciales existentes
    sed -i.bak "s/INNGEST_EVENT_KEY=.*/INNGEST_EVENT_KEY=$EVENT_KEY/" .env
    sed -i.bak "s/INNGEST_SIGNING_KEY=.*/INNGEST_SIGNING_KEY=$SIGNING_KEY/" .env
    sed -i.bak "s/INNGEST_APP_ID=.*/INNGEST_APP_ID=$APP_ID/" .env
    
    echo "✅ Credenciales actualizadas en .env"
else
    echo "Archivo .env no encontrado. Creando uno nuevo..."
    
    cat > .env << EOF
# DGI Service Configuration
# Server Configuration
SERVER_PORT=8081
SERVER_HOST=0.0.0.0
ENVIRONMENT=development

# Database Configuration
DB_HOST=localhost
DB_PORT=5432
DB_USER=dgi_user
DB_PASSWORD=dgi_password_2024
DB_NAME=dgi_service
DB_SSL_MODE=disable

# Redis Configuration
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# Inngest Configuration
INNGEST_EVENT_KEY=$EVENT_KEY
INNGEST_SIGNING_KEY=$SIGNING_KEY
INNGEST_APP_ID=$APP_ID
INNGEST_DEV=true

# JWT Configuration
JWT_SECRET=dgi_jwt_secret_key_2024_development_only_change_in_production
JWT_EXPIRY=24h

# Rate Limiting
RATE_LIMIT_DEFAULT=120
RATE_LIMIT_BURST=10

# Logging
LOG_LEVEL=info
LOG_FORMAT=json

# Email Configuration
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your_email@gmail.com
SMTP_PASSWORD=your_app_password

# PAC Configuration
PAC_API_URL=https://api.pac-provider.com
PAC_TIMEOUT=30s
PAC_MAX_RETRIES=5

# File Storage
STORAGE_TYPE=local
STORAGE_PATH=./storage
STORAGE_BUCKET=dgi-documents
EOF
    
    echo "✅ Archivo .env creado con las credenciales"
fi

# 6. Verificar configuración
echo ""
echo -e "${YELLOW}📋 Paso 6: Verificar configuración${NC}"
echo "Tu configuración de Inngest:"
echo "  Event Key: ${EVENT_KEY:0:8}..."
echo "  Signing Key: ${SIGNING_KEY:0:8}..."
echo "  App ID: $APP_ID"
echo ""

# 7. Instrucciones finales
echo -e "${GREEN}🎉 Configuración completada!${NC}"
echo ""
echo "Próximos pasos:"
echo "1. Ejecuta: make db-start"
echo "2. Ejecuta: make deps"
echo "3. Ejecuta: make run"
echo ""
echo "El servicio se conectará automáticamente a Inngest y registrará los workflows."
echo "Puedes ver los workflows en el dashboard de Inngest en la sección 'Functions'."
echo ""
echo "Para probar:"
echo "1. Crea un documento usando POST /v1/invoices"
echo "2. Ve al dashboard de Inngest para ver la ejecución del workflow"
echo "3. Revisa los logs en tiempo real"
echo ""

# Limpiar backup
if [ -f ".env.bak" ]; then
    rm .env.bak
fi

echo -e "${GREEN}¡Listo para usar Inngest! 🚀${NC}"
