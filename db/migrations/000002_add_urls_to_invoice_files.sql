-- Agregar columnas para URLs de Supabase
ALTER TABLE invoice_files 
ADD COLUMN IF NOT EXISTS pdf_url TEXT,
ADD COLUMN IF NOT EXISTS xml_url TEXT;

-- Crear índices para las nuevas columnas
CREATE INDEX IF NOT EXISTS idx_invoice_files_pdf_url ON invoice_files(pdf_url);
CREATE INDEX IF NOT EXISTS idx_invoice_files_xml_url ON invoice_files(xml_url);

-- Comentarios para documentar las nuevas columnas
COMMENT ON COLUMN invoice_files.pdf_url IS 'URL del archivo PDF en Supabase Storage';
COMMENT ON COLUMN invoice_files.xml_url IS 'URL del archivo XML en Supabase Storage';
