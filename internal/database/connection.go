package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/hypernova-labs/dgi-service/internal/config"
	_ "github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

// DB representa la conexión a la base de datos
type DB struct {
	*sql.DB
}

// Connect establece la conexión a PostgreSQL
func Connect(cfg *config.Config) (*DB, error) {
	dsn := cfg.GetDSN()
	
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}

	// Configurar pool de conexiones optimizado
	db.SetMaxOpenConns(50)           // Aumentar conexiones máximas
	db.SetMaxIdleConns(25)           // Mantener conexiones inactivas
	db.SetConnMaxLifetime(10 * time.Minute)  // Aumentar tiempo de vida
	db.SetConnMaxIdleTime(5 * time.Minute)   // Tiempo máximo inactivo

	// Verificar conexión
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("error pinging database: %w", err)
	}

	return &DB{db}, nil
}

// Close cierra la conexión a la base de datos
func (db *DB) Close() error {
	return db.DB.Close()
}

// HealthCheck verifica la salud de la base de datos
func (db *DB) HealthCheck() error {
	// Verificar conexión básica
	if err := db.Ping(); err != nil {
		return fmt.Errorf("database ping failed: %w", err)
	}

	// Verificar que podemos ejecutar una query simple
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	_, err := db.QueryContext(ctx, "SELECT 1")
	if err != nil {
		return fmt.Errorf("database query test failed: %w", err)
	}

	return nil
}

// GetStats retorna estadísticas de la base de datos
func (db *DB) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})
	
	// Obtener estadísticas del pool
	stats["max_open_connections"] = db.Stats().MaxOpenConnections
	stats["open_connections"] = db.Stats().OpenConnections
	stats["in_use"] = db.Stats().InUse
	stats["idle"] = db.Stats().Idle
	stats["wait_count"] = db.Stats().WaitCount
	stats["wait_duration"] = db.Stats().WaitDuration
	stats["max_idle_closed"] = db.Stats().MaxIdleClosed
	stats["max_lifetime_closed"] = db.Stats().MaxLifetimeClosed

	return stats
}

// BeginTx inicia una transacción con contexto
func (db *DB) BeginTx() (*sql.Tx, error) {
	return db.Begin()
}

// ExecWithTimeout ejecuta una query con timeout
func (db *DB) ExecWithTimeout(query string, args ...interface{}) (sql.Result, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	return db.ExecContext(ctx, query, args...)
}

// QueryWithTimeout ejecuta una query de lectura con timeout
func (db *DB) QueryWithTimeout(query string, args ...interface{}) (*sql.Rows, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	return db.QueryContext(ctx, query, args...)
}

// QueryRowWithTimeout ejecuta una query de una fila con timeout
func (db *DB) QueryRowWithTimeout(query string, args ...interface{}) *sql.Row {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	return db.QueryRowContext(ctx, query, args...)
}

// WithTransaction ejecuta una función dentro de una transacción
func (db *DB) WithTransaction(fn func(*sql.Tx) error) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("error beginning transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("error rolling back transaction: %w, original error: %w", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing transaction: %w", err)
	}

	return nil
}

// LogStats registra las estadísticas de la base de datos
func (db *DB) LogStats(logger *logrus.Logger) {
	stats := db.GetStats()
	logger.WithFields(logrus.Fields(stats)).Info("Database pool statistics")
}
