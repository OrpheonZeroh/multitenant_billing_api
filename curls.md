Curl

> Variables de ejemplo
> `API=https://api.tu-dominio.com`
> `X_API_KEY=pk_live_xxx` (para endpoints CORE)
> `X_ADMIN_KEY=admin_xxx` (para endpoints ADMIN)
> `INV_ID=b3f0d35e-4f2a-4f5b-9b7e-3a8a64f7f8d7`
> `EMITTER_ID=8d1be3f2-6b4b-4d8a-9b0d-3a7f9d2c1e55`

---

# CORE

## 1) Crear documento (Factura/Nota) — `POST /v1/invoices`

```bash
curl -X POST "$API/v1/invoices" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $X_API_KEY" \
  -H "Idempotency-Key: 16e3c0f2-4f2f-4f5b-9b7e-3a8a64f7f8d7" \
  -d '{
    "document_type": "invoice",
    "customer": {
      "name": "Cliente Test",
      "email": "cliente@test.com",
      "phone": "507-5678",
      "address": "Ciudad de Panama",
      "ubi_code": "8-8-7"
    },
    "items": [
      {"sku": "RADIO-001", "description": "Radio para Auto", "qty": 1, "unit_price": 150.00, "tax_rate": "00"}
    ],
    "payment": {"method": "02", "amount": 150.00},
    "overrides": { "pto_fac_df": "001", "i_tp_emis": "01" }
  }'
```

**202 Accepted**

```json
{
  "id": "b3f0d35e-4f2a-4f5b-9b7e-3a8a64f7f8d7",
  "status": "RECEIVED",
  "document_type": "invoice",
  "emitter": {
    "ruc": "155646463-2-2017",
    "pto_fac_df": "001",
    "nrodf": "0000000001"
  },
  "totals": { "net": 150.00, "itbms": 0.00, "total": 150.00 },
  "links": {
    "self": "/v1/invoices/b3f0d35e-4f2a-4f5b-9b7e-3a8a64f7f8d7",
    "files": "/v1/invoices/b3f0d35e-4f2a-4f5b-9b7e-3a8a64f7f8d7/files"
  }
}
```

**409 Conflict (idempotencia)**

```json
{
  "error": {
    "code": "CONFLICT",
    "message": "Idempotency key already used for this resource"
  }
}
```

---

## 2) Consultar estado — `GET /v1/invoices/{id}`

```bash
curl -X GET "$API/v1/invoices/$INV_ID" \
  -H "X-API-Key: $X_API_KEY"
```

**200 OK (AUTHORIZED)**

```json
{
  "id": "b3f0d35e-4f2a-4f5b-9b7e-3a8a64f7f8d7",
  "status": "AUTHORIZED",
  "email_status": "SENT",
  "document_type": "invoice",
  "cufe": "FE0120000155646463-2-2017-8600012024032052095049600010128158741019",
  "url_cufe": "https://dgi-fep-test.mef.gob.pa:40001/Consultas/FacturasPorCUFE?CUFE=FE0120...",
  "emitter": { "pto_fac_df": "001", "nrodf": "0000000001" },
  "totals": { "net": 150.00, "itbms": 0.00, "total": 150.00 },
  "created_at": "2025-08-29T20:24:15Z",
  "links": { "files": "/v1/invoices/b3f0d35e-4f2a-4f5b-9b7e-3a8a64f7f8d7/files" }
}
```

**200 OK (REJECTED)**

```json
{
  "id": "b3f0d35e-4f2a-4f5b-9b7e-3a8a64f7f8d7",
  "status": "REJECTED",
  "document_type": "invoice",
  "reject_reason": {
    "code": "1519",
    "message": "Fecha de emisión muy antigua."
  },
  "emitter": { "pto_fac_df": "001", "nrodf": "0000000002" },
  "totals": { "net": 150.00, "itbms": 0.00, "total": 150.00 },
  "created_at": "2025-08-29T20:24:15Z"
}
```

---

## 3) Artefactos (XMLs y PDF) — `GET /v1/invoices/{id}/files`

```bash
curl -X GET "$API/v1/invoices/$INV_ID/files" \
  -H "X-API-Key: $X_API_KEY"
```

**200 OK**

```json
{
  "xml_fe": "PD94bWwgdmVyc2lvbj0iMS4wIj8+PHJGRSB4bWxucz0iaHR0cDovL2RnaS1mZXAubWVmLmdvYi5wYSI+Li4uPC9yRkU+",
  "xml_protocolo": "PHJSZXRFbnZpRmU+Li4uPC9yUmV0RW52aUZlPg==",
  "cafe_pdf_url": "https://cdn.tu-dominio.com/cafe/b3f0d35e-....pdf",
  "disposition": "inline" 
}
```

> Nota: `xml_fe`/`xml_protocolo` pueden venir `base64` (inline) o puedes ofrecer descarga binaria con `?download=1&type=xml_fe`.

---

## 4) Reenviar email — `POST /v1/invoices/{id}/email`

```bash
curl -X POST "$API/v1/invoices/$INV_ID/email" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $X_API_KEY" \
  -d '{ "to": "otro@correo.com", "cc": ["conta@empresa.com"] }'
```

**202 Accepted**

```json
{ "status": "ENQUEUED" }
```

**200 OK (si síncrono)**

```json
{ "status": "SENT", "provider_message_id": "sg-0c9f..." }
```

---

## 5) Reintentar workflow — `POST /v1/invoices/{id}/retry`

```bash
curl -X POST "$API/v1/invoices/$INV_ID/retry" \
  -H "X-API-Key: $X_API_KEY"
```

**202 Accepted**

```json
{ "status": "ENQUEUED", "resume_from": "send_to_pac" }
```

