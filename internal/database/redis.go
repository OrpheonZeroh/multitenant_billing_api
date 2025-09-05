package database

import (
	"context"
	"fmt"
	"time"

	"github.com/hypernova-labs/dgi-service/internal/config"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// Redis representa la conexión a Redis
type Redis struct {
	*redis.Client
}

// ConnectRedis establece la conexión a Redis
func ConnectRedis(cfg *config.Config) (*Redis, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.GetRedisAddr(),
		Password:     cfg.Redis.Password,
		DB:           cfg.Redis.DB,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 5,
		MaxRetries:   3,
	})

	// Verificar conexión
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("error pinging Redis: %w", err)
	}

	return &Redis{client}, nil
}

// Close cierra la conexión a Redis
func (r *Redis) Close() error {
	return r.Client.Close()
}

// HealthCheck verifica la salud de Redis
func (r *Redis) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	return r.Ping(ctx).Err()
}

// GetStats retorna estadísticas de Redis
func (r *Redis) GetStats() map[string]interface{} {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stats := make(map[string]interface{})
	
	// Obtener estadísticas básicas
	if info, err := r.Info(ctx, "stats").Result(); err == nil {
		stats["info"] = info
	}

	// Obtener memoria
	if mem, err := r.Info(ctx, "memory").Result(); err == nil {
		stats["memory"] = mem
	}

	// Obtener clientes
	if clients, err := r.Info(ctx, "clients").Result(); err == nil {
		stats["clients"] = clients
	}

	return stats
}

// SetWithTTL establece un valor con TTL
func (r *Redis) SetWithTTL(key string, value interface{}, ttl time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	return r.Client.Set(ctx, key, value, ttl).Err()
}

// Get obtiene un valor
func (r *Redis) Get(key string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	return r.Client.Get(ctx, key).Result()
}

// Delete elimina una clave
func (r *Redis) Delete(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	return r.Client.Del(ctx, key).Err()
}

// Exists verifica si existe una clave
func (r *Redis) Exists(key string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	result, err := r.Client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	
	return result > 0, nil
}

// Incr incrementa un contador
func (r *Redis) Incr(key string) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	return r.Client.Incr(ctx, key).Result()
}

// IncrBy incrementa un contador por un valor específico
func (r *Redis) IncrBy(key string, value int64) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	return r.Client.IncrBy(ctx, key, value).Result()
}

// Expire establece TTL para una clave
func (r *Redis) Expire(key string, ttl time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	return r.Client.Expire(ctx, key, ttl).Err()
}

// TTL obtiene el TTL restante de una clave
func (r *Redis) TTL(key string) (time.Duration, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	return r.Client.TTL(ctx, key).Result()
}

// LogStats registra las estadísticas de Redis
func (r *Redis) LogStats(logger *logrus.Logger) {
	stats := r.GetStats()
	logger.WithFields(logrus.Fields(stats)).Info("Redis statistics")
}
