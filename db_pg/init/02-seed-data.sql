-- Script de datos iniciales para DGI Service
-- Inserta datos de catálogo necesarios para funcionamiento

-- Insertar emisor de ejemplo (HYPERNOVA LABS)
INSERT INTO emitters (
    name, company_code, ruc_tipo, ruc_numero, ruc_dv, suc_em, 
    pto_fac_default, iamb, itpemis_default, idoc_default,
    email, phone, address_line, ubi_code,
    brand_logo_url, brand_primary_color, brand_footer_html,
    pac_api_key, pac_subscription_key
) VALUES (
    'HYPERNOVA LABS',
    'HYPE',
    '2',
    '155646463',
    '86',
    '0001',
    '001',
    1, -- Producción
    '01', -- Normal
    '01', -- Factura
    'facturas@hypernova.com',
    '507-1234',
    'AVENIDA PERU, CIUDAD DE PANAMA',
    '8-8-8',
    'https://cdn.hypernova.com/logo.png',
    '#0F172A',
    '<p>Gracias por su compra. Para consultas: facturas@hypernova.com</p>',
    'pac_api_key_example_123',
    'pac_subscription_key_example_456'
) ON CONFLICT (ruc_tipo, ruc_numero, ruc_dv, suc_em) DO NOTHING;

-- Obtener el ID del emisor insertado
DO $$
DECLARE
    v_emitter_id UUID;
BEGIN
    SELECT id INTO v_emitter_id FROM emitters WHERE company_code = 'HYPE' LIMIT 1;
    
    IF v_emitter_id IS NOT NULL THEN
        -- Insertar series de documentos
        INSERT INTO emitter_series (emitter_id, pto_fac_df, doc_kind, next_number) VALUES
            (v_emitter_id, '001', 'invoice', 1),
            (v_emitter_id, '001', 'credit_note', 1),
            (v_emitter_id, '001', 'debit_note', 1),
            (v_emitter_id, '002', 'invoice', 1), -- Punto de facturación alternativo
            (v_emitter_id, '002', 'credit_note', 1)
        ON CONFLICT (emitter_id, pto_fac_df, doc_kind) DO NOTHING;
        
        -- Insertar clientes de ejemplo
        INSERT INTO customers (emitter_id, name, email, phone, address_line, ubi_code, tax_id) VALUES
            (v_emitter_id, 'Cliente Test 1', 'cliente1@test.com', '507-5678', 'Ciudad de Panama', '8-8-7', '123456789'),
            (v_emitter_id, 'Cliente Test 2', 'cliente2@test.com', '507-8765', 'Ciudad de Panama', '8-8-9', '987654321'),
            (v_emitter_id, 'Empresa Demo', 'demo@empresa.com', '507-1111', 'Panama City', '8-8-10', '555666777')
        ON CONFLICT (emitter_id, email) DO NOTHING;
        
        -- Insertar productos de ejemplo
        INSERT INTO products (emitter_id, sku, description, cpbs_abr, cpbs_cmp, unit_price, tax_rate) VALUES
            (v_emitter_id, 'RADIO-001', 'Radio para Auto Bluetooth', '85', '8515', 150.00, '00'),
            (v_emitter_id, 'LAPTOP-001', 'Laptop Gaming 16GB RAM', '84', '8471', 1200.00, '00'),
            (v_emitter_id, 'SERV-001', 'Servicio de Consultoría IT', '94', '9499', 75.00, '00'),
            (v_emitter_id, 'SOFT-001', 'Software de Gestión', '62', '6201', 299.00, '00'),
            (v_emitter_id, 'MANT-001', 'Mantenimiento Preventivo', '94', '9499', 120.00, '00')
        ON CONFLICT (emitter_id, sku) DO NOTHING;
        
        -- Insertar API key de ejemplo para testing
        INSERT INTO api_keys (emitter_id, name, key_hash, rate_limit_per_min) VALUES
            (v_emitter_id, 'Test App', crypt('test_api_key_123', gen_salt('bf')), 1000),
            (v_emitter_id, 'Production App', crypt('prod_api_key_456', gen_salt('bf')), 120)
        ON CONFLICT (emitter_id, name) DO NOTHING;
    END IF;