---

## 6) Avance de folios / KPIs de serie — `GET /v1/series`

```bash
curl -X GET "$API/v1/series?doc_kind=invoice&pto_fac_df=001&page=1&page_size=50" \
  -H "X-API-Key: $X_API_KEY"
```

**200 OK**

```json
{
  "items": [
    {
      "pto_fac_df": "001",
      "doc_kind": "invoice",
      "last_assigned": 124,
      "issued_count": 124,
      "authorized_count": 118,
      "rejected_count": 6
    }
  ],
  "page": 1,
  "page_size": 50,
  "total": 1
}
```

---

# ADMIN (opcional / recomendado)

> Usa `X-Admin-Key` distinto al de clientes externos.

## 7) Crear/actualizar cliente — `POST /v1/customers`

```bash
curl -X POST "$API/v1/customers" \
  -H "Content-Type: application/json" \
  -H "X-Admin-Key: $X_ADMIN_KEY" \
  -d '{
    "name":"Cliente Test",
    "email":"cliente@test.com",
    "phone":"507-5678",
    "address":"Ciudad de Panama",
    "ubi_code":"8-8-7"
  }'
```

**201 Created**

```json
{ "id": "f1dc3eaa-6fc5-4fa1-9a1f-0d6d471b0c23" }
```

---

## 8) Crear/actualizar producto — `POST /v1/products`

```bash
curl -X POST "$API/v1/products" \
  -H "Content-Type: application/json" \
  -H "X-Admin-Key: $X_ADMIN_KEY" \
  -d '{
    "sku":"RADIO-001",
    "description":"Radio para Auto",
    "cpbs_abr":"85",
    "cpbs_cmp":"8515",
    "unit_price":150.00,
    "tax_rate":"00"
  }'
```

**201 Created**

```json
{ "id": "a1a1e7ba-5f1d-4d32-9b58-8a97da1f70e2" }
```

---

## 9) Crear emisor (con branding y credenciales PAC) — `POST /v1/emitters`

```bash
curl -X POST "$API/v1/emitters" \
  -H "Content-Type: application/json" \
  -H "X-Admin-Key: $X_ADMIN_KEY" \
  -d '{
    "name":"HYPERNOVA LABS",
    "company_code":"HYPE",
    "ruc_tipo":"2","ruc_numero":"155646463-2-2017","ruc_dv":"86",
    "suc_em":"0001","pto_fac_default":"001",
    "email":"facturas@empresa.com","phone":"507-1234",
    "address_line":"AVENIDA PERU","ubi_code":"8-8-8",
    "brand_logo_url":"https://cdn/logo.png",
    "brand_primary_color":"#0F172A",
    "brand_footer_html":"Gracias por su compra",
    "pac_api_key":"PAC_xxx",
    "pac_subscription_key":"SUB_xxx",
    "iamb":2, "itpemis_default":"01", "idoc_default":"01"
  }'
```

**201 Created**

```json
{ "id": "8d1be3f2-6b4b-4d8a-9b0d-3a7f9d2c1e55" }
```

---

## 10) Crear serie (punto de facturación) — `POST /v1/emitters/{id}/series`

```bash
curl -X POST "$API/v1/emitters/$EMITTER_ID/series" \
  -H "Content-Type: application/json" \
  -H "X-Admin-Key: $X_ADMIN_KEY" \
  -d '{ "pto_fac_df":"001", "doc_kind":"invoice" }'
```

**201 Created**

```json
{
  "id": "2fcf0a35-7f2b-476d-a44f-8f7a2d23f2b1",
  "emitter_id": "8d1be3f2-6b4b-4d8a-9b0d-3a7f9d2c1e55",
  "pto_fac_df": "001",
  "doc_kind": "invoice",
  "next_number": 1,
  "issued_count": 0,
  "authorized_count": 0,
  "rejected_count": 0
}
```

---

## 11) Generar API key para cliente — `POST /v1/emitters/{id}/apikeys`

```bash
curl -X POST "$API/v1/emitters/$EMITTER_ID/apikeys" \
  -H "Content-Type: application/json" \
  -H "X-Admin-Key: $X_ADMIN_KEY" \
  -d '{ "name":"Mi App", "rate_limit_per_min":120 }'
```

**201 Created (muestra la clave UNA sola vez)**

```json
{
  "id": "6e4f7b4a-42f0-41fd-8f02-9e7b2a1f0d11",
  "name": "Mi App",
  "api_key": "pk_live_2Qm6c...A7", 
  "rate_limit_per_min": 120
}
```

---

## 12) Dashboard por emisor (KPIs) — `GET /v1/emitters/{id}/dashboard`

```bash
curl -X GET "$API/v1/emitters/$EMITTER_ID/dashboard?month=2025-08" \
  -H "X-Admin-Key: $X_ADMIN_KEY"
```

**200 OK**

```json
{
  "month": "2025-08",
  "series": [
    { "pto_fac_df":"001", "doc_kind":"invoice", "issued":124, "authorized":118, "rejected":6 },
    { "pto_fac_df":"001", "doc_kind":"credit_note", "issued":12, "authorized":12, "rejected":0 }
  ],
  "totals": {
    "issued": 136,
    "authorized": 130,
    "rejected": 6
  }
}
```

---

## Errores comunes (formato estándar)

**401 Unauthorized**

```json
{ "error": { "code": "UNAUTHORIZED", "message": "Invalid API key" } }
```

**403 Forbidden**

```json
{ "error": { "code": "FORBIDDEN", "message": "API key inactive or not allowed" } }
```

**429 Rate Limited**

```json
{ "error": { "code": "RATE_LIMITED", "message": "Too many requests. Try later." } }
```
