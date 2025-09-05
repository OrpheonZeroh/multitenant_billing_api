package api

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hypernova-labs/dgi-service/internal/database"
	"github.com/hypernova-labs/dgi-service/internal/models"
	"github.com/hypernova-labs/dgi-service/internal/services"
	"github.com/hypernova-labs/dgi-service/internal/workflows"
	"github.com/sirupsen/logrus"
)

// API maneja todos los endpoints de la API
type API struct {
	invoiceService  *services.InvoiceService
	emitterService  *services.EmitterService
	customerService *services.CustomerService
	productService  *services.ProductService
	apiKeyRepo      *database.APIKeyRepository
	inngestClient   *workflows.InngestClient
	logger          *logrus.Logger
}

// NewAPI crea una nueva instancia de la API
func NewAPI(
	invoiceService *services.InvoiceService,
	emitterService *services.EmitterService,
	customerService *services.CustomerService,
	productService *services.ProductService,
	apiKeyRepo *database.APIKeyRepository,
	inngestClient *workflows.InngestClient,
	logger *logrus.Logger,
) *API {
	return &API{
		invoiceService:  invoiceService,
		emitterService:  emitterService,
		customerService: customerService,
		productService:  productService,
		apiKeyRepo:      apiKeyRepo,
		inngestClient:   inngestClient,
		logger:          logger,
	}
}

// CreateInvoice crea un nuevo documento fiscal
func (api *API) CreateInvoice(c *gin.Context) {
	// Obtener emitter ID del header de autenticación
	emitterID, err := api.getEmitterIDFromAuth(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewUnauthorizedError("Invalid API key"))
		return
	}

	// Parsear request
	var req models.CreateInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.logger.WithError(err).Error("Error binding create invoice request")
		c.JSON(http.StatusBadRequest, models.NewValidationError("Invalid request format", []models.ErrorDetail{
			{Field: "body", Issue: err.Error()},
		}))
		return
	}

	// Obtener idempotency key
	idempotencyKey := c.GetHeader("Idempotency-Key")

	// Crear invoice
	response, err := api.invoiceService.CreateInvoice(emitterID, &req, idempotencyKey)
	if err != nil {
		if strings.Contains(err.Error(), "idempotency key already used") {
			c.JSON(http.StatusConflict, models.NewConflictError("Document with this idempotency key already exists"))
			return
		}
		api.logger.WithError(err).Error("Error creating invoice")
		c.JSON(http.StatusInternalServerError, models.NewInternalError("Error creating document"))
		return
	}

	c.JSON(http.StatusCreated, response)
}

// GetInvoice obtiene un documento por ID
func (api *API) GetInvoice(c *gin.Context) {
	// Obtener emitter ID del header de autenticación
	emitterID, err := api.getEmitterIDFromAuth(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewUnauthorizedError("Invalid API key"))
		return
	}

	// Parsear ID del documento
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewValidationError("Invalid document ID", []models.ErrorDetail{
			{Field: "id", Issue: "Must be a valid UUID"},
		}))
		return
	}

	// Obtener invoice
	response, err := api.invoiceService.GetInvoice(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, models.NewNotFoundError("Document not found"))
			return
		}
		api.logger.WithError(err).Error("Error getting invoice")
		c.JSON(http.StatusInternalServerError, models.NewInternalError("Error retrieving document"))
		return
	}

	// Verificar que el documento pertenece al emisor
	if response.Emitter.RUC != "" && !api.validateEmitterOwnership(emitterID, response.Emitter.RUC) {
		c.JSON(http.StatusForbidden, models.NewForbiddenError("Access denied to this document"))
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetInvoiceFiles obtiene los archivos de un documento
func (api *API) GetInvoiceFiles(c *gin.Context) {
	// Obtener emitter ID del header de autenticación
	_, err := api.getEmitterIDFromAuth(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewUnauthorizedError("Invalid API key"))
		return
	}

	// Parsear ID del documento
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewValidationError("Invalid document ID", []models.ErrorDetail{
			{Field: "id", Issue: "Must be a valid UUID"},
		}))
		return
	}

	// Obtener tipo de archivo desde query parameter
	fileType := c.Query("file_type")
	if fileType == "" {
		// Si no se especifica tipo, retornar información de archivos disponibles
		files, err := api.invoiceService.GetInvoiceFiles(id)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				c.JSON(http.StatusNotFound, models.NewNotFoundError("Document not found"))
				return
			}
			api.logger.WithError(err).Error("Error getting invoice files")
			c.JSON(http.StatusInternalServerError, models.NewInternalError("Error retrieving files"))
			return
		}
		c.JSON(http.StatusOK, files)
		return
	}

	// Validar tipo de archivo
	if fileType != "pdf" && fileType != "xml" {
		c.JSON(http.StatusBadRequest, models.NewValidationError("Invalid file type", []models.ErrorDetail{
			{Field: "file_type", Issue: "Must be 'pdf' or 'xml'"},
		}))
		return
	}

	// Descargar archivo específico
	fileData, fileName, err := api.invoiceService.DownloadInvoiceFile(id, fileType)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, models.NewNotFoundError("File not found"))
			return
		}
		api.logger.WithError(err).Error("Error downloading invoice file")
		c.JSON(http.StatusInternalServerError, models.NewInternalError("Error downloading file"))
		return
	}

	// Configurar headers para descarga
	contentType := "application/octet-stream"
	if fileType == "pdf" {
		contentType = "application/pdf"
	} else if fileType == "xml" {
		contentType = "application/xml"
	}

	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=%s", fileName))
	c.Header("Content-Length", fmt.Sprintf("%d", len(fileData)))

	// Enviar archivo
	c.Data(http.StatusOK, contentType, fileData)
}

