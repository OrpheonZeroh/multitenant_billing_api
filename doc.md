¡Listo, Jonathan! Aquí tienes la **documentación completa** para arrancar el desarrollo del servicio (Go + Gin + Inngest/Temporal). Incluye: endpoints (core + admin), contratos de request/response, auth, idempotencia, rate-limit, manejo de errores, relación con el workflow engine, y ejemplos `curl`.

---

# 0) Resumen de arquitectura

* **API HTTP (Gin)**: valida, crea registros mínimos (transacción con folio), y **dispara workflows**.
* **Workflow Engine (Inngest/Temporal)**: envía al PAC, parsea autorización (0260), genera CAFE (PDF), envía email, emite webhooks, reintentos.
* **PostgreSQL**: catálogo (emisores, clientes, productos), series/folios y KPIs, invoices e ítems, logs de email y webhooks.

---

# 1) Autenticación y encabezados

* `X-API-Key: <clave>` — identifica a la app cliente y **mapea a un emisor**.
* `Idempotency-Key: <uuid>` — recomendado para `POST /v1/invoices` y reintentos del cliente.
* Respuesta común de rate-limit: `429 Too Many Requests` + cabeceras:

  * `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `Retry-After`.

---

# 2) Códigos y formato de errores

```json
{
  "error": {
    "code": "INVALID_REQUEST",
    "message": "customer.email is required",
    "details": [{"field": "customer.email", "issue": "required"}]
  }
}
```

* Códigos:

  * `INVALID_REQUEST` (400) — validaciones de payload.
  * `UNAUTHORIZED` (401) — API key inválida.
  * `FORBIDDEN` (403) — API key inactiva o permisos insuficientes.
  * `NOT_FOUND` (404) — recurso inexistente.
  * `CONFLICT` (409) — idempotencia / folio duplicado.
  * `RATE_LIMITED` (429) — límite excedido.
  * `INTERNAL` (500) — error no controlado.

---

# 3) Endpoints CORE (públicos)

## 3.1 `POST /v1/invoices` — Crear documento y disparar workflow

Crea **Factura / Nota de Crédito / Nota de Débito** según `document_type`, reserva folio y dispara el flujo.

**Headers**:
`X-API-Key`, `Idempotency-Key` (opcional pero recomendado)

**Body (mínimo):**

```json
{
  "document_type": "invoice",             
  "reference": {                          // requerido para notas referenciadas
    "cufe": "CUFE-ORIGINAL",
    "nrodf": "0000001234",
    "pto_fac_df": "001"
  },
  "customer": {
    "name": "Cliente Test",
    "email": "cliente@test.com",
    "phone": "507-5678",
    "address": "Ciudad de Panama",
    "ubi_code": "8-8-7"
  },
  "items": [
    {"sku": "RADIO-001", "description": "Radio para Auto", "qty": 1, "unit_price": 150.0, "tax_rate": "00"}
  ],
  "payment": {"method": "02", "amount": 150.0},
  "overrides": {
    "pto_fac_df": "001",
    "i_tp_emis": "01",
    "i_doc": "01"
  }
}
```

**Notas**

* `document_type` → mapea a `iDoc`.
* Para **notas** referenciadas, `reference` es **obligatorio**.
* Los **totales** los calcula el servicio (no se aceptan del cliente).

**201 Created / 202 Accepted (recomendado)**:

```json
{
  "id": "b3f0d35e-4f2a-4f5b-9b7e-3a8a64f7f8d7",
  "status": "RECEIVED",
  "emitter": {"ruc": "155646463-2-2017", "pto_fac_df": "001", "nrodf": "0000000001"},
  "totals": {"net": 150.0, "itbms": 0.0, "total": 150.0},
  "links": {
    "self": "/v1/invoices/b3f0d35e-4f2a-4f5b-9b7e-3a8a64f7f8d7",
    "files": "/v1/invoices/b3f0d35e-4f2a-4f5b-9b7e-3a8a64f7f8d7/files"
  }
}
```

**Errores**: 400/401/403/409.

**Ejemplo `curl`:**

```bash
curl -X POST https://api.tu-dominio.com/v1/invoices \
 -H "Content-Type: application/json" \
 -H "X-API-Key: <API_KEY>" \
 -H "Idempotency-Key: 16e3c0f2-4f2f-4f5b-9b7e-3a8a64f7f8d7" \
 -d @invoice.json
