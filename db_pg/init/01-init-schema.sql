-- Script de inicialización para DGI Service
-- Crea todas las tablas y estructuras necesarias

-- Habilitar extensiones necesarias
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Crear tipos enumerados
CREATE TYPE document_status AS ENUM (
    'RECEIVED', 'PREPARING', 'SENDING_TO_PAC', 'AUTHORIZED', 'REJECTED', 'ERROR'
);

CREATE TYPE email_status AS ENUM (
    'PENDING', 'SENT', 'FAILED', 'RETRYING'
);

CREATE TYPE document_type AS ENUM (
    'invoice', 'import_invoice', 'export_invoice', 'credit_note', 'debit_note', 
    'zone_franca', 'reembolso', 'foreign_invoice'
);

CREATE TYPE payment_method AS ENUM (
    '01', '02', '03', '04', '05', '06', '07', '08', '09', '10'
);

-- Tabla de emisores (empresas que emiten documentos)
CREATE TABLE emitters (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    company_code VARCHAR(10) UNIQUE NOT NULL,
    ruc_tipo VARCHAR(1) NOT NULL CHECK (ruc_tipo IN ('1', '2', '3')),
    ruc_numero VARCHAR(20) NOT NULL,
    ruc_dv VARCHAR(2) NOT NULL,
    suc_em VARCHAR(4) NOT NULL DEFAULT '0001',
    pto_fac_default VARCHAR(3) NOT NULL DEFAULT '001',
    iamb INTEGER NOT NULL DEFAULT 1 CHECK (iamb IN (1, 2)),
    itpemis_default VARCHAR(2) NOT NULL DEFAULT '01',
    idoc_default VARCHAR(2) NOT NULL DEFAULT '01',
    email VARCHAR(255) NOT NULL,
    phone VARCHAR(20),
    address_line TEXT,
    ubi_code VARCHAR(10),
    brand_logo_url TEXT,
    brand_primary_color VARCHAR(7),
    brand_footer_html TEXT,
    pac_api_key TEXT NOT NULL,
    pac_subscription_key TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Constraint para RUC único
    UNIQUE(ruc_tipo, ruc_numero, ruc_dv, suc_em)
);

-- Tabla de API keys para integración
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    emitter_id UUID NOT NULL REFERENCES emitters(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    key_hash TEXT NOT NULL UNIQUE,
    is_active BOOLEAN NOT NULL DEFAULT true,
    rate_limit_per_min INTEGER NOT NULL DEFAULT 120,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_used_at TIMESTAMP WITH TIME ZONE,
    
    UNIQUE(emitter_id, name)
);

-- Tabla de series de documentos por emisor
CREATE TABLE emitter_series (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    emitter_id UUID NOT NULL REFERENCES emitters(id) ON DELETE CASCADE,
    pto_fac_df VARCHAR(3) NOT NULL,
    doc_kind document_type NOT NULL,
    next_number INTEGER NOT NULL DEFAULT 1,
    issued_count INTEGER NOT NULL DEFAULT 0,
    authorized_count INTEGER NOT NULL DEFAULT 0,
    rejected_count INTEGER NOT NULL DEFAULT 0,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(emitter_id, pto_fac_df, doc_kind)
);

-- Tabla de clientes
CREATE TABLE customers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    emitter_id UUID NOT NULL REFERENCES emitters(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    phone VARCHAR(20),
    address_line TEXT,
    ubi_code VARCHAR(10),
    tax_id VARCHAR(20),
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(emitter_id, email)
);

-- Tabla de productos/servicios
CREATE TABLE products (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    emitter_id UUID NOT NULL REFERENCES emitters(id) ON DELETE CASCADE,
    sku VARCHAR(100) NOT NULL,
    description TEXT NOT NULL,
    cpbs_abr VARCHAR(2),
    cpbs_cmp VARCHAR(4),
    unit_price DECIMAL(15,2) NOT NULL,
    tax_rate VARCHAR(2) NOT NULL DEFAULT '00',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(emitter_id, sku)
);

-- Tabla principal de documentos (invoices, notas, etc.)
CREATE TABLE invoices (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    emitter_id UUID NOT NULL REFERENCES emitters(id) ON DELETE CASCADE,
    series_id UUID NOT NULL REFERENCES emitter_series(id) ON DELETE CASCADE,
    customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    
    -- Información del documento
    doc_kind document_type NOT NULL,
    d_nrodf VARCHAR(10) NOT NULL, -- Folio asignado (10 dígitos)
    d_ptofacdf VARCHAR(3) NOT NULL,
    status document_status NOT NULL DEFAULT 'RECEIVED',
    email_status email_status NOT NULL DEFAULT 'PENDING',
    
    -- Referencias para notas
    ref_cufe VARCHAR(255),
    ref_nrodf VARCHAR(10),
    ref_ptofacdf VARCHAR(3),
    
    -- Respuesta del PAC
    cufe VARCHAR(255),
    url_cufe TEXT,
    xml_in TEXT,
    xml_response TEXT,
    xml_fe TEXT,
    xml_protocolo TEXT,
    cafe_pdf_url TEXT,
    
    -- Configuración DGI
    iamb INTEGER NOT NULL DEFAULT 1,
    itpemis VARCHAR(2) NOT NULL DEFAULT '01',
    idoc VARCHAR(2) NOT NULL DEFAULT '01',
    
    -- Totales calculados
    subtotal DECIMAL(15,2) NOT NULL DEFAULT 0,
    itbms_amount DECIMAL(15,2) NOT NULL DEFAULT 0,
    total_amount DECIMAL(15,2) NOT NULL DEFAULT 0,
    
    -- Metadatos
    idempotency_key VARCHAR(255) UNIQUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Constraints
    UNIQUE(emitter_id, d_ptofacdf, d_nrodf),
    CHECK (d_nrodf ~ '^[0-9]{10}$')
);

