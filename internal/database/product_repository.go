package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hypernova-labs/dgi-service/internal/models"
	"github.com/sirupsen/logrus"
)

// ProductRepository maneja las operaciones de base de datos para Product
type ProductRepository struct {
	db     *DB
	logger *logrus.Logger
}

// NewProductRepository crea una nueva instancia del repositorio
func NewProductRepository(db *DB, logger *logrus.Logger) *ProductRepository {
	return &ProductRepository{
		db:     db,
		logger: logger,
	}
}

// Create crea un nuevo producto
func (r *ProductRepository) Create(req *models.CreateProductRequest, emitterID uuid.UUID) (*models.Product, error) {
	product := &models.Product{
		ID:          uuid.New(),
		EmitterID:   emitterID,
		SKU:         req.SKU,
		Description: req.Description,
		CPBSAbr:     req.CPBSAbr,
		CPBSCmp:     req.CPBSCmp,
		UnitPrice:   req.UnitPrice,
		TaxRate:     req.TaxRate,
		IsActive:    true,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	query := `
		INSERT INTO products (
			id, emitter_id, sku, description, cpbs_abr, cpbs_cmp,
			unit_price, tax_rate, is_active, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
	`
	
	_, err := r.db.ExecWithTimeout(query,
		product.ID, product.EmitterID, product.SKU, product.Description,
		product.CPBSAbr, product.CPBSCmp, product.UnitPrice, product.TaxRate,
		product.IsActive, product.CreatedAt, product.UpdatedAt,
	)
	
	if err != nil {
		return nil, fmt.Errorf("error creating product: %w", err)
	}

	return product, nil
}

// GetByID obtiene un producto por ID
func (r *ProductRepository) GetByID(id uuid.UUID) (*models.Product, error) {
	query := `
		SELECT id, emitter_id, sku, description, cpbs_abr, cpbs_cmp,
			   unit_price, tax_rate, is_active, created_at, updated_at
		FROM products
		WHERE id = $1 AND is_active = true
	`
	
	var product models.Product
	err := r.db.QueryRowWithTimeout(query, id).Scan(
		&product.ID, &product.EmitterID, &product.SKU, &product.Description,
		&product.CPBSAbr, &product.CPBSCmp, &product.UnitPrice, &product.TaxRate,
		&product.IsActive, &product.CreatedAt, &product.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("product not found: %s", id)
		}
		return nil, fmt.Errorf("error querying product: %w", err)
	}

	return &product, nil
}

// GetBySKU obtiene un producto por SKU y emisor
func (r *ProductRepository) GetBySKU(emitterID uuid.UUID, sku string) (*models.Product, error) {
	query := `
		SELECT id, emitter_id, sku, description, cpbs_abr, cpbs_cmp,
			   unit_price, tax_rate, is_active, created_at, updated_at
		FROM products
		WHERE emitter_id = $1 AND sku = $2 AND is_active = true
	`
	
	var product models.Product
	err := r.db.QueryRowWithTimeout(query, emitterID, sku).Scan(
		&product.ID, &product.EmitterID, &product.SKU, &product.Description,
		&product.CPBSAbr, &product.CPBSCmp, &product.UnitPrice, &product.TaxRate,
		&product.IsActive, &product.CreatedAt, &product.UpdatedAt,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("product not found with SKU %s for emitter %s", sku, emitterID)
		}
		return nil, fmt.Errorf("error querying product: %w", err)
	}

	return &product, nil
}

// GetByEmitterID obtiene todos los productos de un emisor
func (r *ProductRepository) GetByEmitterID(emitterID uuid.UUID) ([]models.Product, error) {
	query := `
		SELECT id, emitter_id, sku, description, cpbs_abr, cpbs_cmp,
			   unit_price, tax_rate, is_active, created_at, updated_at
		FROM products
		WHERE emitter_id = $1 AND is_active = true
		ORDER BY description
	`
	
	rows, err := r.db.QueryWithTimeout(query, emitterID)
	if err != nil {
		return nil, fmt.Errorf("error querying products: %w", err)
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var product models.Product
		err := rows.Scan(
			&product.ID, &product.EmitterID, &product.SKU, &product.Description,
			&product.CPBSAbr, &product.CPBSCmp, &product.UnitPrice, &product.TaxRate,
			&product.IsActive, &product.CreatedAt, &product.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning product: %w", err)
		}
		products = append(products, product)
	}

	return products, nil
}

// Update actualiza un producto existente
func (r *ProductRepository) Update(id uuid.UUID, req *models.CreateProductRequest) (*models.Product, error) {
	query := `
		UPDATE products 
		SET description = $1, cpbs_abr = $2, cpbs_cmp = $3, 
		    unit_price = $4, tax_rate = $5, updated_at = $6
		WHERE id = $7 AND is_active = true
	`
	
	result, err := r.db.ExecWithTimeout(query,
		req.Description, req.CPBSAbr, req.CPBSCmp,
		req.UnitPrice, req.TaxRate, time.Now(), id,
	)
	
	if err != nil {
		return nil, fmt.Errorf("error updating product: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("error getting rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return nil, fmt.Errorf("product not found: %s", id)
	}

	// Obtener el producto actualizado
	return r.GetByID(id)
}

// Delete marca un producto como inactivo
func (r *ProductRepository) Delete(id uuid.UUID) error {
	query := `
		UPDATE products 
		SET is_active = false, updated_at = $1
		WHERE id = $2
	`
	
	result, err := r.db.ExecWithTimeout(query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("error deleting product: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("product not found: %s", id)
	}

	return nil
}

// SearchByDescription busca productos por descripción
func (r *ProductRepository) SearchByDescription(emitterID uuid.UUID, description string) ([]models.Product, error) {
	query := `
		SELECT id, emitter_id, sku, description, cpbs_abr, cpbs_cmp,
			   unit_price, tax_rate, is_active, created_at, updated_at
		FROM products
		WHERE emitter_id = $1 AND is_active = true 
		AND description ILIKE $2
		ORDER BY description
	`
	
	searchTerm := "%" + description + "%"
	rows, err := r.db.QueryWithTimeout(query, emitterID, searchTerm)
	if err != nil {
		return nil, fmt.Errorf("error searching products: %w", err)
	}
	defer rows.Close()

	var products []models.Product
	for rows.Next() {
		var product models.Product
		err := rows.Scan(
			&product.ID, &product.EmitterID, &product.SKU, &product.Description,
			&product.CPBSAbr, &product.CPBSCmp, &product.UnitPrice, &product.TaxRate,
			&product.IsActive, &product.CreatedAt, &product.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning product: %w", err)
		}
		products = append(products, product)
	}

	return products, nil
}
