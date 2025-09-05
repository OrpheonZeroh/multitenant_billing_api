package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hypernova-labs/dgi-service/internal/models"
	"github.com/sirupsen/logrus"
)

// CustomerRepository maneja las operaciones de base de datos para Customer
type CustomerRepository struct {
	db     *DB
	logger *logrus.Logger
}

// NewCustomerRepository crea una nueva instancia del repositorio
func NewCustomerRepository(db *DB, logger *logrus.Logger) *CustomerRepository {
	return &CustomerRepository{
		db:     db,
		logger: logger,
	}
}

// Create crea un nuevo cliente
func (r *CustomerRepository) Create(req *models.CreateCustomerRequest, emitterID uuid.UUID) (*models.Customer, error) {
	customer := &models.Customer{
		ID:         uuid.New(),
		EmitterID:  emitterID,
		Name:       req.Name,
		Email:      req.Email,
		Phone:      req.Phone,
		AddressLine: req.Address,
		UBICode:    req.UBICode,
		IsActive:   true,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	query := `
		INSERT INTO customers (
			id, emitter_id, name, email, phone, address_line, ubi_code, 
			is_active, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10
		)
	`
	
	_, err := r.db.ExecWithTimeout(query,
		customer.ID, customer.EmitterID, customer.Name, customer.Email,
		customer.Phone, customer.AddressLine, customer.UBICode,
		customer.IsActive, customer.CreatedAt, customer.UpdatedAt,
	)
	
	if err != nil {
		return nil, fmt.Errorf("error creating customer: %w", err)
	}

	return customer, nil
}

// GetByID obtiene un cliente por ID
func (r *CustomerRepository) GetByID(id uuid.UUID) (*models.Customer, error) {
	query := `
		SELECT id, emitter_id, name, email, phone, address_line, ubi_code,
			   is_active, created_at, updated_at
		FROM customers
		WHERE id = $1 AND is_active = true
	`
	
	var customer models.Customer
	err := r.db.QueryRowWithTimeout(query, id).Scan(
		&customer.ID, &customer.EmitterID, &customer.Name, &customer.Email,
		&customer.Phone, &customer.AddressLine, &customer.UBICode,
		&customer.IsActive, &customer.CreatedAt, &customer.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("customer not found: %s", id)
		}
		return nil, fmt.Errorf("error querying customer: %w", err)
	}

	return &customer, nil
}

// GetByEmail obtiene un cliente por email y emisor
func (r *CustomerRepository) GetByEmail(emitterID uuid.UUID, email string) (*models.Customer, error) {
	query := `
		SELECT id, emitter_id, name, email, phone, address_line, ubi_code,
			   is_active, created_at, updated_at
		FROM customers
		WHERE emitter_id = $1 AND email = $2 AND is_active = true
	`
	
	var customer models.Customer
	err := r.db.QueryRowWithTimeout(query, emitterID, email).Scan(
		&customer.ID, &customer.EmitterID, &customer.Name, &customer.Email,
		&customer.Phone, &customer.AddressLine, &customer.UBICode,
		&customer.IsActive, &customer.CreatedAt, &customer.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("customer not found with email %s for emitter %s", email, emitterID)
		}
		return nil, fmt.Errorf("error querying customer: %w", err)
	}

	return &customer, nil
}

// GetByEmitterID obtiene todos los clientes de un emisor
func (r *CustomerRepository) GetByEmitterID(emitterID uuid.UUID) ([]models.Customer, error) {
	query := `
		SELECT id, emitter_id, name, email, phone, address_line, ubi_code,
			   is_active, created_at, updated_at
		FROM customers
		WHERE emitter_id = $1 AND is_active = true
		ORDER BY name
	`
	
	rows, err := r.db.QueryWithTimeout(query, emitterID)
	if err != nil {
		return nil, fmt.Errorf("error querying customers: %w", err)
	}
	defer rows.Close()

	var customers []models.Customer
	for rows.Next() {
		var customer models.Customer
		err := rows.Scan(
			&customer.ID, &customer.EmitterID, &customer.Name, &customer.Email,
			&customer.Phone, &customer.AddressLine, &customer.UBICode,
			&customer.IsActive, &customer.CreatedAt, &customer.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning customer: %w", err)
		}
		customers = append(customers, customer)
	}

	return customers, nil
}

// Update actualiza un cliente existente
func (r *CustomerRepository) Update(id uuid.UUID, req *models.CreateCustomerRequest) (*models.Customer, error) {
	query := `
		UPDATE customers 
		SET name = $1, phone = $2, address_line = $3, ubi_code = $4, updated_at = $5
		WHERE id = $6 AND is_active = true
	`
	
	result, err := r.db.ExecWithTimeout(query,
		req.Name, req.Phone, req.Address, req.UBICode, time.Now(), id,
	)
	
	if err != nil {
		return nil, fmt.Errorf("error updating customer: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("error getting rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return nil, fmt.Errorf("customer not found: %s", id)
	}

	// Obtener el cliente actualizado
	return r.GetByID(id)
}

// Delete marca un cliente como inactivo
func (r *CustomerRepository) Delete(id uuid.UUID) error {
	query := `
		UPDATE customers 
		SET is_active = false, updated_at = $1
		WHERE id = $2
	`
	
	result, err := r.db.ExecWithTimeout(query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("error deleting customer: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("customer not found: %s", id)
	}

	return nil
}
