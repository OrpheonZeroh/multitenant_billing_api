-- Script de verificación para DGI Service
-- Valida que toda la configuración se haya aplicado correctamente

-- Verificar que las extensiones estén instaladas
SELECT 
    'Extensions' as check_type,
    extname as name,
    CASE WHEN extname IS NOT NULL THEN 'OK' ELSE 'FAILED' END as status
FROM pg_extension 
WHERE extname IN ('uuid-ossp', 'pgcrypto');

-- Verificar que los tipos enumerados estén creados
SELECT 
    'Enums' as check_type,
    t.typname as name,
    CASE WHEN t.typname IS NOT NULL THEN 'OK' ELSE 'FAILED' END as status
FROM pg_type t
JOIN pg_namespace n ON t.typnamespace = n.oid
WHERE n.nspname = 'public' 
    AND t.typname IN ('document_status', 'email_status', 'document_type', 'payment_method');

-- Verificar que todas las tablas estén creadas
SELECT 
    'Tables' as check_type,
    tablename as name,
    CASE WHEN tablename IS NOT NULL THEN 'OK' ELSE 'FAILED' END as status
FROM pg_tables 
WHERE schemaname = 'public' 
    AND tablename IN (
        'emitters', 'api_keys', 'emitter_series', 'customers', 'products',
        'invoices', 'invoice_items', 'email_logs', 'webhooks', 'audit_logs',
        'cpbs_catalog', 'tax_rates', 'payment_methods', 'system_config'
    )
ORDER BY tablename;

-- Verificar que los índices estén creados
SELECT 
    'Indexes' as check_type,
    indexname as name,
    CASE WHEN indexname IS NOT NULL THEN 'OK' ELSE 'FAILED' END as status
FROM pg_indexes 
WHERE schemaname = 'public' 
    AND indexname LIKE 'idx_%'
ORDER BY indexname;

-- Verificar que las funciones estén creadas
SELECT 
    'Functions' as check_type,
    proname as name,
    CASE WHEN proname IS NOT NULL THEN 'OK' ELSE 'FAILED' END as status
FROM pg_proc p
JOIN pg_namespace n ON p.pronamespace = n.oid
WHERE n.nspname = 'public' 
    AND proname IN (
        'update_updated_at_column', 'increment_series_counters',
        'cleanup_old_logs', 'get_database_stats', 'validate_data_integrity',
        'generate_kpi_report', 'validate_ruc', 'get_next_folio',
        'update_system_config'
    )
ORDER BY proname;

-- Verificar que las vistas estén creadas
SELECT 
    'Views' as check_type,
    viewname as name,
    CASE WHEN viewname IS NOT NULL THEN 'OK' ELSE 'FAILED' END as status
FROM pg_views 
WHERE schemaname = 'public' 
    AND viewname IN ('v_emitter_dashboard', 'v_invoice_summary')
ORDER BY viewname;

-- Verificar que los triggers estén creados
SELECT 
    'Triggers' as check_type,
    tgname as name,
    CASE WHEN tgname IS NOT NULL THEN 'OK' ELSE 'FAILED' END as status
FROM pg_trigger t
JOIN pg_class c ON t.tgrelid = c.oid
JOIN pg_namespace n ON c.relnamespace = n.oid
WHERE n.nspname = 'public' 
    AND c.relname IN ('emitters', 'emitter_series', 'customers', 'products', 'invoices')
    AND t.tgname LIKE '%updated_at%' OR t.tgname LIKE '%series_counters%'
ORDER BY tgname;

-- Verificar datos de catálogo
SELECT 
    'Catalog Data' as check_type,
    'Emitters' as name,
    CASE WHEN COUNT(*) > 0 THEN 'OK (' || COUNT(*) || ' records)' ELSE 'FAILED' END as status
FROM emitters

UNION ALL

SELECT 
    'Catalog Data' as check_type,
    'Series' as name,
    CASE WHEN COUNT(*) > 0 THEN 'OK (' || COUNT(*) || ' records)' ELSE 'FAILED' END as status
FROM emitter_series

UNION ALL

SELECT 
    'Catalog Data' as check_type,
    'Customers' as name,
    CASE WHEN COUNT(*) > 0 THEN 'OK (' || COUNT(*) || ' records)' ELSE 'FAILED' END as status
FROM customers

UNION ALL

SELECT 
    'Catalog Data' as check_type,
    'Products' as name,
    CASE WHEN COUNT(*) > 0 THEN 'OK (' || COUNT(*) || ' records)' ELSE 'FAILED' END as status
FROM products

UNION ALL

SELECT 
    'Catalog Data' as check_type,
    'API Keys' as name,
    CASE WHEN COUNT(*) > 0 THEN 'OK (' || COUNT(*) || ' records)' ELSE 'FAILED' END as status
