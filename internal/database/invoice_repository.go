package database

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hypernova-labs/dgi-service/internal/models"
	"github.com/sirupsen/logrus"
)

// InvoiceRepository maneja las operaciones de base de datos para Invoice
type InvoiceRepository struct {
	db     *DB
	logger *logrus.Logger
}

// NewInvoiceRepository crea una nueva instancia del repositorio
func NewInvoiceRepository(db *DB, logger *logrus.Logger) *InvoiceRepository {
	return &InvoiceRepository{
		db:     db,
		logger: logger,
	}
}

// Create crea un nuevo invoice con sus items
func (r *InvoiceRepository) Create(invoice *models.Invoice, items []models.InvoiceItem) error {
	return r.db.WithTransaction(func(tx *sql.Tx) error {
		// Insertar invoice
		query := `
			INSERT INTO invoices (
				id, emitter_id, series_id, customer_id, doc_kind, d_nrodf, d_ptofacdf,
				status, email_status, ref_cufe, ref_nrodf, ref_ptofacdf, iamb, itpemis, idoc,
				subtotal, itbms_amount, total_amount, idempotency_key, created_at, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15,
				$16, $17, $18, $19, $20, $21
			)
		`
		
		_, err := tx.Exec(query,
			invoice.ID, invoice.EmitterID, invoice.SeriesID, invoice.CustomerID,
			invoice.DocumentType, invoice.DocumentNumber, invoice.PtoFacDF,
			invoice.Status, invoice.EmailStatus, invoice.ReferenceCUFE, invoice.ReferenceNumber, invoice.ReferencePtoFac,
			invoice.IAmb, invoice.ITpEmis, invoice.IDoc,
			invoice.Subtotal, invoice.ITBMSAmount, invoice.TotalAmount,
			invoice.IdempotencyKey, invoice.CreatedAt, invoice.UpdatedAt,
		)
		
		if err != nil {
			return fmt.Errorf("error inserting invoice: %w", err)
		}

		// Insertar items
		for _, item := range items {
			itemQuery := `
				INSERT INTO invoice_items (
					id, invoice_id, line_no, sku, description, qty, unit_price,
					itbms_rate, cpbs_abr, cpbs_cmp, line_total, created_at
				) VALUES (
					$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
				)
			`
			
			_, err := tx.Exec(itemQuery,
				item.ID, item.InvoiceID, item.LineNo, item.SKU, item.Description,
				item.Quantity, item.UnitPrice, item.ITBMSRate, item.CPBSAbr, item.CPBSCmp,
				item.LineTotal, item.CreatedAt,
			)
			
			if err != nil {
				return fmt.Errorf("error inserting invoice item: %w", err)
			}
		}

		return nil
	})
}

// GetByID obtiene un invoice por ID con sus relaciones
func (r *InvoiceRepository) GetByID(id uuid.UUID) (*models.Invoice, error) {
	query := `
		SELECT 
			i.id, i.emitter_id, i.series_id, i.customer_id, i.doc_kind, i.d_nrodf, i.d_ptofacdf,
			i.status, i.email_status, i.ref_cufe, i.ref_nrodf, i.ref_ptofacdf, i.cufe, i.url_cufe,
			i.xml_in, i.xml_response, i.xml_fe, i.xml_protocolo, i.cafe_pdf_url,
			i.iamb, i.itpemis, i.idoc, i.subtotal, i.itbms_amount, i.total_amount,
			i.idempotency_key, i.created_at, i.updated_at,
			e.name as emitter_name, e.company_code as emitter_company_code,
			c.name as customer_name, c.email as customer_email
		FROM invoices i
		JOIN emitters e ON i.emitter_id = e.id
		JOIN customers c ON i.customer_id = c.id
		WHERE i.id = $1
	`
	
	var invoice models.Invoice
	var emitter models.Emitter
	var customer models.Customer
	
	err := r.db.QueryRowWithTimeout(query, id).Scan(
		&invoice.ID, &invoice.EmitterID, &invoice.SeriesID, &invoice.CustomerID,
		&invoice.DocumentType, &invoice.DocumentNumber, &invoice.PtoFacDF,
		&invoice.Status, &invoice.EmailStatus, &invoice.ReferenceCUFE, &invoice.ReferenceNumber, &invoice.ReferencePtoFac,
		&invoice.CUFE, &invoice.URLCUFE, &invoice.XMLIn, &invoice.XMLResponse, &invoice.XMLFE, &invoice.XMLProtocolo, &invoice.CAFEPDFURL,
		&invoice.IAmb, &invoice.ITpEmis, &invoice.IDoc, &invoice.Subtotal, &invoice.ITBMSAmount, &invoice.TotalAmount,
		&invoice.IdempotencyKey, &invoice.CreatedAt, &invoice.UpdatedAt,
		&emitter.Name, &emitter.CompanyCode, &customer.Name, &customer.Email,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invoice not found: %s", id)
		}
		return nil, fmt.Errorf("error querying invoice: %w", err)
	}

	// Obtener items
	items, err := r.GetItemsByInvoiceID(id)
	if err != nil {
		r.logger.Warnf("Error getting items for invoice %s: %v", id, err)
	}

	invoice.Items = items
	invoice.Emitter = &emitter
	invoice.Customer = &customer

	return &invoice, nil
}

