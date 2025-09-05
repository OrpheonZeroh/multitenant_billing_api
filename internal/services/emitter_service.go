package services

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hypernova-labs/dgi-service/internal/database"
	"github.com/hypernova-labs/dgi-service/internal/models"
	"github.com/sirupsen/logrus"
)

// EmitterService maneja la lógica de negocio para Emitter
type EmitterService struct {
	emitterRepo *database.EmitterRepository
	apiKeyRepo  *database.APIKeyRepository
	logger      *logrus.Logger
}

// NewEmitterService crea una nueva instancia del servicio
func NewEmitterService(db *database.DB, logger *logrus.Logger) *EmitterService {
	return &EmitterService{
		emitterRepo: database.NewEmitterRepository(db, logger),
		apiKeyRepo:  database.NewAPIKeyRepository(db, logger),
		logger:      logger,
	}
}

// Create crea un nuevo emisor
func (s *EmitterService) Create(req *models.CreateEmitterRequest) (*models.Emitter, error) {
	// Validar RUC
	if err := s.validateRUC(req.RUCTipo, req.RUCNumero, req.RUCDV); err != nil {
		return nil, fmt.Errorf("invalid RUC: %w", err)
	}

	emitter := &models.Emitter{
		ID:                  uuid.New(),
		Name:                req.Name,
		CompanyCode:         req.CompanyCode,
		RUCTipo:             req.RUCTipo,
		RUCNumero:           req.RUCNumero,
		RUCDV:               req.RUCDV,
		SucEm:               req.SucEm,
		PtoFacDefault:       req.PtoFacDefault,
		IAmb:                req.IAmb,
		ITpEmisDefault:      req.ITpEmisDefault,
		IDocDefault:         req.IDocDefault,
		Email:               req.Email,
		Phone:               req.Phone,
		AddressLine:         req.AddressLine,
		UBICode:             req.UBICode,
		BrandLogoURL:        req.BrandLogoURL,
		BrandPrimaryColor:   req.BrandPrimaryColor,
		BrandFooterHTML:     req.BrandFooterHTML,
		PACAPIKey:           req.PACAPIKey,
		PACSubscriptionKey:  req.PACSubscriptionKey,
		IsActive:            true,
		CreatedAt:           time.Now(),
		UpdatedAt:           time.Now(),
	}

	// TODO: Implementar persistencia en base de datos
	// Por ahora solo retornamos el objeto creado
	s.logger.WithFields(logrus.Fields{
		"emitter_id":   emitter.ID,
		"company_code": emitter.CompanyCode,
		"ruc":          fmt.Sprintf("%s-%s-%s-%s", emitter.RUCTipo, emitter.RUCNumero, emitter.RUCDV, emitter.SucEm),
	}).Info("Emitter created successfully")

	return emitter, nil
}

// GetSeries obtiene las series de un emisor con paginación
func (s *EmitterService) GetSeries(emitterID uuid.UUID, page, pageSize int) ([]models.SeriesItem, int, error) {
	// Obtener series del repositorio
	series, err := s.emitterRepo.GetSeriesList(emitterID)
	if err != nil {
		return nil, 0, fmt.Errorf("error getting series: %w", err)
	}

	// Convertir a SeriesItem para la respuesta
	var items []models.SeriesItem
	for _, s := range series {
		item := models.SeriesItem{
			PtoFacDF:        s.PtoFacDF,
			DocKind:         string(s.DocKind),
			LastAssigned:    s.NextNumber - 1,
			IssuedCount:     s.IssuedCount,
			AuthorizedCount: s.AuthorizedCount,
			RejectedCount:   s.RejectedCount,
		}
		items = append(items, item)
	}

	// Calcular total
	total := len(items)

	// Aplicar paginación
	start := (page - 1) * pageSize
	end := start + pageSize

	if start >= total {
		return []models.SeriesItem{}, total, nil
	}

	if end > total {
		end = total
	}

	return items[start:end], total, nil
}