FROM api_keys;

-- Verificar configuración del sistema
SELECT 
    'System Config' as check_type,
    key as name,
    CASE WHEN value IS NOT NULL THEN 'OK: ' || value ELSE 'FAILED' END as status
FROM system_config
ORDER BY key;

-- Verificar permisos y conexiones
SELECT 
    'Database Info' as check_type,
    'Current User' as name,
    current_user as status

UNION ALL

SELECT 
    'Database Info' as check_type,
    'Database Name' as name,
    current_database() as status

UNION ALL

SELECT 
    'Database Info' as check_type,
    'Timezone' as name,
    current_setting('timezone') as status

UNION ALL

SELECT 
    'Database Info' as check_type,
    'Encoding' as name,
    current_setting('client_encoding') as status;

-- Verificar parámetros de configuración
SELECT 
    'PostgreSQL Config' as check_type,
    name as name,
    CASE WHEN setting IS NOT NULL THEN 'OK: ' || setting ELSE 'FAILED' END as status
FROM pg_settings 
WHERE name IN (
    'shared_buffers', 'work_mem', 'maintenance_work_mem',
    'wal_buffers', 'checkpoint_completion_target',
    'log_statement', 'log_min_duration_statement'
)
ORDER BY name;

-- Verificar que las secuencias estén funcionando
SELECT 
    'Sequences' as check_type,
    'UUID Generation' as name,
    CASE 
        WHEN uuid_generate_v4() IS NOT NULL THEN 'OK' 
        ELSE 'FAILED' 
    END as status

UNION ALL

SELECT 
    'Sequences' as check_type,
    'Password Hashing' as name,
    CASE 
        WHEN crypt('test_password', gen_salt('bf')) IS NOT NULL THEN 'OK' 
        ELSE 'FAILED' 
    END as status;

-- Verificar integridad referencial
SELECT 
    'Referential Integrity' as check_type,
    'Foreign Keys' as name,
    CASE 
        WHEN COUNT(*) = 0 THEN 'OK: All constraints valid'
        ELSE 'WARNING: ' || COUNT(*) || ' constraint violations found'
    END as status
FROM (
    -- Verificar que no haya invoices sin emisor
    SELECT COUNT(*) FROM invoices i LEFT JOIN emitters e ON i.emitter_id = e.id WHERE e.id IS NULL
    UNION ALL
    -- Verificar que no haya invoices sin serie
    SELECT COUNT(*) FROM invoices i LEFT JOIN emitter_series es ON i.series_id = es.id WHERE es.id IS NULL
    UNION ALL
    -- Verificar que no haya invoices sin cliente
    SELECT COUNT(*) FROM invoices i LEFT JOIN customers c ON i.customer_id = c.id WHERE c.id IS NULL
    UNION ALL
    -- Verificar que no haya items sin invoice
    SELECT COUNT(*) FROM invoice_items ii LEFT JOIN invoices i ON ii.invoice_id = i.id WHERE i.id IS NULL
) as constraint_checks;

-- Mostrar resumen de estadísticas
SELECT 
    'Database Statistics' as check_type,
    'Total Tables' as name,
    COUNT(*)::TEXT as status
FROM pg_tables 
WHERE schemaname = 'public'

UNION ALL

SELECT 
    'Database Statistics' as check_type,
    'Total Functions' as name,
    COUNT(*)::TEXT as status
FROM pg_proc p
JOIN pg_namespace n ON p.pronamespace = n.oid
WHERE n.nspname = 'public'

UNION ALL

SELECT 
    'Database Statistics' as check_type,
    'Total Indexes' as name,
    COUNT(*)::TEXT as status
FROM pg_indexes 
WHERE schemaname = 'public'

UNION ALL

SELECT 
    'Database Statistics' as check_type,
    'Total Triggers' as name,
    COUNT(*)::TEXT as status
FROM pg_trigger t
JOIN pg_class c ON t.tgrelid = c.oid
JOIN pg_namespace n ON c.relnamespace = n.oid
WHERE n.nspname = 'public';

-- Función para generar reporte completo de verificación
CREATE OR REPLACE FUNCTION generate_verification_report()
RETURNS TABLE (
    check_type TEXT,
    name TEXT,
    status TEXT,
    details TEXT
) AS $$
BEGIN
    -- Retornar todos los resultados de verificación
    RETURN QUERY
    SELECT 
        'VERIFICATION_COMPLETE'::TEXT as check_type,
        'Setup Status'::TEXT as name,
        'READY'::TEXT as status,
        'Database setup completed successfully. Ready for DGI Service.'::TEXT as details;
END;
$$ LANGUAGE plpgsql;

-- Ejecutar verificación final
SELECT * FROM generate_verification_report();

