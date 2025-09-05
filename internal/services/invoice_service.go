package services

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/hypernova-labs/dgi-service/internal/database"
	"github.com/hypernova-labs/dgi-service/internal/email"
	"github.com/hypernova-labs/dgi-service/internal/models"
	"github.com/hypernova-labs/dgi-service/internal/workflows"
	"github.com/sirupsen/logrus"
)

// InvoiceService maneja la lógica de negocio para documentos fiscales
type InvoiceService struct {
	invoiceRepo        *database.InvoiceRepository
	emitterRepo        *database.EmitterRepository
	customerRepo       *database.CustomerRepository
	productRepo        *database.ProductRepository
	invoiceFilesRepo   *database.InvoiceFilesRepository
	inngestClient      *workflows.InngestClient
	resendService      *email.ResendService
	documentGenerator  *DocumentGenerator
	storageService     *HybridStorageService
	logger             *logrus.Logger
}

// NewInvoiceService crea una nueva instancia del servicio
func NewInvoiceService(db *database.DB, inngestClient *workflows.InngestClient, resendService *email.ResendService, supabaseClient *database.SupabaseClient, logger *logrus.Logger) *InvoiceService {
	// Inicializar repositorios
	invoiceRepo := database.NewInvoiceRepository(db, logger)
	emitterRepo := database.NewEmitterRepository(db, logger)
	customerRepo := database.NewCustomerRepository(db, logger)
	productRepo := database.NewProductRepository(db, logger)
	invoiceFilesRepo := database.NewInvoiceFilesRepository(db, logger)

	// Inicializar servicios
	documentGenerator := NewDocumentGenerator(logger)

	// Inicializar servicio de storage híbrido si Supabase está disponible
	var storageService *HybridStorageService
	if supabaseClient != nil {
		storageService = NewHybridStorageService(supabaseClient, invoiceFilesRepo, logger)
		// Inicializar bucket de Supabase
		if err := storageService.InitializeBucket(context.Background()); err != nil {
			logger.Warnf("Could not initialize Supabase bucket: %v", err)
		}
	}

	return &InvoiceService{
		invoiceRepo:       invoiceRepo,
		emitterRepo:       emitterRepo,
		customerRepo:      customerRepo,
		productRepo:       productRepo,
		invoiceFilesRepo:  invoiceFilesRepo,
		inngestClient:     inngestClient,
		resendService:     resendService,
		documentGenerator: documentGenerator,
		storageService:    storageService,
		logger:            logger,
	}
}