// GetPublicInvoiceFile obtiene un archivo público de una factura (sin autenticación)
func (api *API) GetPublicInvoiceFile(c *gin.Context) {
	// Parsear ID del documento
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewValidationError("Invalid document ID", []models.ErrorDetail{
			{Field: "id", Issue: "Must be a valid UUID"},
		}))
		return
	}

	// Obtener tipo de archivo desde query parameter
	fileType := c.Query("file_type")
	if fileType == "" {
		c.JSON(http.StatusBadRequest, models.NewValidationError("File type required", []models.ErrorDetail{
			{Field: "file_type", Issue: "Must specify 'pdf' or 'xml'"},
		}))
		return
	}

	// Validar tipo de archivo
	if fileType != "pdf" && fileType != "xml" {
		c.JSON(http.StatusBadRequest, models.NewValidationError("Invalid file type", []models.ErrorDetail{
			{Field: "file_type", Issue: "Must be 'pdf' or 'xml'"},
		}))
		return
	}

	// Descargar archivo específico
	fileData, fileName, err := api.invoiceService.DownloadInvoiceFile(id, fileType)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, models.NewNotFoundError("File not found"))
			return
		}
		api.logger.WithError(err).Error("Error downloading public invoice file")
		c.JSON(http.StatusInternalServerError, models.NewInternalError("Error downloading file"))
		return
	}

	// Configurar headers para descarga
	contentType := "application/octet-stream"
	if fileType == "pdf" {
		contentType = "application/pdf"
	} else if fileType == "xml" {
		contentType = "application/xml"
	}

	c.Header("Content-Type", contentType)
	c.Header("Content-Disposition", fmt.Sprintf("inline; filename=%s", fileName))
	c.Header("Content-Length", fmt.Sprintf("%d", len(fileData)))

	// Enviar archivo
	c.Data(http.StatusOK, contentType, fileData)
}

// ResendEmail reenvía el email de un documento
func (api *API) ResendEmail(c *gin.Context) {
	// Obtener emitter ID del header de autenticación
	_, err := api.getEmitterIDFromAuth(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewUnauthorizedError("Invalid API key"))
		return
	}

	// Parsear ID del documento
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewValidationError("Invalid document ID", []models.ErrorDetail{
			{Field: "id", Issue: "Must be a valid UUID"},
		}))
		return
	}

	// Parsear request
	var req models.EmailResendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.logger.WithError(err).Error("Error binding email resend request")
		c.JSON(http.StatusBadRequest, models.NewValidationError("Invalid request format", []models.ErrorDetail{
			{Field: "body", Issue: err.Error()},
		}))
		return
	}

	// Reenviar email
	response, err := api.invoiceService.ResendEmail(id, &req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, models.NewNotFoundError("Document not found"))
			return
		}
		api.logger.WithError(err).Error("Error resending email")
		c.JSON(http.StatusInternalServerError, models.NewInternalError("Error resending email"))
		return
	}

	c.JSON(http.StatusOK, response)
}