```

---

## 3.2 `GET /v1/invoices/{id}` — Obtener estado y metadatos

**200 OK:**

```json
{
  "id": "b3f0d35e-4f2a-4f5b-9b7e-3a8a64f7f8d7",
  "status": "AUTHORIZED",
  "email_status": "SENT",
  "document_type": "invoice",
  "cufe": "FE0120...",
  "url_cufe": "https://...",
  "emitter": {"pto_fac_df": "001", "nrodf": "0000000001"},
  "totals": {"net": 150.0, "itbms": 0.0, "total": 150.0},
  "created_at": "2025-08-29T20:24:15Z",
  "links": {"files": "/v1/invoices/.../files"}
}
```

---

## 3.3 `GET /v1/invoices/{id}/files` — Artefactos (FE.xml, Protocolo.xml, CAFE.pdf)

**200 OK (metadatos/URLs):**

```json
{
  "xml_fe": "base64://... (opcional si inline)",
  "xml_protocolo": "base64://...",
  "cafe_pdf_url": "https://bucket/cafe/....pdf"
}
```

> Si necesitas descarga directa, apoya `?download=1` y responde binario por archivo.

---

## 3.4 `POST /v1/invoices/{id}/email` — Reenviar email

**Body (opcional):**

```json
{
  "to": "otro@correo.com",
  "cc": ["contabilidad@empresa.com"]
}
```

**202 Accepted**: dispara `invoice.email.requested`.
**200 OK**: si lo haces síncrono (no recomendado).

---

## 3.5 `POST /v1/invoices/{id}/retry` — Reintentar workflow

Dispara un evento para re-procesar desde el último paso fallido.

**202 Accepted**:

```json
{"status":"ENQUEUED"}
```

---

## 3.6 `GET /v1/series` — Avance de folios/KPIs por emisor

**Query**: `doc_kind` (opcional), `pto_fac_df` (opcional), `page`, `page_size`.

**200 OK:**

```json
{
  "items": [
    {
      "pto_fac_df": "001",
      "doc_kind": "invoice",
      "last_assigned": 124,            // next_number - 1
      "issued_count": 124,
      "authorized_count": 118,
      "rejected_count": 6
    }
  ],
  "page": 1, "page_size": 50, "total": 1
}
```

---

# 4) Endpoints ADMIN (opcionales pero recomendados)

> Protege con un **API key distinto** o JWT interno.

## 4.1 `POST /v1/customers`

Alta/edición (upsert por email opcional).

```json
{
  "name":"Cliente Test","email":"cliente@test.com","phone":"507-5678",
  "address":"Ciudad de Panama","ubi_code":"8-8-7"
}
```

**200/201 OK** → `{"id":"uuid"}`

## 4.2 `POST /v1/products`

Alta/edición (upsert por `sku`).

```json
{
  "sku":"RADIO-001","description":"Radio para Auto",
  "cpbs_abr":"85","cpbs_cmp":"8515",
  "unit_price":150.0,"tax_rate":"00"
}
```

## 4.3 `POST /v1/emitters`

Crear emisor con **branding** y credenciales PAC.

```json
{
  "name":"HYPERNOVA LABS",
  "company_code":"HYPE",
  "ruc_tipo":"2","ruc_numero":"155646463-2-2017","ruc_dv":"86",
  "suc_em":"0001","pto_fac_default":"001",
  "email":"facturas@empresa.com","phone":"507-1234",
  "address_line":"AVENIDA PERU","ubi_code":"8-8-8",
  "brand_logo_url":"https://cdn/logo.png",
  "brand_primary_color":"#0F172A",
  "brand_footer_html":"Gracias por su compra",
  "pac_api_key":"<PAC_KEY>","pac_subscription_key":"<PAC_SUB>"
}
```

## 4.4 `POST /v1/emitters/{id}/series`

Crear **serie** por tipo de documento.

```json
{"pto_fac_df":"001","doc_kind":"invoice"}
```

## 4.5 `POST /v1/emitters/{id}/apikeys`

Generar API key para integrar apps.
**Respuesta (mostrar la clave solo una vez):**

```json
{
  "name":"Mi App",
  "api_key":"<PLAINTEXT-ONLY-ONCE>",
  "rate_limit_per_min":120
}
```

## 4.6 `GET /v1/emitters/{id}/dashboard`

KPIs por mes/serie/tipo.

```json
{
  "month":"2025-08",
  "series":[
    {"pto_fac_df":"001","doc_kind":"invoice","issued":124,"authorized":118,"rejected":6}
  ]
}
```

---

# 5) Mapeos y reglas de negocio

* `document_type` → `iDoc`:

  * `invoice`→`01`, `import_invoice`→`02`, `export_invoice`→`03`,
  * `credit_note`→`04` (referenciada) o `06` (genérica si no hay `reference`),
  * `debit_note`→`05` (referenciada) o `07` (genérica),
  * `zone_franca`→`08`, `reembolso`→`09`, `foreign_invoice`→`10`.
* `iAmb` (1 prod / 2 pruebas) desde emisor.
* `iTpEmis` default `01`; soportar `02/04` (contingencia) con campos extra y reglas de tiempo.
* **Serie/folio**:

  * Tabla `emitter_series (emitter_id, pto_fac_df, doc_kind, next_number, issued/authorized/rejected)`.
  * **Transacción**: `SELECT ... FOR UPDATE`, asigna `dNroDF` **10 dígitos** (left-pad), incrementa `next_number`, inserta invoice.
  * Contadores:

    * `issued_count++` al crear.
    * `authorized_count++` al pasar a AUTHORIZED.
    * `rejected_count++` al pasar a REJECTED.
* **Totales**: servicio calcula líneas, ITBMS, neto, total; valida `payment.amount`.
* **Branding**: por emisor (`brand_logo_url`, `brand_primary_color`, `brand_footer_html`) en CAFE y email.
* **Email**: asunto `Factura {pto}-{nro} | {Emisor}` (o Nota…), adjuntos `FE.xml`, `Protocolo.xml`, `CAFE.pdf`.

---

# 6) Journey con Workflow Engine

### Evento principal:

* `invoice.created` — payload: `{ invoice_id }`

### Pasos del workflow:

1. **prepare**: enriquecer (CPBS), totales, fechas, validar notas → update invoice.
2. **send\_to\_pac**: POST; guardar `xml_in/xml_response`; si 0260 → `AUTHORIZED` + `CUFE/url/xmls`; si rechazo → `REJECTED`.
3. **render\_cafe**: si `AUTHORIZED`, HTML→PDF con branding; guarda `cafe_pdf_url`.
4. **email\_cfe**: envía email con adjuntos; registra en `email_logs`; `email_status=SENT`.
5. **emit\_webhook**: `invoice.authorized` / `invoice.rejected`.

### Eventos auxiliares:

* `invoice.email.requested` → reenvía email.
* `invoice.retry` → re-ejecuta desde último paso fallido.

### Reintentos/backoff:

* `send_to_pac`: p.ej. 5 intentos (1s, 5s, 30s, 2m, 10m).
* `email_cfe`: 3 intentos (10s, 60s, 5m).
* DLQ con alerta si se agotan.

---

# 7) Esquema de datos (resumen mínimo para construcción)

Tablas claves (columnas esenciales):

* `emitters(id, name, ruc_tipo, ruc_numero, ruc_dv, suc_em, pto_fac_default, iamb, itpemis_default, idoc_default, pac_api_key, pac_subscription_key, brand_logo_url, brand_primary_color, brand_footer_html, ...)`
* `api_keys(id, emitter_id, name, key_hash, is_active, rate_limit_per_min, created_at)`
* `emitter_series(id, emitter_id, pto_fac_df, doc_kind, next_number, issued_count, authorized_count, rejected_count, unique(emitter_id, pto_fac_df, doc_kind))`
* `customers(id, emitter_id, name, email, phone, address_line, ubi_code, ... )`
* `products(id, emitter_id, sku, description, cpbs_abr, cpbs_cmp, unit_price, tax_rate, ...)`
* `invoices(id, emitter_id, series_id, customer_id, doc_kind, d_nrodf, d_ptofacdf, status, email_status, cufe, url_cufe, xml_in, xml_response, xml_fe, xml_protocolo, cafe_pdf_url, totals..., ref_*, iamb, itpemis, idoc, created_at, unique(emitter_id, d_ptofacdf, d_nrodf))`
* `invoice_items(id, invoice_id, line_no, sku, description, qty, unit_price, itbms_rate, cpbs_abr, cpbs_cmp, line_total)`
* `email_logs(id, invoice_id, to_email, subject, status, provider_id, error_msg, created_at)`
* `webhooks(id, event_type, payload, attempts, last_error, delivered_at)`

*(Si quieres la versión completa en SQL ya la tengo, me dices y te la paso como script de migración.)*

---

# 8) OpenAPI (extracto abreviado para iniciar)

```yaml
openapi: 3.0.3
info:
  title: DGI Wrapper API
  version: "1.0.0"