// CreateInvoice crea un nuevo documento fiscal
func (s *InvoiceService) CreateInvoice(emitterID uuid.UUID, req *models.CreateInvoiceRequest, idempotencyKey string) (*models.InvoiceResponse, error) {
	// Verificar idempotencia si se proporciona clave
	if idempotencyKey != "" {
		existingInvoice, err := s.invoiceRepo.GetByIdempotencyKey(idempotencyKey)
		if err != nil {
			return nil, fmt.Errorf("error checking idempotency: %w", err)
		}
		if existingInvoice != nil {
			return nil, fmt.Errorf("idempotency key already used")
		}
	}

	// Obtener emisor
	emitter, err := s.emitterRepo.GetByID(emitterID)
	if err != nil {
		return nil, fmt.Errorf("error getting emitter: %w", err)
	}

	// Obtener o crear cliente
	customer, err := s.getOrCreateCustomer(req.Customer, emitterID)
	if err != nil {
		return nil, fmt.Errorf("error getting/creating customer: %w", err)
	}

	// Obtener serie para el documento
	var ptoFacDF string
	if req.Overrides != nil && req.Overrides.PtoFacDF != "" {
		ptoFacDF = req.Overrides.PtoFacDF
	} else {
		// Usar el punto de facturación por defecto del emisor (ya obtenido arriba)
		ptoFacDF = emitter.PtoFacDefault
	}
	
	series, err := s.emitterRepo.GetSeries(emitterID, ptoFacDF, req.DocumentType)
	if err != nil {
		return nil, fmt.Errorf("error getting series: %w", err)
	}

	// Obtener siguiente número de documento
	documentNumber, err := s.invoiceRepo.GetNextDocumentNumber(emitterID, series.PtoFacDF, req.DocumentType)
	if err != nil {
		return nil, fmt.Errorf("error getting next document number: %w", err)
	}

	// Calcular totales
	subtotal, itbmsAmount, totalAmount, err := s.calculateTotals(req.Items)
	if err != nil {
		return nil, fmt.Errorf("error calculating totals: %w", err)
	}

	// Validar que el total coincida con el payment amount (con tolerancia de 0.01)
	if math.Abs(totalAmount - req.Payment.Amount) > 0.01 {
		return nil, fmt.Errorf("calculated total (%.2f) does not match payment amount (%.2f)", totalAmount, req.Payment.Amount)
	}

	// Crear invoice
	invoice := &models.Invoice{
		ID:              uuid.New(),
		EmitterID:       emitterID,
		SeriesID:        series.ID,
		CustomerID:      customer.ID,
		DocumentType:    req.DocumentType,
		DocumentNumber:  documentNumber,
		PtoFacDF:        series.PtoFacDF,
		Status:          models.DocumentStatusReceived,
		EmailStatus:     models.EmailStatusPending,
		ReferenceCUFE:   s.getReferenceValue(req.Reference, "cufe"),
		ReferenceNumber: s.getReferenceValue(req.Reference, "nrodf"),
		ReferencePtoFac: s.getReferenceValue(req.Reference, "pto_fac_df"),
		IAmb:            emitter.IAmb,
		ITpEmis:         s.getOverrideValue(s.getOverrideField(req.Overrides, "ITpEmis"), emitter.ITpEmisDefault),
		IDoc:            s.getOverrideValue(s.getOverrideField(req.Overrides, "IDoc"), emitter.IDocDefault),
		Subtotal:        subtotal,
		ITBMSAmount:     itbmsAmount,
		TotalAmount:     totalAmount,
		IdempotencyKey:  func() *string { if idempotencyKey == "" { return nil } else { return &idempotencyKey } }(),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	// Crear items
	items := make([]models.InvoiceItem, len(req.Items))
	for i, itemReq := range req.Items {
		// Obtener o crear producto si se proporciona SKU o ProductID
		var product *models.Product
		if itemReq.ProductID != nil {
			// Buscar producto por ID
			productID, err := uuid.Parse(*itemReq.ProductID)
			if err != nil {
				return nil, fmt.Errorf("invalid product_id format: %s", *itemReq.ProductID)
			}
			product, err = s.productRepo.GetByID(productID)
			if err != nil {
				return nil, fmt.Errorf("product with ID %s not found: %w", *itemReq.ProductID, err)
			}
		} else if itemReq.SKU != nil {
			// Buscar producto por SKU (mantener compatibilidad)
			product, err = s.productRepo.GetBySKU(emitterID, *itemReq.SKU)
			if err != nil {
				s.logger.Warnf("Product with SKU %s not found, using request data", *itemReq.SKU)
			}
		}

		// Usar descripción del producto si está disponible, sino usar la del request
		description := itemReq.Description
		if product != nil && product.Description != "" {
			description = product.Description
		}

		// Calcular total de línea
		lineTotal := itemReq.Quantity * itemReq.UnitPrice

		items[i] = models.InvoiceItem{
			ID:          uuid.New(),
			InvoiceID:   invoice.ID,
			LineNo:      i + 1,
			SKU:         itemReq.SKU,
			Description: description,
			Quantity:    itemReq.Quantity,
			UnitPrice:   itemReq.UnitPrice,
			ITBMSRate:   itemReq.TaxRate,
			CPBSAbr:     s.getProductValue(product, "cpbs_abr", nil),
			CPBSCmp:     s.getProductValue(product, "cpbs_cmp", nil),
			LineTotal:   lineTotal,
			CreatedAt:   time.Now(),
		}
	}

	// Persistir en base de datos
	if err := s.invoiceRepo.Create(invoice, items); err != nil {
		return nil, fmt.Errorf("error creating invoice: %w", err)
	}

	// Construir respuesta
	response := &models.InvoiceResponse{
		ID:           invoice.ID,
		Status:       invoice.Status,
		DocumentType: invoice.DocumentType,
		Emitter: models.EmitterInfo{
			RUC:    fmt.Sprintf("%s-%s-%s-%s", emitter.RUCTipo, emitter.RUCNumero, emitter.RUCDV, emitter.SucEm),
			PtoFac: invoice.PtoFacDF,
			Number: invoice.DocumentNumber,
		},
		Totals: models.Totals{
			Net:   invoice.Subtotal,
			ITBMS: invoice.ITBMSAmount,
			Total: invoice.TotalAmount,
		},
		Links: models.Links{
			Self:  fmt.Sprintf("/v1/invoices/%s", invoice.ID),
			Files: fmt.Sprintf("/v1/invoices/%s/files", invoice.ID),
		},
	}

	s.logger.WithFields(logrus.Fields{
		"invoice_id": invoice.ID,
		"emitter_id": emitterID,
		"document_number": invoice.DocumentNumber,
		"total_amount": invoice.TotalAmount,
	}).Info("Invoice created successfully")

	// ENVÍO DIRECTO DE EMAIL (para testing sin workflow)
	if s.resendService != nil {
		go func() {
			s.logger.WithField("invoice_id", invoice.ID).Info("Sending email directly via Resend (testing mode)")
			
			// Obtener datos del cliente y emisor para el email
			customer, err := s.customerRepo.GetByID(invoice.CustomerID)
			if err != nil {
				s.logger.WithField("invoice_id", invoice.ID).Errorf("Failed to get customer for direct email: %v", err)
				return
			}
			emitter, err := s.emitterRepo.GetByID(invoice.EmitterID)
			if err != nil {
				s.logger.WithField("invoice_id", invoice.ID).Errorf("Failed to get emitter for direct email: %v", err)
				return
			}

			// Enviar email directamente usando Resend
			err = s.resendService.SendInvoiceEmail(invoice, customer, emitter)
			if err != nil {
				s.logger.WithFields(logrus.Fields{
					"invoice_id": invoice.ID,
					"error": err,
				}).Error("Failed to send email directly via Resend")
				// Opcional: Actualizar estado de email a fallido en DB
				if updateErr := s.invoiceRepo.UpdateEmailStatus(invoice.ID, models.EmailStatusFailed); updateErr != nil {
					s.logger.WithField("invoice_id", invoice.ID).Errorf("Failed to update email status: %v", updateErr)
				}
				return
			}
			s.logger.WithFields(logrus.Fields{
				"invoice_id": invoice.ID,
				"customer_email": customer.Email,
				"document_number": invoice.DocumentNumber,
			}).Info("Email sent successfully via Resend (direct mode)")
			// Opcional: Actualizar estado de email a enviado en DB
			if err := s.invoiceRepo.UpdateEmailStatus(invoice.ID, models.EmailStatusSent); err != nil {
				s.logger.WithField("invoice_id", invoice.ID).Errorf("Failed to update email status: %v", err)
			}
		}()
	} else {
		s.logger.Warn("Resend service not available - email not sent")
	}

	return response, nil
}

// GetInvoice obtiene un invoice por ID
func (s *InvoiceService) GetInvoice(id uuid.UUID) (*models.InvoiceStatusResponse, error) {
	invoice, err := s.invoiceRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// Construir respuesta
	response := &models.InvoiceStatusResponse{
		ID:           invoice.ID,
		Status:       invoice.Status,
		EmailStatus:  invoice.EmailStatus,
		DocumentType: invoice.DocumentType,
		CUFE:         invoice.CUFE,
		URLCUFE:      invoice.URLCUFE,
		Emitter: models.EmitterInfo{
			PtoFac: invoice.PtoFacDF,
			Number: invoice.DocumentNumber,
		},
		Totals: models.Totals{
			Net:   invoice.Subtotal,
			ITBMS: invoice.ITBMSAmount,
			Total: invoice.TotalAmount,
		},
		CreatedAt: invoice.CreatedAt,
		Links: models.Links{
			Files: fmt.Sprintf("/v1/invoices/%s/files", id),
		},
	}

	return response, nil
}

// GetInvoiceFiles obtiene los archivos de un invoice
func (s *InvoiceService) GetInvoiceFiles(id uuid.UUID) (*models.InvoiceFilesResponse, error) {
	// Verificar si ya existen archivos
	existingFiles, err := s.invoiceFilesRepo.GetByInvoiceID(id)
	if err == nil && existingFiles != nil {
		// Archivos ya existen, retornar respuesta con URLs
		return &models.InvoiceFilesResponse{
			XMLFE:       stringPtr(fmt.Sprintf("/v1/invoices/%s/files/xml", id)),
			XMLProtocolo: stringPtr(fmt.Sprintf("/v1/invoices/%s/files/xml", id)),
			CAFEPDFURL:   stringPtr(fmt.Sprintf("/v1/invoices/%s/files/pdf", id)),
			Disposition:  "inline",
		}, nil
	}

	// Archivos no existen, generarlos
	invoice, err := s.invoiceRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("error getting invoice: %w", err)
	}

	customer, err := s.customerRepo.GetByID(invoice.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("error getting customer: %w", err)
	}

	emitter, err := s.emitterRepo.GetByID(invoice.EmitterID)
	if err != nil {
		return nil, fmt.Errorf("error getting emitter: %w", err)
	}

	items, err := s.invoiceRepo.GetItemsByInvoiceID(id)
	if err != nil {
		return nil, fmt.Errorf("error getting invoice items: %w", err)
	}

	// Generar archivos
	files, err := s.documentGenerator.GenerateInvoiceFiles(invoice, customer, emitter, items)
	if err != nil {
		return nil, fmt.Errorf("error generating invoice files: %w", err)
	}

	// Si tenemos Supabase disponible, subir archivos al storage
	if s.storageService != nil {
		ctx := context.Background()
		
		// Usar el servicio híbrido para almacenar archivos
		storageResponse, err := s.storageService.StoreInvoiceFiles(ctx, id, files.PDFData, files.XMLData)
		if err != nil {
			s.logger.WithError(err).Warn("Failed to store files in Supabase, falling back to local storage")
		} else {
			// Actualizar URLs en el modelo
			if storageResponse.CAFEPDFURL != nil {
				files.PDFURL = storageResponse.CAFEPDFURL
			}
			if storageResponse.XMLFE != nil {
				files.XMLURL = storageResponse.XMLFE
			}
			s.logger.WithFields(logrus.Fields{
				"pdf_url": files.PDFURL,
				"xml_url": files.XMLURL,
			}).Info("Files stored in Supabase successfully")
		}
	}

	// Guardar archivos en BD (UPSERT para evitar duplicados)
	if err := s.invoiceFilesRepo.CreateOrUpdate(files); err != nil {
		return nil, fmt.Errorf("error saving invoice files: %w", err)
	}

	// Retornar respuesta con URLs (priorizar Supabase si está disponible)
	response := &models.InvoiceFilesResponse{
		XMLFE:       stringPtr(fmt.Sprintf("/v1/invoices/%s/files/xml", id)),
		XMLProtocolo: stringPtr(fmt.Sprintf("/v1/invoices/%s/files/xml", id)),
		CAFEPDFURL:   stringPtr(fmt.Sprintf("/v1/invoices/%s/files/pdf", id)),
		Disposition:  "inline",
	}

	// Si tenemos URLs de Supabase, usarlas en lugar de las locales
	if files.PDFURL != nil {
		response.CAFEPDFURL = files.PDFURL
	}
	if files.XMLURL != nil {
		response.XMLFE = files.XMLURL
		response.XMLProtocolo = files.XMLURL
	}

	s.logger.WithFields(logrus.Fields{
		"invoice_id": id,
		"pdf_size":   files.PDFSize,
		"xml_size":   files.XMLSize,
	}).Info("Invoice files generated successfully")

	return response, nil
}