// RetryWorkflow reintenta el workflow de un documento
func (api *API) RetryWorkflow(c *gin.Context) {
	// Obtener emitter ID del header de autenticación
	_, err := api.getEmitterIDFromAuth(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewUnauthorizedError("Invalid API key"))
		return
	}

	// Parsear ID del documento
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewValidationError("Invalid document ID", []models.ErrorDetail{
			{Field: "id", Issue: "Must be a valid UUID"},
		}))
		return
	}

	// Reintentar workflow
	response, err := api.invoiceService.RetryWorkflow(id)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, models.NewNotFoundError("Document not found"))
			return
		}
		api.logger.WithError(err).Error("Error retrying workflow")
		c.JSON(http.StatusInternalServerError, models.NewInternalError("Error retrying workflow"))
		return
	}

	c.JSON(http.StatusOK, response)
}

// GetSeries obtiene las series de documentos de un emisor
func (api *API) GetSeries(c *gin.Context) {
	// Obtener emitter ID del header de autenticación
	emitterID, err := api.getEmitterIDFromAuth(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewUnauthorizedError("Invalid API key"))
		return
	}

	// Parsear parámetros de paginación
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	// Obtener series
	series, total, err := api.emitterService.GetSeries(emitterID, page, pageSize)
	if err != nil {
		api.logger.WithError(err).Error("Error getting series")
		c.JSON(http.StatusInternalServerError, models.NewInternalError("Error retrieving series"))
		return
	}

	response := models.SeriesResponse{
		Items:    series,
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	}

	c.JSON(http.StatusOK, response)
}

// CreateCustomer crea un nuevo cliente (endpoint admin)
func (api *API) CreateCustomer(c *gin.Context) {
	// Obtener emitter ID del header de autenticación
	emitterID, err := api.getEmitterIDFromAuth(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewUnauthorizedError("Invalid API key"))
		return
	}

	// Parsear request
	var req models.CreateCustomerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.logger.WithError(err).Error("Error binding create customer request")
		c.JSON(http.StatusBadRequest, models.NewValidationError("Invalid request format", []models.ErrorDetail{
			{Field: "body", Issue: err.Error()},
		}))
		return
	}

	// Crear cliente
	customer, err := api.customerService.Create(&req, emitterID)
	if err != nil {
		api.logger.WithError(err).Error("Error creating customer")
		c.JSON(http.StatusInternalServerError, models.NewInternalError("Error creating customer"))
		return
	}

	response := models.CustomerResponse{
		ID: customer.ID.String(),
	}

	c.JSON(http.StatusCreated, response)
}

// CreateProduct crea un nuevo producto (endpoint admin)
func (api *API) CreateProduct(c *gin.Context) {
	// Obtener emitter ID del header de autenticación
	emitterID, err := api.getEmitterIDFromAuth(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewUnauthorizedError("Invalid API key"))
		return
	}

	// Parsear request
	var req models.CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.logger.WithError(err).Error("Error binding create product request")
		c.JSON(http.StatusBadRequest, models.NewValidationError("Invalid request format", []models.ErrorDetail{
			{Field: "body", Issue: err.Error()},
		}))
		return
	}

	// Crear producto
	product, err := api.productService.Create(&req, emitterID)
	if err != nil {
		api.logger.WithError(err).Error("Error creating product")
		c.JSON(http.StatusInternalServerError, models.NewInternalError("Error creating product"))
		return
	}

	response := models.ProductResponse{
		ID: product.ID.String(),
	}

	c.JSON(http.StatusCreated, response)
}

// CreateEmitter crea un nuevo emisor (endpoint admin)
func (api *API) CreateEmitter(c *gin.Context) {
	// TODO: Implementar autenticación admin
	// Por ahora permitimos crear emisores sin autenticación especial

	// Parsear request
	var req models.CreateEmitterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.logger.WithError(err).Error("Error binding create emitter request")
		c.JSON(http.StatusBadRequest, models.NewValidationError("Invalid request format", []models.ErrorDetail{
			{Field: "body", Issue: err.Error()},
		}))
		return
	}

	// Crear emisor
	emitter, err := api.emitterService.Create(&req)
	if err != nil {
		api.logger.WithError(err).Error("Error creating emitter")
		c.JSON(http.StatusInternalServerError, models.NewInternalError("Error creating emitter"))
		return
	}

	response := models.EmitterResponse{
		ID: emitter.ID.String(),
	}

	c.JSON(http.StatusCreated, response)
}