END $$;

-- Insertar datos de catálogo CPBS (Clasificación Panameña de Bienes y Servicios)
-- Solo algunos ejemplos representativos
CREATE TABLE IF NOT EXISTS cpbs_catalog (
    cpbs_abr VARCHAR(2) NOT NULL,
    cpbs_cmp VARCHAR(4) NOT NULL,
    description TEXT NOT NULL,
    PRIMARY KEY (cpbs_abr, cpbs_cmp)
);

INSERT INTO cpbs_catalog (cpbs_abr, cpbs_cmp, description) VALUES
    ('84', '8471', 'Computadoras portátiles, de bolsillo, de escritorio, servidores y sus unidades'),
    ('85', '8515', 'Equipos de comunicación por radio, televisión y similares'),
    ('94', '9499', 'Servicios de asociaciones, organizaciones y entidades similares'),
    ('62', '6201', 'Servicios de consultores en informática'),
    ('70', '7010', 'Servicios de actividades inmobiliarias realizados por cuenta propia'),
    ('45', '4510', 'Construcción de edificios residenciales'),
    ('47', '4721', 'Venta al por menor de frutas y verduras frescas'),
    ('49', '4921', 'Transporte terrestre de pasajeros por ferrocarril'),
    ('51', '5110', 'Transporte aéreo de pasajeros'),
    ('52', '5210', 'Almacenamiento y depósito')
ON CONFLICT (cpbs_abr, cpbs_cmp) DO NOTHING;

-- Insertar configuración de impuestos ITBMS
CREATE TABLE IF NOT EXISTS tax_rates (
    code VARCHAR(2) NOT NULL PRIMARY KEY,
    description TEXT NOT NULL,
    rate DECIMAL(5,2) NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true
);

INSERT INTO tax_rates (code, description, rate) VALUES
    ('00', 'Exento de ITBMS', 0.00),
    ('01', 'ITBMS 7%', 7.00),
    ('02', 'ITBMS 10%', 10.00),
    ('03', 'ITBMS 15%', 15.00)
ON CONFLICT (code) DO NOTHING;

-- Insertar métodos de pago
CREATE TABLE IF NOT EXISTS payment_methods (
    code VARCHAR(2) NOT NULL PRIMARY KEY,
    description TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true
);

INSERT INTO payment_methods (code, description) VALUES
    ('01', 'Efectivo'),
    ('02', 'Cheque'),
    ('03', 'Transferencia bancaria'),
    ('04', 'Tarjeta de crédito'),
    ('05', 'Tarjeta de débito'),
    ('06', 'Compensación'),
    ('07', 'Permuta'),
    ('08', 'Venta a crédito'),
    ('09', 'Tarjeta prepago'),
    ('10', 'Método mixto')
ON CONFLICT (code) DO NOTHING;

-- Crear vistas útiles para consultas comunes
CREATE OR REPLACE VIEW v_emitter_dashboard AS
SELECT 
    e.id as emitter_id,
    e.name as emitter_name,
    e.company_code,
    es.pto_fac_df,
    es.doc_kind,
    es.issued_count,
    es.authorized_count,
    es.rejected_count,
    es.next_number,
    ROUND(
        CASE 
            WHEN es.issued_count > 0 
            THEN (es.authorized_count::DECIMAL / es.issued_count) * 100 
            ELSE 0 
        END, 2
    ) as success_rate
FROM emitters e
JOIN emitter_series es ON e.id = es.emitter_id
WHERE e.is_active = true AND es.is_active = true
ORDER BY e.name, es.pto_fac_df, es.doc_kind;