// DownloadInvoiceFile descarga un archivo específico de la factura
func (s *InvoiceService) DownloadInvoiceFile(id uuid.UUID, fileType string) ([]byte, string, error) {
	// Obtener archivos de la factura
	files, err := s.invoiceFilesRepo.GetByInvoiceID(id)
	if err != nil {
		return nil, "", fmt.Errorf("error getting invoice files: %w", err)
	}

	var fileData []byte
	var fileName string

	switch fileType {
	case "pdf":
		// Si tenemos datos locales, usarlos
		if len(files.PDFData) > 0 {
			fileData = files.PDFData
		} else if files.PDFURL != nil {
			// Si no hay datos locales pero hay URL de Supabase, descargar desde ahí
			if s.storageService != nil {
				downloadedData, err := s.storageService.DownloadFile(context.Background(), *files.PDFURL)
				if err != nil {
					return nil, "", fmt.Errorf("error downloading PDF from Supabase: %w", err)
				}
				fileData = downloadedData
			} else {
				return nil, "", fmt.Errorf("storage service not available")
			}
		} else {
			return nil, "", fmt.Errorf("PDF file not found for invoice %s", id)
		}
		fileName = fmt.Sprintf("factura_%s.pdf", id.String())
	case "xml":
		// Si tenemos datos locales, usarlos
		if len(files.XMLData) > 0 {
			fileData = files.XMLData
		} else if files.XMLURL != nil {
			// Si no hay datos locales pero hay URL de Supabase, descargar desde ahí
			if s.storageService != nil {
				downloadedData, err := s.storageService.DownloadFile(context.Background(), *files.XMLURL)
				if err != nil {
					return nil, "", fmt.Errorf("error downloading XML from Supabase: %w", err)
				}
				fileData = downloadedData
			} else {
				return nil, "", fmt.Errorf("storage service not available")
			}
		} else {
			return nil, "", fmt.Errorf("XML file not found for invoice %s", id)
		}
		fileName = fmt.Sprintf("factura_%s.xml", id.String())
	default:
		return nil, "", fmt.Errorf("invalid file type: %s", fileType)
	}

	if len(fileData) == 0 {
		return nil, "", fmt.Errorf("file %s not found for invoice %s", fileType, id)
	}

	return fileData, fileName, nil
}

