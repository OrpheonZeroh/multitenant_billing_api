package models

import (
	"time"

	"github.com/google/uuid"
)

// InvoiceFiles representa los archivos generados de una factura
type InvoiceFiles struct {
	ID          uuid.UUID `json:"id" db:"id"`
	InvoiceID   uuid.UUID `json:"invoice_id" db:"invoice_id"`
	PDFData     []byte    `json:"pdf_data" db:"pdf_data"`
	XMLData     []byte    `json:"xml_data" db:"xml_data"`
	PDFSize     int64     `json:"pdf_size" db:"pdf_size"`
	XMLSize     int64     `json:"xml_size" db:"xml_size"`
	PDFURL      *string   `json:"pdf_url" db:"pdf_url"`
	XMLURL      *string   `json:"xml_url" db:"xml_url"`
	GeneratedAt time.Time `json:"generated_at" db:"generated_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// FileDownloadRequest representa la solicitud de descarga
type FileDownloadRequest struct {
	FileType string `json:"file_type" binding:"required,oneof=pdf xml"`
}
