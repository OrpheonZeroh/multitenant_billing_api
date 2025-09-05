package models

import (
	"time"

	"github.com/google/uuid"
)

// Product representa un producto o servicio de un emisor
type Product struct {
	ID          uuid.UUID `json:"id" db:"id"`
	EmitterID   uuid.UUID `json:"emitter_id" db:"emitter_id"`
	SKU         string    `json:"sku" db:"sku"`
	Description string    `json:"description" db:"description"`
	CPBSAbr     *string   `json:"cpbs_abr,omitempty" db:"cpbs_abr"`
	CPBSCmp     *string   `json:"cpbs_cmp,omitempty" db:"cpbs_cmp"`
	UnitPrice   float64   `json:"unit_price" db:"unit_price"`
	TaxRate     string    `json:"tax_rate" db:"tax_rate"`
	IsActive    bool      `json:"is_active" db:"is_active"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// CreateProductRequest representa el request para crear/actualizar un producto
type CreateProductRequest struct {
	SKU         string  `json:"sku" binding:"required"`
	Description string  `json:"description" binding:"required"`
	CPBSAbr     *string `json:"cpbs_abr,omitempty"`
	CPBSCmp     *string `json:"cpbs_cmp,omitempty"`
	UnitPrice   float64 `json:"unit_price" binding:"required,gt=0"`
	TaxRate     string  `json:"tax_rate" binding:"required"`
}

// ProductResponse representa la respuesta al crear un producto
type ProductResponse struct {
	ID string `json:"id"`
}
