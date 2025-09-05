package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hypernova-labs/dgi-service/internal/models"
	"github.com/sirupsen/logrus"
)

// EmitterRepository maneja las operaciones de base de datos para Emitter
type EmitterRepository struct {
	db     *DB
	logger *logrus.Logger
}

// NewEmitterRepository crea una nueva instancia del repositorio
func NewEmitterRepository(db *DB, logger *logrus.Logger) *EmitterRepository {
	return &EmitterRepository{
		db:     db,
		logger: logger,
	}
}

// GetByID obtiene un emisor por ID
func (r *EmitterRepository) GetByID(id uuid.UUID) (*models.Emitter, error) {
	query := `
		SELECT id, name, company_code, ruc_tipo, ruc_numero, ruc_dv, suc_em,
			   pto_fac_default, iamb, itpemis_default, idoc_default, email, phone,
			   address_line, ubi_code, brand_logo_url, brand_primary_color, brand_footer_html,
			   pac_api_key, pac_subscription_key, is_active, created_at, updated_at
		FROM emitters
		WHERE id = $1 AND is_active = true
	`
	
	var emitter models.Emitter
	err := r.db.QueryRowWithTimeout(query, id).Scan(
		&emitter.ID, &emitter.Name, &emitter.CompanyCode, &emitter.RUCTipo, &emitter.RUCNumero, &emitter.RUCDV, &emitter.SucEm,
		&emitter.PtoFacDefault, &emitter.IAmb, &emitter.ITpEmisDefault, &emitter.IDocDefault, &emitter.Email, &emitter.Phone,
		&emitter.AddressLine, &emitter.UBICode, &emitter.BrandLogoURL, &emitter.BrandPrimaryColor, &emitter.BrandFooterHTML,
		&emitter.PACAPIKey, &emitter.PACSubscriptionKey, &emitter.IsActive, &emitter.CreatedAt, &emitter.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("emitter not found: %s", id)
		}
		return nil, fmt.Errorf("error querying emitter: %w", err)
	}

	return &emitter, nil
}

// GetSeries obtiene una serie específica de un emisor
func (r *EmitterRepository) GetSeries(emitterID uuid.UUID, ptoFacDF string, docKind models.DocumentType) (*models.EmitterSeries, error) {
	query := `
		SELECT id, emitter_id, pto_fac_df, doc_kind, next_number, issued_count,
			   authorized_count, rejected_count, is_active, created_at, updated_at
		FROM emitter_series
		WHERE emitter_id = $1 AND pto_fac_df = $2 AND doc_kind = $3 AND is_active = true
	`
	
	var series models.EmitterSeries
	err := r.db.QueryRowWithTimeout(query, emitterID, ptoFacDF, docKind).Scan(
		&series.ID, &series.EmitterID, &series.PtoFacDF, &series.DocKind, &series.NextNumber, &series.IssuedCount,
		&series.AuthorizedCount, &series.RejectedCount, &series.IsActive, &series.CreatedAt, &series.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("series not found for emitter %s, pto_fac_df %s, doc_kind %s", emitterID, ptoFacDF, docKind)
		}
		return nil, fmt.Errorf("error querying series: %w", err)
	}

	return &series, nil
}

// GetSeriesList obtiene todas las series de un emisor
func (r *EmitterRepository) GetSeriesList(emitterID uuid.UUID) ([]models.EmitterSeries, error) {
	query := `
		SELECT id, emitter_id, pto_fac_df, doc_kind, next_number, issued_count,
			   authorized_count, rejected_count, is_active, created_at, updated_at
		FROM emitter_series
		WHERE emitter_id = $1 AND is_active = true
		ORDER BY pto_fac_df, doc_kind
	`
	
	r.logger.Infof("GetSeriesList: querying for emitterID=%s", emitterID)
	rows, err := r.db.QueryWithTimeout(query, emitterID)
	if err != nil {
		r.logger.Errorf("GetSeriesList: error querying series: %v", err)
		return nil, fmt.Errorf("error querying series: %w", err)
	}
	defer rows.Close()

	var series []models.EmitterSeries
	for rows.Next() {
		var s models.EmitterSeries
		err := rows.Scan(
			&s.ID, &s.EmitterID, &s.PtoFacDF, &s.DocKind, &s.NextNumber, &s.IssuedCount,
			&s.AuthorizedCount, &s.RejectedCount, &s.IsActive, &s.CreatedAt, &s.UpdatedAt,
		)
		if err != nil {
			r.logger.Errorf("GetSeriesList: error scanning series: %v", err)
			return nil, fmt.Errorf("error scanning series: %w", err)
		}
		r.logger.Infof("GetSeriesList: scanned series: %+v", s)
		series = append(series, s)
	}

	r.logger.Infof("GetSeriesList: found %d series", len(series))
	return series, nil
}