// GetByIdempotencyKey obtiene un invoice por clave de idempotencia
func (r *InvoiceRepository) GetByIdempotencyKey(key string) (*models.Invoice, error) {
	query := `
		SELECT id FROM invoices WHERE idempotency_key = $1
	`
	
	var id uuid.UUID
	err := r.db.QueryRowWithTimeout(query, key).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("error querying invoice by idempotency key: %w", err)
	}

	return r.GetByID(id)
}

// GetItemsByInvoiceID obtiene los items de un invoice
func (r *InvoiceRepository) GetItemsByInvoiceID(invoiceID uuid.UUID) ([]models.InvoiceItem, error) {
	query := `
		SELECT id, invoice_id, line_no, sku, description, qty, unit_price,
			   itbms_rate, cpbs_abr, cpbs_cmp, line_total, created_at
		FROM invoice_items
		WHERE invoice_id = $1
		ORDER BY line_no
	`
	
	rows, err := r.db.QueryWithTimeout(query, invoiceID)
	if err != nil {
		return nil, fmt.Errorf("error querying invoice items: %w", err)
	}
	defer rows.Close()

	var items []models.InvoiceItem
	for rows.Next() {
		var item models.InvoiceItem
		err := rows.Scan(
			&item.ID, &item.InvoiceID, &item.LineNo, &item.SKU, &item.Description,
			&item.Quantity, &item.UnitPrice, &item.ITBMSRate, &item.CPBSAbr, &item.CPBSCmp,
			&item.LineTotal, &item.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning invoice item: %w", err)
		}
		items = append(items, item)
	}

	return items, nil
}