// ResendEmail reenvía el email de un invoice
func (s *InvoiceService) ResendEmail(id uuid.UUID, req *models.EmailResendRequest) (*models.EmailResendResponse, error) {
	// Verificar que el invoice existe
	_, err := s.invoiceRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// TODO: Implementar lógica de envío de email
	// Por ahora solo actualizamos el estado
	if err := s.invoiceRepo.UpdateEmailStatus(id, models.EmailStatusPending); err != nil {
		return nil, fmt.Errorf("error updating email status: %w", err)
	}

	response := &models.EmailResendResponse{
		Status: "ENQUEUED",
	}

	s.logger.WithFields(logrus.Fields{
		"invoice_id": id,
		"email_status": "ENQUEUED",
	}).Info("Email resend requested")

	return response, nil
}

// RetryWorkflow reintenta el workflow de un invoice
func (s *InvoiceService) RetryWorkflow(id uuid.UUID) (*models.RetryResponse, error) {
	// Verificar que el invoice existe
	_, err := s.invoiceRepo.GetByID(id)
	if err != nil {
		return nil, err
	}

	// TODO: Implementar lógica de reintento de workflow
	// Por ahora solo retornamos éxito
	response := &models.RetryResponse{
		Status:     "ENQUEUED",
		ResumeFrom: "send_to_pac", // Por defecto desde el paso del PAC
	}

	s.logger.WithFields(logrus.Fields{
		"invoice_id": id,
		"resume_from": "send_to_pac",
	}).Info("Workflow retry requested")

	return response, nil
}

