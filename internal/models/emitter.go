package models

import (
	"time"

	"github.com/google/uuid"
)

// Emitter representa una empresa que emite documentos fiscales
type Emitter struct {
	ID                  uuid.UUID `json:"id" db:"id"`
	Name                string    `json:"name" db:"name"`
	CompanyCode         string    `json:"company_code" db:"company_code"`
	RUCTipo             string    `json:"ruc_tipo" db:"ruc_tipo"`
	RUCNumero           string    `json:"ruc_numero" db:"ruc_numero"`
	RUCDV               string    `json:"ruc_dv" db:"ruc_dv"`
	SucEm               string    `json:"suc_em" db:"suc_em"`
	PtoFacDefault       string    `json:"pto_fac_default" db:"pto_fac_default"`
	IAmb                int       `json:"i_amb" db:"iamb"`
	ITpEmisDefault      string    `json:"itpemis_default" db:"itpemis_default"`
	IDocDefault         string    `json:"idoc_default" db:"idoc_default"`
	Email               string    `json:"email" db:"email"`
	Phone               *string   `json:"phone,omitempty" db:"phone"`
	AddressLine         *string   `json:"address_line,omitempty" db:"address_line"`
	UBICode             *string   `json:"ubi_code,omitempty" db:"ubi_code"`
	BrandLogoURL        *string   `json:"brand_logo_url,omitempty" db:"brand_logo_url"`
	BrandPrimaryColor   *string   `json:"brand_primary_color,omitempty" db:"brand_primary_color"`
	BrandFooterHTML     *string   `json:"brand_footer_html,omitempty" db:"brand_footer_html"`
	PACAPIKey           string    `json:"pac_api_key" db:"pac_api_key"`
	PACSubscriptionKey  string    `json:"pac_subscription_key" db:"pac_subscription_key"`
	IsActive            bool      `json:"is_active" db:"is_active"`
	CreatedAt           time.Time `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time `json:"updated_at" db:"updated_at"`
}

// EmitterSeries representa una serie de documentos por emisor
type EmitterSeries struct {
	ID               uuid.UUID   `json:"id" db:"id"`
	EmitterID        uuid.UUID   `json:"emitter_id" db:"emitter_id"`
	PtoFacDF         string      `json:"pto_fac_df" db:"pto_fac_df"`
	DocKind          DocumentType `json:"doc_kind" db:"doc_kind"`
	NextNumber       int         `json:"next_number" db:"next_number"`
	IssuedCount      int         `json:"issued_count" db:"issued_count"`
	AuthorizedCount  int         `json:"authorized_count" db:"authorized_count"`
	RejectedCount    int         `json:"rejected_count" db:"rejected_count"`
	IsActive         bool        `json:"is_active" db:"is_active"`
	CreatedAt        time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time   `json:"updated_at" db:"updated_at"`
}

// APIKey representa una clave de API para integración
type APIKey struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	EmitterID       uuid.UUID  `json:"emitter_id" db:"emitter_id"`
	Name            string     `json:"name" db:"name"`
	KeyHash         string     `json:"key_hash" db:"key_hash"`
	IsActive        bool       `json:"is_active" db:"is_active"`
	RateLimitPerMin int        `json:"rate_limit_per_min" db:"rate_limit_per_min"`
	CreatedAt       time.Time  `json:"created_at" db:"created_at"`
	LastUsedAt      *time.Time `json:"last_used_at,omitempty" db:"last_used_at"`
}

// CreateEmitterRequest representa el request para crear un emisor
type CreateEmitterRequest struct {
	Name                string  `json:"name" binding:"required"`
	CompanyCode         string  `json:"company_code" binding:"required"`
	RUCTipo             string  `json:"ruc_tipo" binding:"required,oneof=1 2 3"`
	RUCNumero           string  `json:"ruc_numero" binding:"required"`
	RUCDV               string  `json:"ruc_dv" binding:"required"`
	SucEm               string  `json:"suc_em" binding:"required"`
	PtoFacDefault       string  `json:"pto_fac_default" binding:"required"`
	IAmb                int     `json:"iamb" binding:"required,oneof=1 2"`
	ITpEmisDefault      string  `json:"itpemis_default" binding:"required"`
	IDocDefault         string  `json:"idoc_default" binding:"required"`
	Email               string  `json:"email" binding:"required,email"`
	Phone               *string `json:"phone,omitempty"`
	AddressLine         *string `json:"address_line,omitempty"`
	UBICode             *string `json:"ubi_code,omitempty"`
	BrandLogoURL        *string `json:"brand_logo_url,omitempty"`
	BrandPrimaryColor   *string `json:"brand_primary_color,omitempty"`
	BrandFooterHTML     *string `json:"brand_footer_html,omitempty"`
	PACAPIKey           string  `json:"pac_api_key" binding:"required"`
	PACSubscriptionKey  string  `json:"pac_subscription_key" binding:"required"`
}

// CreateSeriesRequest representa el request para crear una serie
type CreateSeriesRequest struct {
	PtoFacDF string      `json:"pto_fac_df" binding:"required"`
	DocKind  DocumentType `json:"doc_kind" binding:"required"`
}

// CreateAPIKeyRequest representa el request para crear una API key
type CreateAPIKeyRequest struct {
	Name            string `json:"name" binding:"required"`
	RateLimitPerMin int    `json:"rate_limit_per_min" binding:"required,min=1,max=10000"`
}

// CreateAPIKeyResponse representa la respuesta al crear una API key
type CreateAPIKeyResponse struct {
	ID              uuid.UUID `json:"id"`
	Name            string    `json:"name"`
	APIKey          string    `json:"api_key"`
	RateLimitPerMin int       `json:"rate_limit_per_min"`
}

// SeriesResponse representa la respuesta para consultar series
type SeriesResponse struct {
	Items    []SeriesItem `json:"items"`
	Page     int          `json:"page"`
	PageSize int          `json:"page_size"`
	Total    int          `json:"total"`
}

// SeriesItem representa un ítem de serie en la respuesta
type SeriesItem struct {
	PtoFacDF        string `json:"pto_fac_df"`
	DocKind         string `json:"doc_kind"`
	LastAssigned    int    `json:"last_assigned"`
	IssuedCount     int    `json:"issued_count"`
	AuthorizedCount int    `json:"authorized_count"`
	RejectedCount   int    `json:"rejected_count"`
}

// DashboardResponse representa la respuesta del dashboard
type DashboardResponse struct {
	Month  string        `json:"month"`
	Series []SeriesItem  `json:"series"`
	Totals DashboardTotals `json:"totals"`
}

// DashboardTotals representa los totales del dashboard
type DashboardTotals struct {
	Issued     int `json:"issued"`
	Authorized int `json:"authorized"`
	Rejected   int `json:"rejected"`
}

// EmitterResponse representa la respuesta al crear un emisor
type EmitterResponse struct {
	ID string `json:"id"`
}
