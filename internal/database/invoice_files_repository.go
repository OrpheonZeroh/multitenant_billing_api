package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hypernova-labs/dgi-service/internal/models"
	"github.com/sirupsen/logrus"
)

// InvoiceFilesRepository maneja las operaciones de base de datos para archivos de facturas
type InvoiceFilesRepository struct {
	db     *DB
	logger *logrus.Logger
}

// NewInvoiceFilesRepository crea una nueva instancia del repositorio
func NewInvoiceFilesRepository(db *DB, logger *logrus.Logger) *InvoiceFilesRepository {
	return &InvoiceFilesRepository{
		db:     db,
		logger: logger,
	}
}

// CreateOrUpdate crea o actualiza los archivos de una factura (UPSERT)
func (r *InvoiceFilesRepository) CreateOrUpdate(files *models.InvoiceFiles) error {
	// Primero verificar si ya existen archivos
	exists, err := r.Exists(files.InvoiceID)
	if err != nil {
		return fmt.Errorf("error checking existence: %w", err)
	}
	
	if exists {
		// Actualizar archivos existentes
		return r.Update(files)
	} else {
		// Crear nuevos archivos
		return r.Create(files)
	}
}

// Create crea un nuevo registro de archivos de factura
func (r *InvoiceFilesRepository) Create(files *models.InvoiceFiles) error {
	query := `
		INSERT INTO invoice_files (
			id, invoice_id, pdf_data, xml_data, pdf_size, xml_size, 
			pdf_url, xml_url, generated_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		)
	`
	
	_, err := r.db.ExecWithTimeout(query,
		files.ID, files.InvoiceID, files.PDFData, files.XMLData,
		files.PDFSize, files.XMLSize, files.PDFURL, files.XMLURL,
		files.GeneratedAt, files.UpdatedAt,
	)
	
	if err != nil {
		return fmt.Errorf("error creating invoice files: %w", err)
	}
	
	return nil
}

// GetByInvoiceID obtiene los archivos de una factura
func (r *InvoiceFilesRepository) GetByInvoiceID(invoiceID uuid.UUID) (*models.InvoiceFiles, error) {
	query := `
		SELECT id, invoice_id, pdf_data, xml_data, pdf_size, xml_size, 
			   pdf_url, xml_url, generated_at, updated_at
		FROM invoice_files
		WHERE invoice_id = $1
	`
	
	var files models.InvoiceFiles
	err := r.db.QueryRowWithTimeout(query, invoiceID).Scan(
		&files.ID, &files.InvoiceID, &files.PDFData, &files.XMLData,
		&files.PDFSize, &files.XMLSize, &files.PDFURL, &files.XMLURL,
		&files.GeneratedAt, &files.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invoice files not found for invoice %s", invoiceID)
		}
		return nil, fmt.Errorf("error querying invoice files: %w", err)
	}
	
	return &files, nil
}

// Update actualiza los archivos de una factura
func (r *InvoiceFilesRepository) Update(files *models.InvoiceFiles) error {
	query := `
		UPDATE invoice_files 
		SET pdf_data = $1, xml_data = $2, pdf_size = $3, xml_size = $4, 
		    pdf_url = $5, xml_url = $6, updated_at = $7
		WHERE invoice_id = $8
	`
	
	_, err := r.db.ExecWithTimeout(query,
		files.PDFData, files.XMLData, files.PDFSize, files.XMLSize,
		files.PDFURL, files.XMLURL, time.Now(), files.InvoiceID,
	)
	
	if err != nil {
		return fmt.Errorf("error updating invoice files: %w", err)
	}
	
	return nil
}

// Delete elimina los archivos de una factura
func (r *InvoiceFilesRepository) Delete(invoiceID uuid.UUID) error {
	query := `DELETE FROM invoice_files WHERE invoice_id = $1`
	
	_, err := r.db.ExecWithTimeout(query, invoiceID)
	if err != nil {
		return fmt.Errorf("error deleting invoice files: %w", err)
	}
	
	return nil
}

// Exists verifica si existen archivos para una factura
func (r *InvoiceFilesRepository) Exists(invoiceID uuid.UUID) (bool, error) {
	query := `SELECT COUNT(*) FROM invoice_files WHERE invoice_id = $1`
	
	var count int
	err := r.db.QueryRowWithTimeout(query, invoiceID).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("error checking invoice files existence: %w", err)
	}
	
	return count > 0, nil
}
