package models

import (
	"time"

	"github.com/google/uuid"
)

// DocumentType representa el tipo de documento fiscal
type DocumentType string

const (
	DocumentTypeInvoice        DocumentType = "invoice"
	DocumentTypeImportInvoice  DocumentType = "import_invoice"
	DocumentTypeExportInvoice  DocumentType = "export_invoice"
	DocumentTypeCreditNote     DocumentType = "credit_note"
	DocumentTypeDebitNote      DocumentType = "debit_note"
	DocumentTypeZoneFranca     DocumentType = "zone_franca"
	DocumentTypeReembolso      DocumentType = "reembolso"
	DocumentTypeForeignInvoice DocumentType = "foreign_invoice"
)

// DocumentStatus representa el estado del documento
type DocumentStatus string

const (
	DocumentStatusReceived      DocumentStatus = "RECEIVED"
	DocumentStatusPreparing     DocumentStatus = "PREPARING"
	DocumentStatusSendingToPAC  DocumentStatus = "SENDING_TO_PAC"
	DocumentStatusAuthorized    DocumentStatus = "AUTHORIZED"
	DocumentStatusRejected      DocumentStatus = "REJECTED"
	DocumentStatusError         DocumentStatus = "ERROR"
)

// EmailStatus representa el estado del email
type EmailStatus string

const (
	EmailStatusPending EmailStatus = "PENDING"
	EmailStatusSent    EmailStatus = "SENT"
	EmailStatusFailed  EmailStatus = "FAILED"
	EmailStatusRetrying EmailStatus = "RETRYING"
)

// PaymentMethod representa el método de pago
type PaymentMethod string

const (
	PaymentMethodCash           PaymentMethod = "01"
	PaymentMethodCheck          PaymentMethod = "02"
	PaymentMethodBankTransfer   PaymentMethod = "03"
	PaymentMethodCreditCard     PaymentMethod = "04"
	PaymentMethodDebitCard      PaymentMethod = "05"
	PaymentMethodCompensation   PaymentMethod = "06"
	PaymentMethodBarter         PaymentMethod = "07"
	PaymentMethodCreditSale     PaymentMethod = "08"
	PaymentMethodPrepaidCard    PaymentMethod = "09"
	PaymentMethodMixed          PaymentMethod = "10"
)

// Invoice representa un documento fiscal (factura, nota, etc.)
type Invoice struct {
	ID              uuid.UUID      `json:"id" db:"id"`
	EmitterID       uuid.UUID      `json:"emitter_id" db:"emitter_id"`
	SeriesID        uuid.UUID      `json:"series_id" db:"series_id"`
	CustomerID      uuid.UUID      `json:"customer_id" db:"customer_id"`
	
	// Información del documento
	DocumentType    DocumentType   `json:"document_type" db:"doc_kind"`
	DocumentNumber  string         `json:"document_number" db:"d_nrodf"`
	PtoFacDF        string         `json:"pto_fac_df" db:"d_ptofacdf"`
	Status          DocumentStatus `json:"status" db:"status"`
	EmailStatus     EmailStatus    `json:"email_status" db:"email_status"`
	
	// Referencias para notas
	ReferenceCUFE   *string        `json:"reference_cufe,omitempty" db:"ref_cufe"`
	ReferenceNumber *string        `json:"reference_number,omitempty" db:"ref_nrodf"`
	ReferencePtoFac *string        `json:"reference_pto_fac,omitempty" db:"ref_ptofacdf"`
	
	// Respuesta del PAC
	CUFE            *string        `json:"cufe,omitempty" db:"cufe"`
	URLCUFE         *string        `json:"url_cufe,omitempty" db:"url_cufe"`
	XMLIn           *string        `json:"xml_in,omitempty" db:"xml_in"`
	XMLResponse     *string        `json:"xml_response,omitempty" db:"xml_response"`
	XMLFE           *string        `json:"xml_fe,omitempty" db:"xml_fe"`
	XMLProtocolo    *string        `json:"xml_protocolo,omitempty" db:"xml_protocolo"`
	CAFEPDFURL      *string        `json:"cafe_pdf_url,omitempty" db:"cafe_pdf_url"`
	
	// Configuración DGI
	IAmb            int            `json:"i_amb" db:"iamb"`
	ITpEmis         string         `json:"i_tp_emis" db:"itpemis"`
	IDoc            string         `json:"i_doc" db:"idoc"`
	
	// Totales calculados
	Subtotal        float64        `json:"subtotal" db:"subtotal"`
	ITBMSAmount     float64        `json:"itbms_amount" db:"itbms_amount"`
	TotalAmount     float64        `json:"total_amount" db:"total_amount"`
	
	// Metadatos
	IdempotencyKey  *string        `json:"idempotency_key,omitempty" db:"idempotency_key"`
	CreatedAt       time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at" db:"updated_at"`
	
	// Relaciones (populadas en consultas)
	Emitter         *Emitter       `json:"emitter,omitempty"`
	Customer        *Customer      `json:"customer,omitempty"`
	Items           []InvoiceItem  `json:"items,omitempty"`
}

