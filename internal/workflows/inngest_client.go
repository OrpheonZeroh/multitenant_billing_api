package workflows

import (
	"fmt"

	"github.com/hypernova-labs/dgi-service/internal/config"
	"github.com/hypernova-labs/dgi-service/internal/database"
	"github.com/hypernova-labs/dgi-service/internal/email"
	"github.com/inngest/inngestgo"
	"github.com/sirupsen/logrus"
)

// InngestClient maneja la configuración y registro de workflows
type InngestClient struct {
	client inngestgo.Client
	logger *logrus.Logger
}

// NewInngestClient crea una nueva instancia del cliente
func NewInngestClient(cfg *config.Config, logger *logrus.Logger) (*InngestClient, error) {
	// Verificar que las credenciales estén configuradas
	if cfg.Inngest.EventKey == "" {
		return nil, fmt.Errorf("INNGEST_EVENT_KEY not configured")
	}

	if cfg.Inngest.SigningKey == "" {
		return nil, fmt.Errorf("INNGEST_SIGNING_KEY not configured")
	}

	// Crear cliente de Inngest
	client, err := inngestgo.NewClient(inngestgo.ClientOpts{
		EventKey:    &cfg.Inngest.EventKey,
		SigningKey:  &cfg.Inngest.SigningKey,
		AppID:       cfg.Inngest.AppID,
	})
	if err != nil {
		return nil, fmt.Errorf("error creating Inngest client: %w", err)
	}

	return &InngestClient{
		client: client,
		logger: logger,
	}, nil
}

// RegisterWorkflows registra todos los workflows con Inngest
func (c *InngestClient) RegisterWorkflows(emailService *email.ResendService, invoiceRepo *database.InvoiceRepository, customerRepo *database.CustomerRepository, emitterRepo *database.EmitterRepository) error {
	c.logger.Info("Registering workflows with Inngest")

	// TODO: Implementar registro real de workflows
	// Por ahora solo logueamos que se intentó registrar
	c.logger.Info("Workflow registration placeholder - not yet implemented")

	return nil
}

// GetClient retorna el cliente de Inngest
func (c *InngestClient) GetClient() inngestgo.Client {
	return c.client
}