// getOrCreateCustomer obtiene o crea un cliente
func (s *InvoiceService) getOrCreateCustomer(req models.CustomerRequest, emitterID uuid.UUID) (*models.Customer, error) {
	// Intentar obtener cliente existente
	customer, err := s.customerRepo.GetByEmail(emitterID, req.Email)
	if err == nil {
		return customer, nil
	}

	// Crear nuevo cliente
	customerReq := &models.CreateCustomerRequest{
		Name:    req.Name,
		Email:   req.Email,
		Phone:   req.Phone,
		Address: req.Address,
		UBICode: req.UBICode,
	}

	customer, err = s.customerRepo.Create(customerReq, emitterID)
	if err != nil {
		return nil, fmt.Errorf("error creating customer: %w", err)
	}

	return customer, nil
}

// calculateTotals calcula los totales del documento
func (s *InvoiceService) calculateTotals(items []models.ItemRequest) (subtotal, itbmsAmount, totalAmount float64, err error) {
	s.logger.Infof("calculateTotals: processing %d items", len(items))
	
	for i, item := range items {
		s.logger.Infof("Item %d: Quantity=%.2f, UnitPrice=%.2f, TaxRate=%s", i+1, item.Quantity, item.UnitPrice, item.TaxRate)
		
		lineTotal := item.Quantity * item.UnitPrice
		subtotal += lineTotal
		s.logger.Infof("Item %d: LineTotal=%.2f, Subtotal=%.2f", i+1, lineTotal, subtotal)

		// Calcular ITBMS según la tasa
		var itbmsRate float64
		switch item.TaxRate {
		case "00":
			itbmsRate = 0.0
		case "01":
			itbmsRate = 0.07
		case "02":
			itbmsRate = 0.10
		case "03":
			itbmsRate = 0.15
		default:
			return 0, 0, 0, fmt.Errorf("invalid tax rate: %s", item.TaxRate)
		}

		lineITBMS := lineTotal * itbmsRate
		itbmsAmount += lineITBMS
		s.logger.Infof("Item %d: ITBMSRate=%.2f, LineITBMS=%.2f, TotalITBMS=%.2f", i+1, itbmsRate, lineITBMS, itbmsAmount)
	}

	totalAmount = subtotal + itbmsAmount
	s.logger.Infof("Final totals: Subtotal=%.2f, ITBMS=%.2f, Total=%.2f", subtotal, itbmsAmount, totalAmount)
	return subtotal, itbmsAmount, totalAmount, nil
}