// InvoiceItem representa una línea de un documento
type InvoiceItem struct {
	ID          uuid.UUID `json:"id" db:"id"`
	InvoiceID   uuid.UUID `json:"invoice_id" db:"invoice_id"`
	LineNo      int       `json:"line_no" db:"line_no"`
	SKU         *string   `json:"sku,omitempty" db:"sku"`
	Description string    `json:"description" db:"description"`
	Quantity    float64   `json:"quantity" db:"qty"`
	UnitPrice   float64   `json:"unit_price" db:"unit_price"`
	ITBMSRate   string    `json:"itbms_rate" db:"itbms_rate"`
	CPBSAbr     *string   `json:"cpbs_abr,omitempty" db:"cpbs_abr"`
	CPBSCmp     *string   `json:"cpbs_cmp,omitempty" db:"cpbs_cmp"`
	LineTotal   float64   `json:"line_total" db:"line_total"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// CreateInvoiceRequest representa el request para crear un documento
type CreateInvoiceRequest struct {
	DocumentType DocumentType     `json:"document_type" binding:"required,oneof=invoice import_invoice export_invoice credit_note debit_note zone_franca reembolso foreign_invoice"`
	Reference    *Reference       `json:"reference,omitempty"`
	Customer     CustomerRequest  `json:"customer" binding:"required"`
	Items        []ItemRequest    `json:"items" binding:"required,min=1"`
	Payment      PaymentRequest   `json:"payment" binding:"required"`
	Overrides    *Overrides       `json:"overrides,omitempty"`
}

// Reference representa la referencia para notas de crédito/débito
type Reference struct {
	CUFE    string `json:"cufe" binding:"required"`
	Number  string `json:"nrodf" binding:"required"`
	PtoFac  string `json:"pto_fac_df" binding:"required"`
}

// CustomerRequest representa el request para crear/actualizar un cliente
type CustomerRequest struct {
	Name    string  `json:"name" binding:"required"`
	Email   string  `json:"email" binding:"required,email"`
	Phone   *string `json:"phone,omitempty"`
	Address *string `json:"address,omitempty"`
	UBICode *string `json:"ubi_code,omitempty"`
}

// ItemRequest representa el request para un ítem del documento
type ItemRequest struct {
	ProductID   *string  `json:"product_id,omitempty"`
	SKU         *string  `json:"sku,omitempty"`
	Description string   `json:"description" binding:"required"`
	Quantity    float64  `json:"quantity" binding:"required,gt=0"`
	UnitPrice   float64  `json:"unit_price" binding:"required,gt=0"`
	TaxRate     string   `json:"tax_rate" binding:"required"`
}

// PaymentRequest representa el request para el pago
type PaymentRequest struct {
	Method string  `json:"method" binding:"required,oneof=01 02 03 04 05 06 07 08 09 10"`
	Amount float64 `json:"amount" binding:"required,gt=0"`
}

// Overrides representa configuraciones que sobrescriben los defaults
type Overrides struct {
	PtoFacDF string `json:"pto_fac_df,omitempty"`
	ITpEmis  string `json:"i_tp_emis,omitempty"`
	IDoc     string `json:"i_doc,omitempty"`
}

// InvoiceResponse representa la respuesta al crear un documento
type InvoiceResponse struct {
	ID           uuid.UUID     `json:"id"`
	Status       DocumentStatus `json:"status"`
	DocumentType DocumentType   `json:"document_type"`
	Emitter      EmitterInfo   `json:"emitter"`
	Totals       Totals        `json:"totals"`
	Links        Links         `json:"links"`
}

// EmitterInfo representa información del emisor en la respuesta
type EmitterInfo struct {
	RUC    string `json:"ruc"`
	PtoFac string `json:"pto_fac_df"`
	Number string `json:"nrodf"`
}

// Totals representa los totales del documento
type Totals struct {
	Net   float64 `json:"net"`
	ITBMS float64 `json:"itbms"`
	Total float64 `json:"total"`
}

// Links representa los enlaces relacionados
type Links struct {
	Self  string `json:"self"`
	Files string `json:"files"`
}

// InvoiceStatusResponse representa la respuesta al consultar estado
type InvoiceStatusResponse struct {
	ID           uuid.UUID     `json:"id"`
	Status       DocumentStatus `json:"status"`
	EmailStatus  EmailStatus   `json:"email_status"`
	DocumentType DocumentType   `json:"document_type"`
	CUFE         *string       `json:"cufe,omitempty"`
	URLCUFE      *string       `json:"url_cufe,omitempty"`
	Emitter      EmitterInfo   `json:"emitter"`
	Totals       Totals        `json:"totals"`
	CreatedAt    time.Time     `json:"created_at"`
	Links        Links         `json:"links"`
}

// InvoiceFilesResponse representa la respuesta para obtener archivos
type InvoiceFilesResponse struct {
	XMLFE        *string `json:"xml_fe,omitempty"`
	XMLProtocolo *string `json:"xml_protocolo,omitempty"`
	CAFEPDFURL   *string `json:"cafe_pdf_url,omitempty"`
	Disposition  string  `json:"disposition"`
}

// EmailResendRequest representa el request para reenviar email
type EmailResendRequest struct {
	To *string   `json:"to,omitempty"`
	CC []string `json:"cc,omitempty"`
}

// EmailResendResponse representa la respuesta al reenviar email
type EmailResendResponse struct {
	Status string `json:"status"`
}

// RetryResponse representa la respuesta al reintentar workflow
type RetryResponse struct {
	Status     string `json:"status"`
	ResumeFrom string `json:"resume_from,omitempty"`
}