servers:
  - url: https://api.tu-dominio.com
components:
  securitySchemes:
    ApiKeyAuth:
      type: apiKey
      in: header
      name: X-API-Key
security:
  - ApiKeyAuth: []
paths:
  /v1/invoices:
    post:
      summary: Create invoice / credit note / debit note
      parameters:
        - in: header
          name: Idempotency-Key
          required: false
          schema: { type: string, format: uuid }
      requestBody:
        required: true
        content:
          application/json:
            schema: { $ref: '#/components/schemas/CreateInvoiceRequest' }
      responses:
        '202': { $ref: '#/components/responses/InvoiceAccepted' }
        '201': { $ref: '#/components/responses/InvoiceAccepted' }
        '400': { $ref: '#/components/responses/Error' }
  /v1/invoices/{id}:
    get:
      summary: Get invoice status
      parameters: [{ in: path, name: id, required: true, schema: { type: string, format: uuid } }]
      responses:
        '200': { $ref: '#/components/responses/Invoice' }
  /v1/invoices/{id}/files:
    get:
      summary: Get invoice artifacts
      parameters: [{ in: path, name: id, required: true, schema: { type: string, format: uuid } }]
      responses:
        '200': { $ref: '#/components/responses/InvoiceFiles' }
  /v1/invoices/{id}/email:
    post:
      summary: Resend email with attachments
      parameters: [{ in: path, name: id, required: true, schema: { type: string, format: uuid } }]
      requestBody:
        content: { application/json: { schema: { $ref: '#/components/schemas/EmailResend' } } }
      responses:
        '202': { description: Enqueued }
  /v1/invoices/{id}/retry:
    post:
      summary: Retry workflow
      parameters: [{ in: path, name: id, required: true, schema: { type: string, format: uuid } }]
      responses:
        '202': { description: Enqueued }
  /v1/series:
    get:
      summary: Series dashboard
      responses:
        '200': { $ref: '#/components/responses/SeriesList' }

