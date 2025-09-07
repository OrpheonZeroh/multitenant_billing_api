package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hypernova-labs/dgi-service/internal/api"
	"github.com/hypernova-labs/dgi-service/internal/config"
	"github.com/hypernova-labs/dgi-service/internal/database"
	"github.com/hypernova-labs/dgi-service/internal/email"
	"github.com/hypernova-labs/dgi-service/internal/services"
	"github.com/hypernova-labs/dgi-service/internal/workflows"
	"github.com/sirupsen/logrus"
)

func main() {
	// Cargar configuración
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	// Configurar logging
	logger := setupLogger(cfg)
	logger.Info("Starting DGI Service...")

	// Configurar modo de Gin
	if cfg.IsDevelopment() {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Conectar a la base de datos
	db, err := database.Connect(cfg)
	if err != nil {
		logger.Fatalf("Error connecting to database: %v", err)
	}
	defer db.Close()

	// Conectar a Redis
	redis, err := database.ConnectRedis(cfg)
	if err != nil {
		logger.Warnf("Error connecting to Redis: %v", err)
		redis = nil
	} else {
		defer redis.Close()
	}

	// Inicializar cliente de Supabase
	var supabaseClient *database.SupabaseClient
	if cfg.Supabase.StorageEndpoint != "" && cfg.Supabase.AccessKeyID != "" && cfg.Supabase.SecretAccessKey != "" {
		supabaseClient, err = database.NewSupabaseClient(&cfg.Supabase, logger)
		if err != nil {
			logger.Warnf("Error initializing Supabase client: %v", err)
			supabaseClient = nil
		} else {
			// Verificar conexión a Supabase
			if err := supabaseClient.HealthCheck(); err != nil {
				logger.Warnf("Supabase health check failed: %v", err)
			} else {
				logger.Info("Supabase storage connection healthy")
			}
		}
	} else {
		logger.Warn("Supabase storage credentials not provided, storage service will not be available")
	}

	// Inicializar servicios
	// Inicializar servicio de Resend
	var resendService *email.ResendService
	if cfg.Email.ResendAPIKey != "" {
		resendService = email.NewResendService(cfg.Email.ResendAPIKey, cfg.Server.BaseURL, logger)
		logger.Info("Resend service initialized successfully")
	} else {
		logger.Warn("Resend API key not provided, email service will not be available")
	}

	// Inicializar cliente de Inngest
	inngestClient, err := workflows.NewInngestClient(cfg, logger)
	if err != nil {
		logger.Warnf("Error initializing Inngest client: %v", err)
		inngestClient = nil
	}
	
	if inngestClient != nil && cfg.Inngest.EventKey != "" && cfg.Inngest.SigningSecret != "" {
		// Registrar workflows
		if err := inngestClient.RegisterWorkflows(resendService, database.NewInvoiceRepository(db, logger), database.NewCustomerRepository(db, logger), database.NewEmitterRepository(db, logger)); err != nil {
			logger.Warnf("Error registering workflows: %v", err)
		}
	} else {
		logger.Warn("Inngest credentials not provided, workflows will not be available")
	}

	// Inicializar más servicios
	invoiceService := services.NewInvoiceService(db, inngestClient, resendService, supabaseClient, logger)
	emitterService := services.NewEmitterService(db, logger)
	customerService := services.NewCustomerService(db, logger)
	productService := services.NewProductService(db, logger)

	// Inicializar repositorio de API Keys
	apiKeyRepo := database.NewAPIKeyRepository(db, logger)

	// Inicializar API
	apiHandler := api.NewAPI(
		invoiceService,
		emitterService,
		customerService,
		productService,
		apiKeyRepo,
		inngestClient,
		logger,
	)

	// Configurar router
	router := setupRouter(apiHandler, cfg)

	// Crear servidor HTTP
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Canal para señales de terminación
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// Iniciar servidor en goroutine
	go func() {
		logger.Infof("Server starting on %s:%s", cfg.Server.Host, cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Error starting server: %v", err)
		}
	}()

	// Esperar señal de terminación
	<-quit
	logger.Info("Shutting down server...")

	// Contexto con timeout para shutdown graceful
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown graceful del servidor
	if err := server.Shutdown(ctx); err != nil {
		logger.Errorf("Server forced to shutdown: %v", err)
	}

	logger.Info("Server exited")
}

// setupLogger configura el logger según la configuración
func setupLogger(cfg *config.Config) *logrus.Logger {
	logger := logrus.New()

	// Configurar nivel de log
	level, err := logrus.ParseLevel(cfg.Logging.Level)
	if err != nil {
		level = logrus.InfoLevel
	}
	logger.SetLevel(level)

	// Configurar formato
	if cfg.Logging.Format == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{
			FullTimestamp: true,
		})
	}

	return logger
}

// setupRouter configura el router principal
func setupRouter(apiHandler *api.API, cfg *config.Config) *gin.Engine {
	router := gin.New()

	// Middleware global
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	// Middleware de CORS para desarrollo
	if cfg.IsDevelopment() {
		router.Use(func(c *gin.Context) {
			c.Header("Access-Control-Allow-Origin", "*")
			c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-API-Key, Idempotency-Key")
			
			if c.Request.Method == "OPTIONS" {
				c.AbortWithStatus(204)
				return
			}
			
			c.Next()
		})
	}

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "ok",
			"timestamp": time.Now().UTC(),
			"service":   "dgi-service",
			"version":   "1.0.0",
		})
	})

	// API v1
	v1 := router.Group("/v1")
	{
		// Endpoints CORE (públicos)
		core := v1.Group("")
		{
			// Emitters (público para registro inicial)
			core.POST("/emitters", apiHandler.CreateEmitter)
			
			// Invoices
			core.POST("/invoices", apiHandler.CreateInvoice)
			core.GET("/invoices/:id", apiHandler.GetInvoice)
			core.GET("/invoices/:id/files", apiHandler.GetInvoiceFiles)
			core.POST("/invoices/:id/email", apiHandler.ResendEmail)
			core.POST("/invoices/:id/retry", apiHandler.RetryWorkflow)
			
			// Series
			core.GET("/series", apiHandler.GetSeries)
		}

		// Endpoints PÚBLICOS (sin autenticación)
		public := v1.Group("/files")
		{
			// Descarga pública de archivos de facturas
			public.GET("/invoices/:id", apiHandler.GetPublicInvoiceFile)
		}

		// Endpoints ADMIN (protegidos)
		admin := v1.Group("")
		admin.Use(apiHandler.AdminAuthMiddleware())
		{
			// Customers
			admin.POST("/customers", apiHandler.CreateCustomer)
			
			// Products
			admin.POST("/products", apiHandler.CreateProduct)
			
			// Emitters (endpoints protegidos)
			admin.POST("/emitters/:id/series", apiHandler.CreateSeries)
			admin.POST("/emitters/:id/apikeys", apiHandler.CreateAPIKey)
			admin.GET("/emitters/:id/dashboard", apiHandler.GetDashboard)
		}
	}

	return router
}
