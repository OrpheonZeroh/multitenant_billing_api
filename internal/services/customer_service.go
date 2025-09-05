package services

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/hypernova-labs/dgi-service/internal/database"
	"github.com/hypernova-labs/dgi-service/internal/models"
	"github.com/sirupsen/logrus"
)

// CustomerService maneja la lógica de negocio para Customer
type CustomerService struct {
	customerRepo *database.CustomerRepository
	logger       *logrus.Logger
}

// NewCustomerService crea una nueva instancia del servicio
func NewCustomerService(db *database.DB, logger *logrus.Logger) *CustomerService {
	return &CustomerService{
		customerRepo: database.NewCustomerRepository(db, logger),
		logger:       logger,
	}
}

// Create crea un nuevo cliente
func (s *CustomerService) Create(req *models.CreateCustomerRequest, emitterID uuid.UUID) (*models.Customer, error) {
	// Validar datos del cliente
	if err := s.validateCustomerData(req); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	// Verificar si el cliente ya existe
	existingCustomer, err := s.customerRepo.GetByEmail(emitterID, req.Email)
	if err == nil && existingCustomer != nil {
		// Cliente ya existe, actualizar datos si es necesario
		s.logger.WithFields(logrus.Fields{
			"emitter_id": emitterID,
			"email":      req.Email,
			"customer_id": existingCustomer.ID,
		}).Info("Customer already exists, updating if needed")

		// TODO: Implementar lógica de actualización si los datos son diferentes
		return existingCustomer, nil
	}

	// Crear nuevo cliente
	customer, err := s.customerRepo.Create(req, emitterID)
	if err != nil {
		return nil, fmt.Errorf("error creating customer: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"emitter_id":  emitterID,
		"customer_id": customer.ID,
		"email":       customer.Email,
		"name":        customer.Name,
	}).Info("Customer created successfully")

	return customer, nil
}

// GetByID obtiene un cliente por ID
func (s *CustomerService) GetByID(id uuid.UUID) (*models.Customer, error) {
	customer, err := s.customerRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("error getting customer: %w", err)
	}

	return customer, nil
}

// GetByEmail obtiene un cliente por email y emisor
func (s *CustomerService) GetByEmail(emitterID uuid.UUID, email string) (*models.Customer, error) {
	customer, err := s.customerRepo.GetByEmail(emitterID, email)
	if err != nil {
		return nil, fmt.Errorf("error getting customer: %w", err)
	}

	return customer, nil
}

// GetByEmitterID obtiene todos los clientes de un emisor
func (s *CustomerService) GetByEmitterID(emitterID uuid.UUID) ([]models.Customer, error) {
	customers, err := s.customerRepo.GetByEmitterID(emitterID)
	if err != nil {
		return nil, fmt.Errorf("error getting customers: %w", err)
	}

	return customers, nil
}

// Update actualiza un cliente existente
func (s *CustomerService) Update(id uuid.UUID, req *models.CreateCustomerRequest) (*models.Customer, error) {
	// Validar datos del cliente
	if err := s.validateCustomerData(req); err != nil {
		return nil, fmt.Errorf("validation error: %w", err)
	}

	// Actualizar cliente
	customer, err := s.customerRepo.Update(id, req)
	if err != nil {
		return nil, fmt.Errorf("error updating customer: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"customer_id": id,
		"email":       customer.Email,
		"name":        customer.Name,
	}).Info("Customer updated successfully")

	return customer, nil
}

// Delete marca un cliente como inactivo
func (s *CustomerService) Delete(id uuid.UUID) error {
	err := s.customerRepo.Delete(id)
	if err != nil {
		return fmt.Errorf("error deleting customer: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"customer_id": id,
	}).Info("Customer deleted successfully")

	return nil
}

// validateCustomerData valida los datos del cliente
func (s *CustomerService) validateCustomerData(req *models.CreateCustomerRequest) error {
	// Validar nombre
	if strings.TrimSpace(req.Name) == "" {
		return fmt.Errorf("name is required")
	}

	// Validar email
	if strings.TrimSpace(req.Email) == "" {
		return fmt.Errorf("email is required")
	}

	// Validar formato de email básico
	if !s.isValidEmail(req.Email) {
		return fmt.Errorf("invalid email format")
	}

	// Validar longitud del nombre
	if len(req.Name) > 255 {
		return fmt.Errorf("name too long (max 255 characters)")
	}

	// Validar longitud del email
	if len(req.Email) > 255 {
		return fmt.Errorf("email too long (max 255 characters)")
	}

	// Validar longitud del teléfono si se proporciona
	if req.Phone != nil && len(*req.Phone) > 20 {
		return fmt.Errorf("phone too long (max 20 characters)")
	}

	// Validar longitud de la dirección si se proporciona
	if req.Address != nil && len(*req.Address) > 500 {
		return fmt.Errorf("address too long (max 500 characters)")
	}

	// Validar longitud del código UBI si se proporciona
	if req.UBICode != nil && len(*req.UBICode) > 10 {
		return fmt.Errorf("UBI code too long (max 10 characters)")
	}

	return nil
}

// isValidEmail valida el formato básico del email
func (s *CustomerService) isValidEmail(email string) bool {
	// Validación básica: debe contener @ y un punto después del @
	if !strings.Contains(email, "@") {
		return false
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}

	if len(parts[0]) == 0 || len(parts[1]) == 0 {
		return false
	}

	if !strings.Contains(parts[1], ".") {
		return false
	}

	return true
}