// getReferenceValue obtiene un valor de referencia o nil
func (s *InvoiceService) getReferenceValue(ref *models.Reference, field string) *string {
	if ref == nil {
		return nil
	}

	switch field {
	case "cufe":
		return &ref.CUFE
	case "nrodf":
		return &ref.Number
	case "pto_fac_df":
		return &ref.PtoFac
	default:
		return nil
	}
}

// getOverrideValue obtiene un valor de override o el valor por defecto
func (s *InvoiceService) getOverrideValue(override, defaultValue string) string {
	if override != "" {
		return override
	}
	return defaultValue
}

// getOverrideField obtiene un campo de override de forma segura
func (s *InvoiceService) getOverrideField(overrides *models.Overrides, field string) string {
	if overrides == nil {
		return ""
	}
	
	switch field {
	case "ITpEmis":
		return overrides.ITpEmis
	case "IDoc":
		return overrides.IDoc
	case "PtoFacDF":
		return overrides.PtoFacDF
	default:
		return ""
	}
}

// getProductValue obtiene un valor del producto o el valor del request
func (s *InvoiceService) getProductValue(product *models.Product, field string, requestValue *string) *string {
	if product != nil {
		switch field {
		case "cpbs_abr":
			return product.CPBSAbr
		case "cpbs_cmp":
			return product.CPBSCmp
		}
	}
	return requestValue
}

// stringPtr convierte un string a *string
func stringPtr(s string) *string {
	return &s
}