// CreateSeries crea una nueva serie para un emisor
func (s *EmitterService) CreateSeries(emitterID uuid.UUID, req *models.CreateSeriesRequest) (*models.EmitterSeries, error) {
	s.logger.Infof("CreateSeries: emitterID=%s, req=%+v", emitterID, req)
	
	// Validar que el emisor existe
	emitter, err := s.emitterRepo.GetByID(emitterID)
	if err != nil {
		s.logger.Errorf("Error getting emitter: %v", err)
		return nil, fmt.Errorf("error getting emitter: %w", err)
	}
	
	s.logger.Infof("Emitter found: %+v", emitter)

	// Validar que el punto de facturación existe
	if req.PtoFacDF != emitter.PtoFacDefault {
		// TODO: Validar contra lista de puntos de facturación válidos
		s.logger.Warnf("Using non-default punto de facturación: %s (default: %s)", req.PtoFacDF, emitter.PtoFacDefault)
	}

	// Crear serie
	s.logger.Infof("Creating series with emitterID=%s, req=%+v", emitterID, req)
	series, err := s.emitterRepo.CreateSeries(emitterID, req)
	if err != nil {
		s.logger.Errorf("Error creating series: %v", err)
		return nil, fmt.Errorf("error creating series: %w", err)
	}
	
	s.logger.Infof("Series created successfully: %+v", series)

	s.logger.WithFields(logrus.Fields{
		"emitter_id": emitterID,
		"pto_fac_df": req.PtoFacDF,
		"doc_kind":   req.DocKind,
		"series_id":  series.ID,
	}).Info("Series created successfully")

	return series, nil
}

// CreateAPIKey crea una nueva API key para un emisor
func (s *EmitterService) CreateAPIKey(emitterID uuid.UUID, req *models.CreateAPIKeyRequest) (*models.CreateAPIKeyResponse, error) {
	// Validar que el emisor existe
	_, err := s.emitterRepo.GetByID(emitterID)
	if err != nil {
		return nil, fmt.Errorf("error getting emitter: %w", err)
	}

	// Crear API key usando el repositorio
	apiKeyModel, apiKey, err := s.apiKeyRepo.Create(emitterID, req.Name, req.RateLimitPerMin)
	if err != nil {
		return nil, fmt.Errorf("error creating API key: %w", err)
	}

	response := &models.CreateAPIKeyResponse{
		ID:              apiKeyModel.ID,
		Name:            apiKeyModel.Name,
		APIKey:          apiKey, // Solo se retorna una vez
		RateLimitPerMin: apiKeyModel.RateLimitPerMin,
	}

	s.logger.WithFields(logrus.Fields{
		"emitter_id":     emitterID,
		"api_key_name":   req.Name,
		"rate_limit":     req.RateLimitPerMin,
		"api_key_id":     response.ID,
	}).Info("API key created successfully")

	return response, nil
}

// GetDashboard obtiene el dashboard de un emisor
func (s *EmitterService) GetDashboard(emitterID uuid.UUID) (*models.DashboardResponse, error) {
	// Validar que el emisor existe
	emitter, err := s.emitterRepo.GetByID(emitterID)
	if err != nil {
		return nil, fmt.Errorf("error getting emitter: %w", err)
	}

	// Obtener dashboard del repositorio
	response, err := s.emitterRepo.GetDashboard(emitterID)
	if err != nil {
		return nil, fmt.Errorf("error getting dashboard: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"emitter_id": emitterID,
		"company":    emitter.CompanyCode,
		"month":      response.Month,
		"total_issued": response.Totals.Issued,
	}).Info("Dashboard retrieved successfully")

	return response, nil
}

// validateRUC valida el formato del RUC
func (s *EmitterService) validateRUC(rucTipo, rucNumero, rucDV string) error {
	// Validar tipo de RUC
	if rucTipo != "1" && rucTipo != "2" && rucTipo != "3" {
		return fmt.Errorf("invalid RUC type: %s (must be 1, 2, or 3)", rucTipo)
	}

	// Validar número de RUC (debe ser numérico y tener 8 dígitos)
	if len(rucNumero) != 8 {
		return fmt.Errorf("invalid RUC number length: %s (must be 8 digits)", rucNumero)
	}

	// Validar dígito verificador (debe ser numérico y tener 1 dígito)
	if len(rucDV) != 1 {
		return fmt.Errorf("invalid RUC DV length: %s (must be 1 digit)", rucDV)
	}

	// TODO: Implementar validación del algoritmo de dígito verificador
	// Por ahora solo validamos el formato

	return nil
}

// generateAPIKey genera una API key única
func (s *EmitterService) generateAPIKey() string {
	// Generar una clave de 32 caracteres
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	key := make([]byte, 32)
	for i := range key {
		key[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(key)
}