// CreateSeries crea una nueva serie para un emisor
func (r *EmitterRepository) CreateSeries(emitterID uuid.UUID, req *models.CreateSeriesRequest) (*models.EmitterSeries, error) {
	r.logger.Infof("CreateSeries: emitterID=%s, req=%+v", emitterID, req)
	
	// Verificar que la serie no exista
	existingSeries, err := r.GetSeries(emitterID, req.PtoFacDF, req.DocKind)
	if err == nil && existingSeries != nil {
		r.logger.Warnf("Series already exists: %+v", existingSeries)
		return nil, fmt.Errorf("series already exists for emitter %s, pto_fac_df %s, doc_kind %s", emitterID, req.PtoFacDF, req.DocKind)
	}
	
	r.logger.Infof("No existing series found, creating new one")

	series := &models.EmitterSeries{
		ID:               uuid.New(),
		EmitterID:        emitterID,
		PtoFacDF:         req.PtoFacDF,
		DocKind:          req.DocKind,
		NextNumber:       1,
		IssuedCount:      0,
		AuthorizedCount:  0,
		RejectedCount:    0,
		IsActive:         true,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}

	query := `
		INSERT INTO emitter_series (
			id, emitter_id, pto_fac_df, doc_kind, next_number, issued_count,
			authorized_count, rejected_count, is_active, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
	`
	
	r.logger.Infof("Executing query: %s", query)
	r.logger.Infof("With params: %s, %s, %s, %s, %d, %d, %d, %d, %t, %s, %s", 
		series.ID, series.EmitterID, series.PtoFacDF, series.DocKind, series.NextNumber, series.IssuedCount,
		series.AuthorizedCount, series.RejectedCount, series.IsActive, series.CreatedAt, series.UpdatedAt)
	
	_, err = r.db.ExecWithTimeout(query,
		series.ID, series.EmitterID, series.PtoFacDF, series.DocKind, series.NextNumber, series.IssuedCount,
		series.AuthorizedCount, series.RejectedCount, series.IsActive, series.CreatedAt, series.UpdatedAt,
	)
	
	if err != nil {
		r.logger.Errorf("Error executing query: %v", err)
		return nil, fmt.Errorf("error creating series: %w", err)
	}
	
	r.logger.Infof("Series created successfully: %+v", series)

	return series, nil
}

// GetDashboard obtiene el dashboard de un emisor
func (r *EmitterRepository) GetDashboard(emitterID uuid.UUID) (*models.DashboardResponse, error) {
	// Obtener estadísticas por mes actual
	currentMonth := time.Now().Format("2006-01")
	
	query := `
		SELECT 
			d_ptofacdf,
			doc_kind,
			COUNT(*) as total_issued,
			COUNT(CASE WHEN status = 'AUTHORIZED' THEN 1 END) as total_authorized,
			COUNT(CASE WHEN status = 'REJECTED' THEN 1 END) as total_rejected
		FROM invoices
		WHERE emitter_id = $1 
		AND DATE_TRUNC('month', created_at) = DATE_TRUNC('month', CURRENT_DATE)
		GROUP BY d_ptofacdf, doc_kind
		ORDER BY d_ptofacdf, doc_kind
	`
	
	rows, err := r.db.QueryWithTimeout(query, emitterID)
	if err != nil {
		return nil, fmt.Errorf("error querying dashboard: %w", err)
	}
	defer rows.Close()

	var series []models.SeriesItem
	var totalIssued, totalAuthorized, totalRejected int

	for rows.Next() {
		var item models.SeriesItem
		var issued, authorized, rejected int
		
		err := rows.Scan(
			&item.PtoFacDF, &item.DocKind, &issued, &authorized, &rejected,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning dashboard item: %w", err)
		}

		item.IssuedCount = issued
		item.AuthorizedCount = authorized
		item.RejectedCount = rejected
		item.LastAssigned = issued // Por simplicidad, usamos el total emitido

		series = append(series, item)
		totalIssued += issued
		totalAuthorized += authorized
		totalRejected += rejected
	}

	response := &models.DashboardResponse{
		Month: currentMonth,
		Series: series,
		Totals: models.DashboardTotals{
			Issued:     totalIssued,
			Authorized: totalAuthorized,
			Rejected:   totalRejected,
		},
	}

	return response, nil
}

// UpdateSeriesCounters actualiza los contadores de una serie
func (r *EmitterRepository) UpdateSeriesCounters(seriesID uuid.UUID, status models.DocumentStatus) error {
	var field string
	switch status {
	case models.DocumentStatusAuthorized:
		field = "authorized_count"
	case models.DocumentStatusRejected:
		field = "rejected_count"
	default:
		field = "issued_count"
	}

	query := fmt.Sprintf(`
		UPDATE emitter_series 
		SET %s = %s + 1, updated_at = $1
		WHERE id = $2
	`, field, field)
	
	result, err := r.db.ExecWithTimeout(query, time.Now(), seriesID)
	if err != nil {
		return fmt.Errorf("error updating series counters: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("series not found: %s", seriesID)
	}

	return nil
}
