-- Crear tabla para archivos de facturas
CREATE TABLE IF NOT EXISTS invoice_files (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    invoice_id UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    pdf_data BYTEA,
    xml_data BYTEA,
    pdf_size BIGINT DEFAULT 0,
    xml_size BIGINT DEFAULT 0,
    generated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Crear índices para mejorar performance
CREATE INDEX IF NOT EXISTS idx_invoice_files_invoice_id ON invoice_files(invoice_id);
CREATE INDEX IF NOT EXISTS idx_invoice_files_generated_at ON invoice_files(generated_at);

-- Crear trigger para actualizar updated_at
CREATE OR REPLACE FUNCTION update_invoice_files_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_invoice_files_updated_at
    BEFORE UPDATE ON invoice_files
    FOR EACH ROW
    EXECUTE FUNCTION update_invoice_files_updated_at();

-- Comentarios
COMMENT ON TABLE invoice_files IS 'Almacena archivos PDF y XML generados para facturas';
COMMENT ON COLUMN invoice_files.pdf_data IS 'Contenido binario del archivo PDF';
COMMENT ON COLUMN invoice_files.xml_data IS 'Contenido del archivo XML';
COMMENT ON COLUMN invoice_files.pdf_size IS 'Tamaño del archivo PDF en bytes';
COMMENT ON COLUMN invoice_files.xml_size IS 'Tamaño del archivo XML en bytes';
