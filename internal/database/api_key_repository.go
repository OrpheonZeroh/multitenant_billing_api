package database

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hypernova-labs/dgi-service/internal/models"
	"github.com/sirupsen/logrus"
)

// APIKeyRepository maneja las operaciones de base de datos para API Keys
type APIKeyRepository struct {
	db     *DB
	logger *logrus.Logger
}

// NewAPIKeyRepository crea una nueva instancia del repositorio
func NewAPIKeyRepository(db *DB, logger *logrus.Logger) *APIKeyRepository {
	return &APIKeyRepository{
		db:     db,
		logger: logger,
	}
}

// Create crea una nueva API key
func (r *APIKeyRepository) Create(emitterID uuid.UUID, name string, rateLimit int) (*models.APIKey, string, error) {
	// Generar API key única
	apiKey := r.generateAPIKey()
	keyHash := r.HashAPIKey(apiKey)

	apiKeyModel := &models.APIKey{
		ID:              uuid.New(),
		EmitterID:       emitterID,
		Name:            name,
		KeyHash:         keyHash,
		IsActive:        true,
		RateLimitPerMin: rateLimit,
		CreatedAt:       time.Now(),
	}

	query := `
		INSERT INTO api_keys (
			id, emitter_id, name, key_hash, is_active, rate_limit_per_min, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		)
	`
	
	_, err := r.db.ExecWithTimeout(query,
		apiKeyModel.ID, apiKeyModel.EmitterID, apiKeyModel.Name,
		apiKeyModel.KeyHash, apiKeyModel.IsActive, apiKeyModel.RateLimitPerMin,
		apiKeyModel.CreatedAt,
	)
	
	if err != nil {
		return nil, "", fmt.Errorf("error creating API key: %w", err)
	}

	return apiKeyModel, apiKey, nil
}

// GetByHash obtiene una API key por su hash con retry logic
func (r *APIKeyRepository) GetByHash(hash string) (*models.APIKey, error) {
	var apiKey *models.APIKey
	var err error
	
	// Retry logic para evitar context canceled
	maxRetries := 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		apiKey, err = r.getByHashWithRetry(hash)
		if err == nil {
			return apiKey, nil
		}
		
		// Si es context canceled, reintentar
		if strings.Contains(err.Error(), "context canceled") && attempt < maxRetries {
			r.logger.Warnf("Attempt %d failed with context canceled, retrying...", attempt)
			time.Sleep(time.Duration(attempt) * time.Second)
			continue
		}
		
		// Si es otro error, no reintentar
		break
	}
	
	return nil, err
}

// getByHashWithRetry es la implementación interna con retry
func (r *APIKeyRepository) getByHashWithRetry(hash string) (*models.APIKey, error) {
	query := `
		SELECT id, emitter_id, name, key_hash, is_active, rate_limit_per_min, created_at, last_used_at
		FROM api_keys
		WHERE key_hash = $1 AND is_active = true
	`
	
	fmt.Printf("DEBUG: Buscando API key con hash: %s\n", hash)
	
	var apiKey models.APIKey
	err := r.db.QueryRowWithTimeout(query, hash).Scan(
		&apiKey.ID, &apiKey.EmitterID, &apiKey.Name, &apiKey.KeyHash,
		&apiKey.IsActive, &apiKey.RateLimitPerMin, &apiKey.CreatedAt, &apiKey.LastUsedAt,
	)
	
	if err != nil {
		fmt.Printf("DEBUG: Error en consulta: %v\n", err)
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("API key not found or inactive")
		}
		return nil, fmt.Errorf("error querying API key: %w", err)
	}

	fmt.Printf("DEBUG: API key encontrada: %s\n", apiKey.Name)
	return &apiKey, nil
}

// GetByEmitterID obtiene todas las API keys de un emisor
func (r *APIKeyRepository) GetByEmitterID(emitterID uuid.UUID) ([]models.APIKey, error) {
	query := `
		SELECT id, emitter_id, name, key_hash, is_active, rate_limit_per_min, 
			   created_at, last_used_at
		FROM api_keys
		WHERE emitter_id = $1
		ORDER BY created_at DESC
	`
	
	rows, err := r.db.QueryWithTimeout(query, emitterID)
	if err != nil {
		return nil, fmt.Errorf("error querying API keys: %w", err)
	}
	defer rows.Close()

	var apiKeys []models.APIKey
	for rows.Next() {
		var apiKey models.APIKey
		err := rows.Scan(
			&apiKey.ID, &apiKey.EmitterID, &apiKey.Name, &apiKey.KeyHash,
			&apiKey.IsActive, &apiKey.RateLimitPerMin, &apiKey.CreatedAt, &apiKey.LastUsedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning API key: %w", err)
		}
		apiKeys = append(apiKeys, apiKey)
	}

	return apiKeys, nil
}

// UpdateLastUsed actualiza la última vez que se usó la API key
func (r *APIKeyRepository) UpdateLastUsed(id uuid.UUID) error {
	query := `
		UPDATE api_keys 
		SET last_used_at = $1
		WHERE id = $2
	`
	
	_, err := r.db.ExecWithTimeout(query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("error updating API key last used: %w", err)
	}

	return nil
}

// Deactivate desactiva una API key
func (r *APIKeyRepository) Deactivate(id uuid.UUID) error {
	query := `
		UPDATE api_keys 
		SET is_active = false
		WHERE id = $1
	`
	
	result, err := r.db.ExecWithTimeout(query, id)
	if err != nil {
		return fmt.Errorf("error deactivating API key: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("API key not found: %s", id)
	}

	return nil
}

// generateAPIKey genera una API key única de 32 caracteres
func (r *APIKeyRepository) generateAPIKey() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	key := make([]byte, 32)
	for i := range key {
		key[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(key)
}

// HashAPIKey genera el hash SHA-256 de la API key
func (r *APIKeyRepository) HashAPIKey(apiKey string) string {
	hash := sha256.Sum256([]byte(apiKey))
	return fmt.Sprintf("%x", hash)
}
