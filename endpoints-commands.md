# 🚀 DGI Service - Comandos de Todos los Endpoints

## 📋 Cómo Ejecutar Cada Endpoint

### **🔧 Configuración Base**
```bash
# Variables de entorno
BASE_URL="http://localhost:8081"
API_KEY="ssssssssssssssssssAAAAAAAAAAAAAA"
```

---

## **1. Health Check (Público)**
```bash
curl -X GET "$BASE_URL/health"
```
**Respuesta esperada:**
```json
{
  "service": "dgi-service",
  "status": "ok",
  "timestamp": "2025-09-01T14:16:33.207459Z",
  "version": "1.0.0"
}
```

---

## **2. Crear Emisor (Público)**
```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -d '{
    "name": "EMPRESA TEST COMPLETA",
    "company_code": "TEST001",
    "ruc_tipo": "1",
    "ruc_numero": "12345678",
    "ruc_dv": "1",
    "suc_em": "001",
    "pto_fac_default": "001",
    "iamb": 1,
    "itpemis_default": "1",
    "idoc_default": "01",
    "email": "test@empresa.com",
    "phone": "+1234567890",
    "address_line": "Dirección Test 123",
    "pac_api_key": "test-pac-api-key",
    "pac_subscription_key": "test-pac-subscription-key"
  }' \
  "$BASE_URL/v1/emitters"
```
**Respuesta esperada:**
```json
{
  "id": "bf0e1b6c-bb6f-45e3-ae81-8ba0bbb59f47"
}
```

---

## **3. Obtener Series (Con API Key)**
```bash
curl -X GET \
  -H "X-API-Key: $API_KEY" \
  "$BASE_URL/v1/series"
```
**Respuesta esperada:**
```json
{
  "items": [],
  "page": 1,
  "page_size": 10,
  "total": 0
}
```

---

## **4. Crear Cliente (Con API Key)**
```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $API_KEY" \
  -d '{
    "name": "Cliente Test Completo",
    "ruc": "87654321",
    "email": "cliente@test.com",
    "phone": "+0987654321",
    "address": "Dirección Cliente 456"
  }' \
  "$BASE_URL/v1/customers"
```
**Respuesta esperada:**
```json
{
  "id": "33847fbb-fc14-4624-a8ea-8cea50835386"
}
```

---

## **5. Crear Producto (Con API Key)**
```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $API_KEY" \
  -d '{
    "sku": "SKU-TEST-ALL-001",
    "description": "Producto Test All Endpoints - Descripción del producto test",
    "unit_price": 100.50,
    "tax_rate": "03"
  }' \
  "$BASE_URL/v1/products"
```
**Respuesta esperada:**
```json
{
  "id": "4e6deab0-6b76-4a25-b1db-c07b745b93f2"
}
```

---

## **6. Crear Serie (Con API Key)**
```bash
# Reemplazar {EMITTER_ID} con el ID del emisor creado
curl -X POST \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $API_KEY" \
  -d '{
    "pto_fac_df": "001",
    "doc_kind": "invoice"
  }' \
  "$BASE_URL/v1/emitters/{EMITTER_ID}/series"
```
**Respuesta esperada:**
```json
{
  "id": "series-uuid-here"
}
```

---

## **7. Crear API Key (Con API Key)**
```bash
# Reemplazar {EMITTER_ID} con el ID del emisor creado
curl -X POST \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $API_KEY" \
  -d '{
    "name": "Test API Key All Endpoints",
    "rate_limit_per_min": 1000
  }' \
  "$BASE_URL/v1/emitters/{EMITTER_ID}/apikeys"
```
**Respuesta esperada:**
```json
{
  "id": "api-key-uuid-here",
  "api_key": "generated-api-key-here",
  "name": "Test API Key All Endpoints",
  "rate_limit_per_min": 1000
}
```

---

## **8. Crear Factura (Con API Key)**
```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $API_KEY" \
  -d '{
    "document_type": "invoice",
    "customer": {
      "name": "Cliente Test Completo",
      "email": "cliente@test.com",
      "phone": "+0987654321",
      "address": "Dirección Cliente 456"
    },
    "items": [
      {
        "description": "Producto Test All Endpoints - Descripción del producto test",
        "quantity": 2,
        "unit_price": 100.50,
        "discount": 0,
        "tax_rate": "03"
      }
    ],
    "payment": {
      "method": "01",
      "amount": 214.00
    },
    "notes": "Factura de prueba completa - todos los endpoints"
  }' \
  "$BASE_URL/v1/invoices"
```
**Respuesta esperada:**
```json
{
  "id": "invoice-uuid-here",
  "status": "RECEIVED",
  "document_number": "0000000001"
}
```