CREATE OR REPLACE VIEW v_invoice_summary AS
SELECT 
    i.id,
    i.d_nrodf,
    i.d_ptofacdf,
    i.doc_kind,
    i.status,
    i.email_status,
    i.cufe,
    i.total_amount,
    i.created_at,
    e.name as emitter_name,
    c.name as customer_name,
    c.email as customer_email
FROM invoices i
JOIN emitters e ON i.emitter_id = e.id
JOIN customers c ON i.customer_id = c.id
ORDER BY i.created_at DESC;

-- Crear función para obtener estadísticas de series
CREATE OR REPLACE FUNCTION get_series_stats(
    p_emitter_id UUID DEFAULT NULL,
    p_month VARCHAR(7) DEFAULT NULL
)
RETURNS TABLE (
    emitter_name VARCHAR(255),
    pto_fac_df VARCHAR(3),
    doc_kind document_type,
    issued_count BIGINT,
    authorized_count BIGINT,
    rejected_count BIGINT,
    success_rate DECIMAL(5,2)
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        e.name,
        es.pto_fac_df,
        es.doc_kind,
        es.issued_count,
        es.authorized_count,
        es.rejected_count,
        ROUND(
            CASE 
                WHEN es.issued_count > 0 
                THEN (es.authorized_count::DECIMAL / es.issued_count) * 100 
                ELSE 0 
            END, 2
        ) as success_rate
    FROM emitter_series es
    JOIN emitters e ON es.emitter_id = e.id
    WHERE (p_emitter_id IS NULL OR es.emitter_id = p_emitter_id)
        AND es.is_active = true
    ORDER BY e.name, es.pto_fac_df, es.doc_kind;
END;
$$ LANGUAGE plpgsql;

-- Crear función para validar RUC
CREATE OR REPLACE FUNCTION validate_ruc(
    p_ruc_tipo VARCHAR(1),
    p_ruc_numero VARCHAR(20),
    p_ruc_dv VARCHAR(2)
)
RETURNS BOOLEAN AS $$
BEGIN
    -- Validaciones básicas
    IF p_ruc_tipo NOT IN ('1', '2', '3') THEN
        RETURN FALSE;
    END IF;
    
    IF p_ruc_numero IS NULL OR LENGTH(p_ruc_numero) < 8 THEN
        RETURN FALSE;
    END IF;
    
    IF p_ruc_dv IS NULL OR LENGTH(p_ruc_dv) != 2 THEN
        RETURN FALSE;
    END IF;
    
    -- Aquí se podría agregar algoritmo de validación de dígito verificador
    -- Por ahora retornamos TRUE si pasa las validaciones básicas
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- Crear función para generar folio siguiente
CREATE OR REPLACE FUNCTION get_next_folio(
    p_emitter_id UUID,
    p_pto_fac_df VARCHAR(3),
    p_doc_kind document_type
)
RETURNS VARCHAR(10) AS $$
DECLARE
    next_num INTEGER;
    folio VARCHAR(10);
BEGIN
    -- Obtener y bloquear el siguiente número
    SELECT next_number INTO next_num
    FROM emitter_series
    WHERE emitter_id = p_emitter_id 
        AND pto_fac_df = p_pto_fac_df 
        AND doc_kind = p_doc_kind
        AND is_active = true
    FOR UPDATE;
    
    IF next_num IS NULL THEN
        RAISE EXCEPTION 'Serie no encontrada o inactiva';
    END IF;
    
    -- Generar folio de 10 dígitos con left-pad
    folio := LPAD(next_num::TEXT, 10, '0');
    
    -- Incrementar el contador
    UPDATE emitter_series 
    SET next_number = next_number + 1,
        updated_at = NOW()
    WHERE emitter_id = p_emitter_id 
        AND pto_fac_df = p_pto_fac_df 
        AND doc_kind = p_doc_kind;
    
    RETURN folio;
END;
$$ LANGUAGE plpgsql;