// CreateSeries crea una nueva serie para un emisor (endpoint admin)
func (api *API) CreateSeries(c *gin.Context) {
	// Obtener emitter ID del header de autenticación
	emitterID, err := api.getEmitterIDFromAuth(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewUnauthorizedError("Invalid API key"))
		return
	}

	// Parsear request
	var req models.CreateSeriesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.logger.WithError(err).Error("Error binding create series request")
		c.JSON(http.StatusBadRequest, models.NewValidationError("Invalid request format", []models.ErrorDetail{
			{Field: "body", Issue: err.Error()},
		}))
		return
	}

	// Crear serie
	series, err := api.emitterService.CreateSeries(emitterID, &req)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			c.JSON(http.StatusConflict, models.NewConflictError("Series already exists"))
			return
		}
		api.logger.WithError(err).Error("Error creating series")
		c.JSON(http.StatusInternalServerError, models.NewInternalError("Error creating series"))
		return
	}

	response := models.SeriesResponse{
		Items: []models.SeriesItem{
			{
				PtoFacDF:        series.PtoFacDF,
				DocKind:         string(series.DocKind),
				LastAssigned:    series.NextNumber - 1,
				IssuedCount:     series.IssuedCount,
				AuthorizedCount: series.AuthorizedCount,
				RejectedCount:   series.RejectedCount,
			},
		},
		Page:     1,
		PageSize: 1,
		Total:    1,
	}

	c.JSON(http.StatusCreated, response)
}

// CreateAPIKey crea una nueva API key para un emisor (endpoint admin)
func (api *API) CreateAPIKey(c *gin.Context) {
	// Obtener emitter ID del header de autenticación
	emitterID, err := api.getEmitterIDFromAuth(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewUnauthorizedError("Invalid API key"))
		return
	}

	// Parsear request
	var req models.CreateAPIKeyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		api.logger.WithError(err).Error("Error binding create API key request")
		c.JSON(http.StatusBadRequest, models.NewValidationError("Invalid request format", []models.ErrorDetail{
			{Field: "body", Issue: err.Error()},
		}))
		return
	}

	// Crear API key
	response, err := api.emitterService.CreateAPIKey(emitterID, &req)
	if err != nil {
		api.logger.WithError(err).Error("Error creating API key")
		c.JSON(http.StatusInternalServerError, models.NewInternalError("Error creating API key"))
		return
	}

	c.JSON(http.StatusCreated, response)
}

// GetDashboard obtiene el dashboard de un emisor (endpoint admin)
func (api *API) GetDashboard(c *gin.Context) {
	// Obtener emitter ID del header de autenticación
	emitterID, err := api.getEmitterIDFromAuth(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.NewUnauthorizedError("Invalid API key"))
		return
	}

	// Obtener dashboard
	response, err := api.emitterService.GetDashboard(emitterID)
	if err != nil {
		api.logger.WithError(err).Error("Error getting dashboard")
		c.JSON(http.StatusInternalServerError, models.NewInternalError("Error retrieving dashboard"))
		return
	}

	c.JSON(http.StatusOK, response)
}

// getEmitterIDFromAuth extrae el emitter ID del header de autenticación
func (api *API) getEmitterIDFromAuth(c *gin.Context) (uuid.UUID, error) {
	apiKey := c.GetHeader("X-API-Key")
	// Debug: Log la API key recibida
	api.logger.Infof("API Key recibida: %s", apiKey)
	if apiKey == "" {
		return uuid.Nil, models.NewAPIError(models.NewUnauthorizedError("API key required"))
	}

	// Validar API key usando el repositorio
	apiKeyModel, err := api.apiKeyRepo.GetByHash(api.apiKeyRepo.HashAPIKey(apiKey))
	if err != nil {
		return uuid.Nil, models.NewAPIError(models.NewUnauthorizedError("Invalid API key"))
	}

	// Actualizar último uso
	if err := api.apiKeyRepo.UpdateLastUsed(apiKeyModel.ID); err != nil {
		api.logger.Warnf("Error updating API key last used: %v", err)
	}

	return apiKeyModel.EmitterID, nil
}

// AdminAuthMiddleware retorna middleware para autenticación de admin
func (api *API) AdminAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Obtener emitter ID del header de autenticación
		emitterID, err := api.getEmitterIDFromAuth(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, models.NewUnauthorizedError("Invalid API key"))
			c.Abort()
			return
		}

		// TODO: Implementar validación de permisos de admin
		// Por ahora solo validamos que la API key sea válida
		
		// Agregar emitter ID al contexto para uso posterior
		c.Set("emitter_id", emitterID)
		c.Next()
	}
}

// validateEmitterOwnership valida que un documento pertenece al emisor
func (api *API) validateEmitterOwnership(emitterID uuid.UUID, ruc string) bool {
	// TODO: Implementar validación real
	// Por ahora retornamos true para desarrollo
	return true
}