---

## **9. Obtener Factura (Con API Key)**
```bash
# Reemplazar {INVOICE_ID} con el ID de la factura creada
curl -X GET \
  -H "X-API-Key: $API_KEY" \
  "$BASE_URL/v1/invoices/{INVOICE_ID}"
```
**Respuesta esperada:**
```json
{
  "id": "invoice-uuid-here",
  "emitter_id": "emitter-uuid-here",
  "customer_id": "customer-uuid-here",
  "status": "RECEIVED",
  "document_number": "0000000001",
  "total_amount": 214.00
}
```

---

## **10. Obtener Archivos de Factura (Con API Key)**
```bash
# Reemplazar {INVOICE_ID} con el ID de la factura creada
curl -X GET \
  -H "X-API-Key: $API_KEY" \
  "$BASE_URL/v1/invoices/{INVOICE_ID}/files"
```
**Respuesta esperada:**
```json
{
  "files": {
    "xml": "url-to-xml-file",
    "pdf": "url-to-pdf-file"
  }
}
```

---

## **11. Reenviar Email (Con API Key)**
```bash
# Reemplazar {INVOICE_ID} con el ID de la factura creada
curl -X POST \
  -H "X-API-Key: $API_KEY" \
  "$BASE_URL/v1/invoices/{INVOICE_ID}/email"
```
**Respuesta esperada:**
```json
{
  "message": "Email sent successfully"
}
```

---

## **12. Reintentar Workflow (Con API Key)**
```bash
# Reemplazar {INVOICE_ID} con el ID de la factura creada
curl -X POST \
  -H "X-API-Key: $API_KEY" \
  "$BASE_URL/v1/invoices/{INVOICE_ID}/retry"
```
**Respuesta esperada:**
```json
{
  "message": "Workflow retry initiated"
}
```

---

## **13. Obtener Dashboard (Con API Key)**
```bash
# Reemplazar {EMITTER_ID} con el ID del emisor creado
curl -X GET \
  -H "X-API-Key: $API_KEY" \
  "$BASE_URL/v1/emitters/{EMITTER_ID}/dashboard"
```
**Respuesta esperada:**
```json
{
  "month": "2025-09",
  "series": null,
  "totals": {
    "issued": 0,
    "authorized": 0,
    "rejected": 0
  }
}
```

---

## **🧪 Pruebas de Casos de Error**

### **14. Probar con API Key Inválida**
```bash
curl -X GET \
  -H "X-API-Key: invalid-key" \
  "$BASE_URL/v1/series"
```
**Respuesta esperada:**
```json
{
  "error": {
    "code": "UNAUTHORIZED",
    "message": "Invalid API key"
  }
}
```

### **15. Probar Crear Cliente con Datos Inválidos**
```bash
curl -X POST \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $API_KEY" \
  -d '{
    "name": "",
    "ruc": "invalid-ruc",
    "email": "invalid-email"
  }' \
  "$BASE_URL/v1/customers"
```
**Respuesta esperada:**
```json
{
  "error": {
    "code": "INVALID_REQUEST",
    "message": "Invalid request format",
    "details": [
      {
        "field": "body",
        "issue": "Key: 'CreateCustomerRequest.Name' Error:Field validation for 'Name' failed on the 'required' tag"
      }
    ]
  }
}
```

---

## **📝 Notas Importantes**

1. **API Key**: Usar `ssssssssssssssssssAAAAAAAAAAAAAA` para todas las pruebas
2. **IDs**: Los IDs se generan automáticamente y deben ser reemplazados en los comandos
3. **Tax Rate**: Usar códigos como `"03"` en lugar de `"0.19"`
4. **Document Type**: Usar `"invoice"` en lugar de `"01"`
5. **Errores**: Algunos endpoints pueden mostrar errores internos (normal en desarrollo)

---

## **🚀 Script de Ejecución Automática**

Para ejecutar todos los endpoints automáticamente:

```bash
# Hacer ejecutable
chmod +x test-all-endpoints.sh

# Ejecutar
./test-all-endpoints.sh
```

Este script ejecutará todos los endpoints y mostrará las respuestas completas.