components:
  schemas:
    CreateInvoiceRequest:
      type: object
      required: [document_type, customer, items, payment]
      properties:
        document_type: { type: string, enum: [invoice, credit_note, debit_note, import_invoice, export_invoice, zone_franca, reembolso, foreign_invoice] }
        reference:
          type: object
          properties:
            cufe: { type: string }
            nrodf: { type: string }
            pto_fac_df: { type: string }
        customer:
          type: object
          required: [name, email]
          properties:
            name: { type: string }
            email: { type: string, format: email }
            phone: { type: string }
            address: { type: string }
            ubi_code: { type: string }
        items:
          type: array
          minItems: 1
          items:
            type: object
            required: [description, qty, unit_price]
            properties:
              sku: { type: string }
              description: { type: string }
              qty: { type: number }
              unit_price: { type: number }
              tax_rate: { type: string }
        payment:
          type: object
          required: [method, amount]
          properties:
            method: { type: string }
            amount: { type: number }
        overrides:
          type: object
          properties:
            pto_fac_df: { type: string }
            i_tp_emis: { type: string }
            i_doc: { type: string }
  responses:
    InvoiceAccepted:
      description: Accepted
      content:
        application/json:
          schema:
            type: object
            properties:
              id: { type: string, format: uuid }
              status: { type: string }
              emitter:
                type: object
                properties:
                  ruc: { type: string }
                  pto_fac_df: { type: string }
                  nrodf: { type: string }
              totals:
                type: object
                properties:
                  net: { type: number }
                  itbms: { type: number }
                  total: { type: number }
              links:
                type: object
                properties:
                  self: { type: string }
                  files: { type: string }
    Invoice:
      description: Invoice status
      content: { application/json: { schema: { type: object } } }
    InvoiceFiles:
      description: Artifacts
      content: { application/json: { schema: { type: object } } }
    Error:
      description: Error
      content:
        application/json:
          schema:
            type: object
            properties:
              error:
                type: object
                properties:
                  code: { type: string }
                  message: { type: string }
                  details: { type: array, items: { type: object } }
```

---

# 9) Ejemplo de secuencia (texto)

1. **Cliente** llama `POST /v1/invoices` con `X-API-Key`.
2. **API** valida → bloquea `emitter_series`, asigna `dNroDF`, crea `invoice` + `items`, dispara `invoice.created`.
3. **Workflow**:

   * `prepare` → totales + CPBS.
   * `send_to_pac` → guarda XMLs, si 0260 → `AUTHORIZED` + CUFE/URLs.
   * `render_cafe` → genera PDF con **logo del emisor**.
   * `email_cfe` → envía email con adjuntos; `email_status=SENT`.
4. **Cliente** consulta `GET /v1/invoices/{id}` y `GET /v1/invoices/{id}/files`.
5. **Dashboard** usa `GET /v1/series` para KPIs y folios.

---

# 10) Buenas prácticas a implementar

* **HTTP rápido** (validar + persistir + encolar).
* **Idempotencia** en creación (no duplica folios).
* **Rate-limit** por API key.
* **Logs de auditoría**: `xml_in/xml_response`, `email_logs`.
* **Observabilidad**: métricas por paso del workflow y contadores por serie.
* **Feature flags** por emisor: iAmb, plantillas CAFE, proveedor de email.
* **Seguridad**: hashear API keys, no loggear credenciales PAC, CORS mínimo.

---
