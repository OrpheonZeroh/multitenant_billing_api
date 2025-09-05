package models

import (
	"time"

	"github.com/google/uuid"
)

// Customer representa un cliente de un emisor
type Customer struct {
	ID         uuid.UUID `json:"id" db:"id"`
	EmitterID  uuid.UUID `json:"emitter_id" db:"emitter_id"`
	Name       string    `json:"name" db:"name"`
	Email      string    `json:"email" db:"email"`
	Phone      *string   `json:"phone,omitempty" db:"phone"`
	AddressLine *string  `json:"address_line,omitempty" db:"address_line"`
	UBICode    *string   `json:"ubi_code,omitempty" db:"ubi_code"`
	TaxID      *string   `json:"tax_id,omitempty" db:"tax_id"`
	IsActive   bool      `json:"is_active" db:"is_active"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

// CreateCustomerRequest representa el request para crear/actualizar un cliente
type CreateCustomerRequest struct {
	Name       string  `json:"name" binding:"required"`
	Email      string  `json:"email" binding:"required,email"`
	Phone      *string `json:"phone,omitempty"`
	Address    *string `json:"address,omitempty"`
	UBICode    *string `json:"ubi_code,omitempty"`
	TaxID      *string `json:"tax_id,omitempty"`
}

// CustomerResponse representa la respuesta al crear un cliente
type CustomerResponse struct {
	ID string `json:"id"`
}
