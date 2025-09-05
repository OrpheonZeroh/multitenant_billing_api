-- Script de configuración del entorno para DGI Service
-- Configura parámetros y permisos específicos

-- Configurar timezone y locale
SET timezone = 'UTC';
SET client_encoding = 'UTF8';

-- Crear usuario específico para la aplicación (opcional, para producción)
-- CREATE USER dgi_app WITH PASSWORD 'dgi_app_password_2024';

-- Crear esquema específico si es necesario
-- CREATE SCHEMA IF NOT EXISTS dgi;

-- Configurar parámetros de performance específicos para DGI
-- Nota: pg_stat_statements requiere que la extensión esté instalada
-- ALTER SYSTEM SET shared_preload_libraries = 'pg_stat_statements';
-- ALTER SYSTEM SET pg_stat_statements.track = 'all';
-- ALTER SYSTEM SET pg_stat_statements.max = 10000;

-- Configurar logging específico para auditoría
ALTER SYSTEM SET log_statement = 'mod';
ALTER SYSTEM SET log_min_duration_statement = 1000;
ALTER SYSTEM SET log_checkpoints = on;
ALTER SYSTEM SET log_connections = on;
ALTER SYSTEM SET log_disconnections = on;
ALTER SYSTEM SET log_lock_waits = on;

-- Configurar autovacuum para tablas de auditoría
ALTER SYSTEM SET autovacuum_vacuum_scale_factor = 0.1;
ALTER SYSTEM SET autovacuum_analyze_scale_factor = 0.05;

-- Configurar work_mem para operaciones complejas
ALTER SYSTEM SET work_mem = '8MB';
ALTER SYSTEM SET maintenance_work_mem = '128MB';

-- Configurar checkpoint para mejor performance
ALTER SYSTEM SET checkpoint_completion_target = 0.9;
ALTER SYSTEM SET wal_buffers = '32MB';

-- Crear índices adicionales para consultas frecuentes
CREATE INDEX IF NOT EXISTS idx_invoices_status_created ON invoices(status, created_at);
CREATE INDEX IF NOT EXISTS idx_invoices_emitter_status_created ON invoices(emitter_id, status, created_at);
CREATE INDEX IF NOT EXISTS idx_email_logs_status_created ON email_logs(status, created_at);
CREATE INDEX IF NOT EXISTS idx_webhooks_event_type_created ON webhooks(event_type, created_at);

-- Crear índices para búsquedas de texto
CREATE INDEX IF NOT EXISTS idx_customers_email_lower ON customers(LOWER(email));
CREATE INDEX IF NOT EXISTS idx_products_sku_lower ON products(LOWER(sku));
CREATE INDEX IF NOT EXISTS idx_products_description_gin ON products USING gin(to_tsvector('spanish', description));

-- Crear índices para consultas de auditoría
CREATE INDEX IF NOT EXISTS idx_audit_logs_table_action_created ON audit_logs(table_name, action, changed_at);
CREATE INDEX IF NOT EXISTS idx_audit_logs_record_created ON audit_logs(record_id, changed_at);

-- Configurar particionamiento por fecha para tablas grandes (opcional, para producción)
-- Esto se puede implementar más adelante cuando el volumen de datos sea alto

