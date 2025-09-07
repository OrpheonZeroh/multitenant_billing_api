package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config representa la configuración del servidor
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis    RedisConfig
	Inngest  InngestConfig
	JWT      JWTConfig
	RateLimit RateLimitConfig
	Logging  LoggingConfig
	Email    EmailConfig
	PAC      PACConfig
	Storage  StorageConfig
	Supabase SupabaseConfig
}

// ServerConfig representa la configuración del servidor HTTP
type ServerConfig struct {
	Port     string
	Host     string
	Env      string
	BaseURL  string
}

// DatabaseConfig representa la configuración de la base de datos
type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	SSLMode  string
}

// RedisConfig representa la configuración de Redis
type RedisConfig struct {
	Host     string
	Port     string
	Password string
	DB       int
}

// InngestConfig representa la configuración de Inngest
type InngestConfig struct {
	EventKey        string
	SigningKey      string
	SigningSecret   string
	AppID           string
	Dev             bool
}

// JWTConfig representa la configuración de JWT
type JWTConfig struct {
	Secret  string
	Expiry  time.Duration
}

// RateLimitConfig representa la configuración de rate limiting
type RateLimitConfig struct {
	Default int
	Burst   int
}

// LoggingConfig representa la configuración de logging
type LoggingConfig struct {
	Level  string
	Format string
}

// EmailConfig representa la configuración de email
type EmailConfig struct {
	Host         string
	Port         int
	Username     string
	Password     string
	ResendAPIKey string
}

// PACConfig representa la configuración del PAC
type PACConfig struct {
	APIURL      string
	Timeout     time.Duration
	MaxRetries  int
}

// StorageConfig representa la configuración de almacenamiento
type StorageConfig struct {
	Type   string
	Path   string
	Bucket string
}

// SupabaseConfig representa la configuración de Supabase
type SupabaseConfig struct {
	URL           string
	AnonKey       string
	ServiceKey    string
	ProjectID     string
	ServiceRole   string
	StorageEndpoint string
	StorageRegion   string
	AccessKeyID     string
	SecretAccessKey string
}

// Load carga la configuración desde variables de entorno
func Load() (*Config, error) {
	// Cargar archivo .env si existe
	if err := godotenv.Load(); err != nil {
		// No es crítico si no existe el archivo .env
	}

	config := &Config{
		Server: ServerConfig{
			Port:    getEnv("SERVER_PORT", "8081"),
			Host:    getEnv("SERVER_HOST", "0.0.0.0"),
			Env:     getEnv("SERVER_ENV", "development"),
			BaseURL: getEnv("SERVER_BASE_URL", "http://localhost:8081"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("PGHOST", "localhost"),
			Port:     getEnv("PGPORT", "5432"),
			User:     getEnv("PGUSER", "postgres"),
			Password: getEnv("PGPASSWORD", "postgres"),
			Name:     getEnv("PGDATABASE", "railway"),
			SSLMode:  getEnv("DB_SSLMODE", "require"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnv("REDIS_PORT", "6379"),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		Inngest: InngestConfig{
			EventKey:      getEnv("INNGEST_EVENT_KEY", ""),
			SigningKey:    getEnv("INNGEST_SIGNING_KEY", ""),
			SigningSecret: getEnv("INNGEST_SIGNING_SECRET", ""),
			AppID:         getEnv("INNGEST_APP_ID", "dgi-service"),
			Dev:           getEnvAsBool("INNGEST_DEV", true),
		},
		JWT: JWTConfig{
			Secret: getEnv("JWT_SECRET", "your_jwt_secret_key_here"),
			Expiry: getEnvAsDuration("JWT_EXPIRY", 24*time.Hour),
		},
		RateLimit: RateLimitConfig{
			Default: getEnvAsInt("RATE_LIMIT_DEFAULT", 120),
			Burst:   getEnvAsInt("RATE_LIMIT_BURST", 10),
		},
		Logging: LoggingConfig{
			Level:  getEnv("LOG_LEVEL", "info"),
			Format: getEnv("LOG_FORMAT", "json"),
		},
		Email: EmailConfig{
			Host:         getEnv("SMTP_HOST", "smtp.gmail.com"),
			Port:         getEnvAsInt("SMTP_PORT", 587),
			Username:     getEnv("SMTP_USERNAME", ""),
			Password:     getEnv("SMTP_PASSWORD", ""),
			ResendAPIKey: getEnv("RESEND_API_KEY", ""),
		},
		PAC: PACConfig{
			APIURL:     getEnv("PAC_API_URL", "https://api.pac-provider.com"),
			Timeout:    getEnvAsDuration("PAC_TIMEOUT", 30*time.Second),
			MaxRetries: getEnvAsInt("PAC_MAX_RETRIES", 5),
		},
		Storage: StorageConfig{
			Type:   getEnv("STORAGE_TYPE", "local"),
			Path:   getEnv("STORAGE_PATH", "./storage"),
			Bucket: getEnv("STORAGE_BUCKET", "dgi-documents"),
		},
		Supabase: SupabaseConfig{
			URL:           getEnv("SUPABASE_URL", ""),
			AnonKey:       getEnv("SUPABASE_ANON_KEY", ""),
			ServiceKey:    getEnv("SUPABASE_SERVICE_KEY", ""),
			ProjectID:     getEnv("SUPABASE_PROJECT_ID", ""),
			ServiceRole:   getEnv("SUPABASE_SERVICE_ROLE", ""),
			StorageEndpoint: getEnv("SUPABASE_STORAGE_ENDPOINT", ""),
			StorageRegion:   getEnv("SUPABASE_STORAGE_REGION", ""),
			AccessKeyID:     getEnv("SUPABASE_ACCESS_KEY_ID", ""),
			SecretAccessKey: getEnv("SUPABASE_SECRET_ACCESS_KEY", ""),
		},
	}

	return config, nil
}

// getEnv obtiene una variable de entorno o retorna un valor por defecto
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvAsInt obtiene una variable de entorno como entero
func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getEnvAsBool obtiene una variable de entorno como booleano
func getEnvAsBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

// getEnvAsDuration obtiene una variable de entorno como duración
func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// IsDevelopment retorna true si el entorno es de desarrollo
func (c *Config) IsDevelopment() bool {
	return c.Server.Env == "development"
}

// IsProduction retorna true si el entorno es de producción
func (c *Config) IsProduction() bool {
	return c.Server.Env == "production"
}

// GetDSN retorna la cadena de conexión a la base de datos
func (c *Config) GetDSN() string {
	return "host=" + c.Database.Host +
		" port=" + c.Database.Port +
		" user=" + c.Database.User +
		" password=" + c.Database.Password +
		" dbname=" + c.Database.Name +
		" sslmode=" + c.Database.SSLMode
}

// GetRedisAddr retorna la dirección de Redis
func (c *Config) GetRedisAddr() string {
	return c.Redis.Host + ":" + c.Redis.Port
}