-- Tabla de ítems de documentos
CREATE TABLE invoice_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    line_no INTEGER NOT NULL,
    sku VARCHAR(100),
    description TEXT NOT NULL,
    qty DECIMAL(10,3) NOT NULL,
    unit_price DECIMAL(15,2) NOT NULL,
    itbms_rate VARCHAR(2) NOT NULL DEFAULT '00',
    cpbs_abr VARCHAR(2),
    cpbs_cmp VARCHAR(4),
    line_total DECIMAL(15,2) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(invoice_id, line_no)
);

-- Tabla de logs de email
CREATE TABLE email_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    to_email VARCHAR(255) NOT NULL,
    cc_emails TEXT[],
    subject VARCHAR(500) NOT NULL,
    status email_status NOT NULL DEFAULT 'PENDING',
    provider_id VARCHAR(100),
    error_msg TEXT,
    attempts INTEGER NOT NULL DEFAULT 0,
    sent_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Tabla de webhooks
CREATE TABLE webhooks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_type VARCHAR(100) NOT NULL,
    payload JSONB NOT NULL,
    attempts INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 3,
    last_error TEXT,
    delivered_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    next_retry_at TIMESTAMP WITH TIME ZONE
);

-- Tabla de auditoría de cambios
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    table_name VARCHAR(100) NOT NULL,
    record_id UUID NOT NULL,
    action VARCHAR(20) NOT NULL, -- INSERT, UPDATE, DELETE
    old_values JSONB,
    new_values JSONB,
    changed_by VARCHAR(100),
    changed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Crear índices para performance
CREATE INDEX idx_invoices_emitter_status ON invoices(emitter_id, status);
CREATE INDEX idx_invoices_emitter_created ON invoices(emitter_id, created_at);
CREATE INDEX idx_invoices_idempotency ON invoices(idempotency_key);
CREATE INDEX idx_invoices_cufe ON invoices(cufe);
CREATE INDEX idx_invoice_items_invoice ON invoice_items(invoice_id);
CREATE INDEX idx_email_logs_invoice ON email_logs(invoice_id);
CREATE INDEX idx_webhooks_event_type ON webhooks(event_type);
CREATE INDEX idx_webhooks_next_retry ON webhooks(next_retry_at);
CREATE INDEX idx_audit_logs_table_record ON audit_logs(table_name, record_id);

-- Crear función para actualizar updated_at automáticamente
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Crear triggers para updated_at
CREATE TRIGGER update_emitters_updated_at BEFORE UPDATE ON emitters
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_emitter_series_updated_at BEFORE UPDATE ON emitter_series
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_customers_updated_at BEFORE UPDATE ON customers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_products_updated_at BEFORE UPDATE ON products
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_invoices_updated_at BEFORE UPDATE ON invoices
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Función para incrementar contadores de series
CREATE OR REPLACE FUNCTION increment_series_counters()
RETURNS TRIGGER AS $$
BEGIN
    -- Incrementar issued_count al crear
    IF TG_OP = 'INSERT' THEN
        UPDATE emitter_series 
        SET issued_count = issued_count + 1,
            updated_at = NOW()
        WHERE id = NEW.series_id;
    END IF;
    
    -- Actualizar contadores de autorización/rechazo
    IF TG_OP = 'UPDATE' AND OLD.status != NEW.status THEN
        IF NEW.status = 'AUTHORIZED' THEN
            UPDATE emitter_series 
            SET authorized_count = authorized_count + 1,
                updated_at = NOW()
            WHERE id = NEW.series_id;
        ELSIF NEW.status = 'REJECTED' THEN
            UPDATE emitter_series 
            SET rejected_count = rejected_count + 1,
                updated_at = NOW()
            WHERE id = NEW.series_id;
        END IF;
        
        -- Revertir contadores si cambia de AUTHORIZED/REJECTED a otro estado
        IF OLD.status = 'AUTHORIZED' AND NEW.status != 'AUTHORIZED' THEN
            UPDATE emitter_series 
            SET authorized_count = GREATEST(authorized_count - 1, 0),
                updated_at = NOW()
            WHERE id = NEW.series_id;
        ELSIF OLD.status = 'REJECTED' AND NEW.status != 'REJECTED' THEN
            UPDATE emitter_series 
            SET rejected_count = GREATEST(rejected_count - 1, 0),
                updated_at = NOW()
            WHERE id = NEW.series_id;
        END IF;
    END IF;
    
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Trigger para contadores de series
CREATE TRIGGER trigger_series_counters
    AFTER INSERT OR UPDATE ON invoices
    FOR EACH ROW EXECUTE FUNCTION increment_series_counters();