-- Crear función para limpiar logs antiguos
CREATE OR REPLACE FUNCTION cleanup_old_logs(
    p_days_to_keep INTEGER DEFAULT 90
)
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER := 0;
BEGIN
    -- Limpiar logs de email antiguos
    DELETE FROM email_logs 
    WHERE created_at < NOW() - INTERVAL '1 day' * p_days_to_keep;
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    
    -- Limpiar webhooks entregados antiguos
    DELETE FROM webhooks 
    WHERE delivered_at IS NOT NULL 
        AND delivered_at < NOW() - INTERVAL '1 day' * p_days_to_keep;
    
    GET DIAGNOSTICS deleted_count = deleted_count + ROW_COUNT;
    
    -- Limpiar logs de auditoría antiguos (mantener más tiempo)
    DELETE FROM audit_logs 
    WHERE changed_at < NOW() - INTERVAL '1 day' * (p_days_to_keep * 2);
    
    GET DIAGNOSTICS deleted_count = deleted_count + ROW_COUNT;
    
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Crear función para obtener estadísticas de uso
CREATE OR REPLACE FUNCTION get_database_stats()
RETURNS TABLE (
    table_name TEXT,
    row_count BIGINT,
    table_size TEXT,
    index_size TEXT,
    total_size TEXT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        schemaname||'.'||tablename as table_name,
        n_tup_ins + n_tup_upd + n_tup_del as row_count,
        pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as table_size,
        pg_size_pretty(pg_indexes_size(schemaname||'.'||tablename)) as index_size,
        pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as total_size
    FROM pg_stat_user_tables
    WHERE schemaname NOT IN ('information_schema', 'pg_catalog')
    ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
END;
$$ LANGUAGE plpgsql;

-- Crear función para validar integridad de datos
CREATE OR REPLACE FUNCTION validate_data_integrity()
RETURNS TABLE (
    issue_type TEXT,
    table_name TEXT,
    record_id UUID,
    description TEXT
) AS $$
BEGIN
    RETURN QUERY
    
    -- Verificar invoices sin items
    SELECT 
        'INVOICE_WITHOUT_ITEMS'::TEXT as issue_type,
        'invoices'::TEXT as table_name,
        i.id as record_id,
        'Invoice ' || i.d_nrodf || ' has no items' as description
    FROM invoices i
    LEFT JOIN invoice_items ii ON i.id = ii.invoice_id
    WHERE ii.id IS NULL
    
    UNION ALL
    
    -- Verificar series con contadores inconsistentes
    SELECT 
        'SERIES_COUNTER_MISMATCH'::TEXT as issue_type,
        'emitter_series'::TEXT as table_name,
        es.id as record_id,
        'Series counter mismatch: issued=' || es.issued_count || 
        ', authorized=' || es.authorized_count || 
        ', rejected=' || es.rejected_count as description
    FROM emitter_series es
    WHERE es.issued_count < (es.authorized_count + es.rejected_count)
    
    UNION ALL
    
    -- Verificar invoices con estado inválido
    SELECT 
        'INVALID_INVOICE_STATUS'::TEXT as issue_type,
        'invoices'::TEXT as table_name,
        i.id as record_id,
        'Invalid status transition for invoice ' || i.d_nrodf as description
    FROM invoices i
    WHERE i.status NOT IN ('RECEIVED', 'PREPARING', 'SENDING_TO_PAC', 'AUTHORIZED', 'REJECTED', 'ERROR');
END;
$$ LANGUAGE plpgsql;

-- Crear función para generar reporte de KPIs
CREATE OR REPLACE FUNCTION generate_kpi_report(
    p_emitter_id UUID DEFAULT NULL,
    p_start_date DATE DEFAULT NULL,
    p_end_date DATE DEFAULT NULL
)
RETURNS TABLE (
    metric_name TEXT,
    metric_value TEXT,
    description TEXT
) AS $$
DECLARE
    start_dt DATE := COALESCE(p_start_date, CURRENT_DATE - INTERVAL '30 days');
    end_dt DATE := COALESCE(p_end_date, CURRENT_DATE);
    total_invoices BIGINT;
    authorized_invoices BIGINT;
    rejected_invoices BIGINT;
    total_amount DECIMAL(15,2);
    avg_processing_time INTERVAL;
BEGIN
    -- Obtener métricas básicas
    SELECT 
        COUNT(*),
        COUNT(*) FILTER (WHERE status = 'AUTHORIZED'),
        COUNT(*) FILTER (WHERE status = 'REJECTED'),
        COALESCE(SUM(total_amount), 0),
        AVG(updated_at - created_at) FILTER (WHERE status IN ('AUTHORIZED', 'REJECTED'))
    INTO total_invoices, authorized_invoices, rejected_invoices, total_amount, avg_processing_time
    FROM invoices
    WHERE created_at::DATE BETWEEN start_dt AND end_dt
        AND (p_emitter_id IS NULL OR emitter_id = p_emitter_id);
    
    RETURN QUERY
    SELECT 
        'Total Documents'::TEXT as metric_name,
        total_invoices::TEXT as metric_value,
        'Total documents created in period'::TEXT as description
    
    UNION ALL
    
    SELECT 
        'Authorization Rate'::TEXT as metric_name,
        CASE 
            WHEN total_invoices > 0 
            THEN ROUND((authorized_invoices::DECIMAL / total_invoices) * 100, 2)::TEXT || '%'
            ELSE '0%'
        END as metric_value,
        'Percentage of documents authorized'::TEXT as description
    
    UNION ALL
    
    SELECT 
        'Total Amount'::TEXT as metric_name,
        '$' || total_amount::TEXT as metric_value,
        'Total amount of all documents'::TEXT as description
    
    UNION ALL
    
    SELECT 
        'Avg Processing Time'::TEXT as metric_name,
        COALESCE(avg_processing_time::TEXT, 'N/A') as metric_value,
        'Average time from creation to final status'::TEXT as description;
END;
$$ LANGUAGE plpgsql;

-- Configurar permisos (para cuando se cree el usuario de aplicación)
-- GRANT CONNECT ON DATABASE dgi_service TO dgi_app;
-- GRANT USAGE ON SCHEMA public TO dgi_app;
-- GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO dgi_app;
-- GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO dgi_app;

-- Crear tabla de configuración del sistema
CREATE TABLE IF NOT EXISTS system_config (
    key VARCHAR(100) PRIMARY KEY,
    value TEXT NOT NULL,
    description TEXT,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

INSERT INTO system_config (key, value, description) VALUES
    ('system.version', '1.0.0', 'Versión actual del sistema'),
    ('system.environment', 'development', 'Entorno actual'),
    ('system.maintenance_mode', 'false', 'Modo mantenimiento activo'),
    ('email.max_retries', '3', 'Máximo de reintentos para emails'),
    ('webhook.max_retries', '5', 'Máximo de reintentos para webhooks'),
    ('pac.timeout_seconds', '30', 'Timeout para llamadas al PAC'),
    ('pdf.cache_ttl_hours', '24', 'TTL del cache de PDFs en horas')
ON CONFLICT (key) DO UPDATE SET 
    value = EXCLUDED.value,
    description = EXCLUDED.description,
    updated_at = NOW();

-- Crear función para actualizar configuración
CREATE OR REPLACE FUNCTION update_system_config(
    p_key VARCHAR(100),
    p_value TEXT,
    p_description TEXT DEFAULT NULL
)
RETURNS BOOLEAN AS $$
BEGIN
    INSERT INTO system_config (key, value, description)
    VALUES (p_key, p_value, COALESCE(p_description, 
        (SELECT description FROM system_config WHERE key = p_key)
    ))
    ON CONFLICT (key) DO UPDATE SET 
        value = EXCLUDED.value,
        description = COALESCE(EXCLUDED.description, system_config.description),
        updated_at = NOW();
    
    RETURN TRUE;
EXCEPTION
    WHEN OTHERS THEN
        RETURN FALSE;
END;
$$ LANGUAGE plpgsql;