// UpdateStatus actualiza el estado de un invoice
func (r *InvoiceRepository) UpdateStatus(id uuid.UUID, status models.DocumentStatus) error {
	query := `
		UPDATE invoices 
		SET status = $1, updated_at = $2
		WHERE id = $3
	`
	
	result, err := r.db.ExecWithTimeout(query, status, time.Now(), id)
	if err != nil {
		return fmt.Errorf("error updating invoice status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("invoice not found: %s", id)
	}

	return nil
}

// UpdatePACResponse actualiza la respuesta del PAC
func (r *InvoiceRepository) UpdatePACResponse(id uuid.UUID, cufe, urlCUFE, xmlFE, xmlProtocolo string) error {
	query := `
		UPDATE invoices 
		SET cufe = $1, url_cufe = $2, xml_fe = $3, xml_protocolo = $4, updated_at = $5
		WHERE id = $3
	`
	
	result, err := r.db.ExecWithTimeout(query, cufe, urlCUFE, xmlFE, xmlProtocolo, time.Now(), id)
	if err != nil {
		return fmt.Errorf("error updating PAC response: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("invoice not found: %s", id)
	}

	return nil
}

// UpdateEmailStatus actualiza el estado del email
func (r *InvoiceRepository) UpdateEmailStatus(id uuid.UUID, status models.EmailStatus) error {
	query := `
		UPDATE invoices 
		SET email_status = $1, updated_at = $2
		WHERE id = $3
	`
	
	result, err := r.db.ExecWithTimeout(query, status, time.Now(), id)
	if err != nil {
		return fmt.Errorf("error updating email status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("invoice not found: %s", id)
	}

	return nil
}

// GetByEmitterID obtiene invoices por emisor con paginación
func (r *InvoiceRepository) GetByEmitterID(emitterID uuid.UUID, page, pageSize int) ([]models.Invoice, int, error) {
	// Contar total
	countQuery := `SELECT COUNT(*) FROM invoices WHERE emitter_id = $1`
	var total int
	err := r.db.QueryRowWithTimeout(countQuery, emitterID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("error counting invoices: %w", err)
	}

	// Obtener invoices
	offset := (page - 1) * pageSize
	query := `
		SELECT id, doc_kind, d_nrodf, d_ptofacdf, status, email_status, 
			   subtotal, itbms_amount, total_amount, created_at
		FROM invoices
		WHERE emitter_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`
	
	rows, err := r.db.QueryWithTimeout(query, emitterID, pageSize, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("error querying invoices: %w", err)
	}
	defer rows.Close()

	var invoices []models.Invoice
	for rows.Next() {
		var invoice models.Invoice
		err := rows.Scan(
			&invoice.ID, &invoice.DocumentType, &invoice.DocumentNumber, &invoice.PtoFacDF,
			&invoice.Status, &invoice.EmailStatus, &invoice.Subtotal, &invoice.ITBMSAmount,
			&invoice.TotalAmount, &invoice.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("error scanning invoice: %w", err)
		}
		invoices = append(invoices, invoice)
	}

	return invoices, total, nil
}

// GetNextDocumentNumber obtiene el siguiente número de documento para una serie
func (r *InvoiceRepository) GetNextDocumentNumber(emitterID uuid.UUID, ptoFacDF string, docKind models.DocumentType) (string, error) {
	query := `
		SELECT get_next_folio($1, $2, $3)
	`
	
	var nextNumber string
	err := r.db.QueryRowWithTimeout(query, emitterID, ptoFacDF, docKind).Scan(&nextNumber)
	if err != nil {
		return "", fmt.Errorf("error getting next document number: %w", err)
	}

	return nextNumber, nil
}

// SearchByFilters busca invoices con filtros
func (r *InvoiceRepository) SearchByFilters(filters map[string]interface{}, page, pageSize int) ([]models.Invoice, int, error) {
	// Construir query dinámica
	whereClauses := []string{"1=1"}
	args := []interface{}{}
	argIndex := 1

	for key, value := range filters {
		switch key {
		case "emitter_id":
			whereClauses = append(whereClauses, fmt.Sprintf("emitter_id = $%d", argIndex))
			args = append(args, value)
			argIndex++
		case "status":
			whereClauses = append(whereClauses, fmt.Sprintf("status = $%d", argIndex))
			args = append(args, value)
			argIndex++
		case "document_type":
			whereClauses = append(whereClauses, fmt.Sprintf("doc_kind = $%d", argIndex))
			args = append(args, value)
			argIndex++
		case "date_from":
			whereClauses = append(whereClauses, fmt.Sprintf("created_at >= $%d", argIndex))
			args = append(args, value)
			argIndex++
		case "date_to":
			whereClauses = append(whereClauses, fmt.Sprintf("created_at <= $%d", argIndex))
			args = append(args, value)
			argIndex++
		}
	}

	whereClause := strings.Join(whereClauses, " AND ")

	// Contar total
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM invoices WHERE %s", whereClause)
	var total int
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	err := r.db.QueryRowWithTimeout(countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("error counting invoices: %w", err)
	}

	// Obtener invoices
	offset := (page - 1) * pageSize
	queryArgs := append(args, pageSize, offset)
	query := fmt.Sprintf(`
		SELECT id, doc_kind, d_nrodf, d_ptofacdf, status, email_status, 
			   subtotal, itbms_amount, total_amount, created_at
		FROM invoices
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIndex, argIndex+1)
	
	rows, err := r.db.QueryWithTimeout(query, queryArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("error querying invoices: %w", err)
	}
	defer rows.Close()

	var invoices []models.Invoice
	for rows.Next() {
		var invoice models.Invoice
		err := rows.Scan(
			&invoice.ID, &invoice.DocumentType, &invoice.DocumentNumber, &invoice.PtoFacDF,
			&invoice.Status, &invoice.EmailStatus, &invoice.Subtotal, &invoice.ITBMSAmount,
			&invoice.TotalAmount, &invoice.CreatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("error scanning invoices: %w", err)
		}
		invoices = append(invoices, invoice)
	}

	return invoices, total, nil
}
