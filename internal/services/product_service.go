package services

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/hypernova-labs/dgi-service/internal/database"
	"github.com/hypernova-labs/dgi-service/internal/models"
	"github.com/sirupsen/logrus"
)

// ProductService maneja la lógica de negocio para Product
type ProductService struct {
	productRepo *database.ProductRepository
	logger      *logrus.Logger
}

// NewProductService crea una nueva instancia del servicio
func NewProductService(db *database.DB, logger *logrus.Logger) *ProductService {
	return &ProductService{
		productRepo: database.NewProductRepository(db, logger),
		logger:      logger,
	}
}

// Create crea un nuevo producto
func (s *ProductService) Create(req *models.CreateProductRequest, emitterID uuid.UUID) (*models.Product, error) {
	s.logger.Infof("Create: req=%+v, emitterID=%s", req, emitterID)
	
	// Validar datos del producto
	if err := s.validateProductData(req); err != nil {
		s.logger.Errorf("Validation error: %v", err)
		return nil, fmt.Errorf("validation error: %w", err)
	}

	// Verificar si el producto ya existe por SKU
	if req.SKU != "" {
		existingProduct, err := s.productRepo.GetBySKU(emitterID, req.SKU)
		if err == nil && existingProduct != nil {
			s.logger.WithFields(logrus.Fields{
				"emitter_id": emitterID,
				"sku":        req.SKU,
				"product_id": existingProduct.ID,
			}).Warn("Product with SKU already exists")

			// TODO: Implementar lógica de actualización o retornar error de conflicto
			return nil, fmt.Errorf("product with SKU %s already exists", req.SKU)
		}
	}

	// Crear nuevo producto
	s.logger.Infof("Creating product with req=%+v, emitterID=%s", req, emitterID)
	product, err := s.productRepo.Create(req, emitterID)
	if err != nil {
		s.logger.Errorf("Error creating product: %v", err)
		return nil, fmt.Errorf("error creating product: %w", err)
	}
	
	s.logger.Infof("Product created successfully: %+v", product)

	s.logger.WithFields(logrus.Fields{
		"emitter_id":  emitterID,
		"product_id":  product.ID,
		"sku":         product.SKU,
		"description": product.Description,
		"unit_price":  product.UnitPrice,
	}).Info("Product created successfully")

	return product, nil
}

// GetByID obtiene un producto por ID
func (s *ProductService) GetByID(id uuid.UUID) (*models.Product, error) {
	product, err := s.productRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("error getting product: %w", err)
	}

	return product, nil
}

// GetBySKU obtiene un producto por SKU y emisor
func (s *ProductService) GetBySKU(emitterID uuid.UUID, sku string) (*models.Product, error) {
	product, err := s.productRepo.GetBySKU(emitterID, sku)
	if err != nil {
		return nil, fmt.Errorf("error getting product: %w", err)
	}

	return product, nil
}

// GetByEmitterID obtiene todos los productos de un emisor
func (s *ProductService) GetByEmitterID(emitterID uuid.UUID) ([]models.Product, error) {
	products, err := s.productRepo.GetByEmitterID(emitterID)
	if err != nil {
		return nil, fmt.Errorf("error getting products: %w", err)
	}

	return products, nil
}

// Update actualiza un producto existente
func (s *ProductService) Update(id uuid.UUID, req *models.CreateProductRequest) (*models.Product, error) {
	// Validar datos del producto
	if err := s.validateProductData(req); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	// Verificar que el producto existe
	existingProduct, err := s.productRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("error getting existing product: %w", err)
	}

	// Verificar que el SKU no esté duplicado si se está cambiando
	if req.SKU != existingProduct.SKU {
		duplicateProduct, err := s.productRepo.GetBySKU(existingProduct.EmitterID, req.SKU)
		if err == nil && duplicateProduct != nil && duplicateProduct.ID != id {
			return nil, fmt.Errorf("product with SKU %s already exists", req.SKU)
		}
	}

	// Actualizar producto
	product, err := s.productRepo.Update(id, req)
	if err != nil {
		return nil, fmt.Errorf("error updating product: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"product_id":  id,
		"sku":         product.SKU,
		"description": product.Description,
		"unit_price":  product.UnitPrice,
	}).Info("Product updated successfully")

	return product, nil
}

// Delete marca un producto como inactivo
func (s *ProductService) Delete(id uuid.UUID) error {
	err := s.productRepo.Delete(id)
	if err != nil {
		return fmt.Errorf("error deleting product: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"product_id": id,
	}).Info("Product deleted successfully")

	return nil
}

// SearchByDescription busca productos por descripción
func (s *ProductService) SearchByDescription(emitterID uuid.UUID, description string) ([]models.Product, error) {
	if strings.TrimSpace(description) == "" {
		return nil, fmt.Errorf("description search term is required")
	}

	products, err := s.productRepo.SearchByDescription(emitterID, description)
	if err != nil {
		return nil, fmt.Errorf("error searching products: %w", err)
	}

	return products, nil
}

// validateProductData valida los datos del producto
func (s *ProductService) validateProductData(req *models.CreateProductRequest) error {
	// Validar descripción
	if strings.TrimSpace(req.Description) == "" {
		return fmt.Errorf("description is required")
	}

	// Validar que el SKU no esté vacío
	if req.SKU == "" {
		return fmt.Errorf("SKU is required")
	}

	// Validar precio unitario
	if req.UnitPrice <= 0 {
		return fmt.Errorf("unit price must be greater than 0")
	}

	// Validar tasa de impuesto
	if strings.TrimSpace(req.TaxRate) == "" {
		return fmt.Errorf("tax rate is required")
	}

	// Validar tasas de impuesto válidas
	validTaxRates := []string{"00", "01", "02", "03"}
	isValidTaxRate := false
	for _, validRate := range validTaxRates {
		if req.TaxRate == validRate {
			isValidTaxRate = true
			break
		}
	}
	if !isValidTaxRate {
		return fmt.Errorf("invalid tax rate: %s (must be one of: %v)", req.TaxRate, validTaxRates)
	}

	// Validar longitud de la descripción
	if len(req.Description) > 500 {
		return fmt.Errorf("description too long (max 500 characters)")
	}

	// Validar longitud del SKU
	if len(req.SKU) > 50 {
		return fmt.Errorf("SKU too long (max 50 characters)")
	}

	// Validar longitud del código CPBS si se proporciona
	if req.CPBSAbr != nil && len(*req.CPBSAbr) > 10 {
		return fmt.Errorf("CPBS abbreviation too long (max 10 characters)")
	}

	if req.CPBSCmp != nil && len(*req.CPBSCmp) > 10 {
		return fmt.Errorf("CPBS component too long (max 10 characters)")
	}

	// Validar límites de precio
	if req.UnitPrice > 999999999.99 {
		return fmt.Errorf("unit price too high (max 999,999,999.99)")
	}

	return nil
}

// CalculateLineTotal calcula el total de una línea de producto
func (s *ProductService) CalculateLineTotal(quantity, unitPrice float64) float64 {
	return quantity * unitPrice
}

// CalculateTaxAmount calcula el monto del impuesto para una línea
func (s *ProductService) CalculateTaxAmount(lineTotal float64, taxRate string) (float64, error) {
	var taxRateFloat float64
	switch taxRate {
	case "00":
		taxRateFloat = 0.0
	case "01":
		taxRateFloat = 0.07
	case "02":
		taxRateFloat = 0.10
	case "03":
		taxRateFloat = 0.15
	default:
		return 0, fmt.Errorf("invalid tax rate: %s", taxRate)
	}

	return lineTotal * taxRateFloat, nil
}
