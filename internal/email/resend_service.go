package email

import (
	"fmt"

	"github.com/hypernova-labs/dgi-service/internal/models"
	"github.com/resend/resend-go/v2"
	"github.com/sirupsen/logrus"
)

// ResendService maneja el envío de correos electrónicos usando Resend API
type ResendService struct {
	client    *resend.Client
	fromEmail string
	baseURL   string
	logger    *logrus.Logger
}

// NewResendService crea una nueva instancia de ResendService
func NewResendService(apiKey string, baseURL string, logger *logrus.Logger) *ResendService {
	return &ResendService{
		client:    resend.NewClient(apiKey),
		fromEmail: "onboarding@resend.dev", // Usar dominio verificado de Resend
		baseURL:   baseURL,
		logger:    logger,
	}
}

// SendInvoiceEmail envía un email con la factura adjunta
func (s *ResendService) SendInvoiceEmail(invoice *models.Invoice, customer *models.Customer, emitter *models.Emitter) error {
	// Construir contenido del email
	subject := fmt.Sprintf("Factura #%s - %s", invoice.DocumentNumber, emitter.Name)
	
	// Template HTML del email
	htmlContent := fmt.Sprintf(`
<!DOCTYPE html>
<html>
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Factura Electrónica</title>
    <style>
        body { font-family: Arial, sans-serif; line-height: 1.6; color: #333; }
        .container { max-width: 600px; margin: 0 auto; padding: 20px; }
        .header { background-color: #f8f9fa; padding: 20px; text-align: center; border-radius: 8px; }
        .content { padding: 20px; }
        .button { display: inline-block; padding: 12px 24px; background-color: #007bff; color: white; text-decoration: none; border-radius: 5px; margin: 10px 5px; }
        .button:hover { background-color: #0056b3; }
        .footer { margin-top: 30px; padding: 20px; background-color: #f8f9fa; border-radius: 8px; font-size: 14px; color: #666; }
        .total { font-size: 18px; font-weight: bold; color: #007bff; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Factura Electrónica</h1>
            <p>Número: %s</p>
            <p>Fecha: %s</p>
        </div>
        
        <div class="content">
            <h2>Hola %s,</h2>
            
            <p>Adjunto encontrarás tu factura electrónica con los siguientes detalles:</p>
            
            <ul>
                <li><strong>Emisor:</strong> %s</li>
                <li><strong>RUC:</strong> %s-%s-%s-%s</li>
                <li><strong>Documento:</strong> %s</li>
                <li><strong>Total:</strong> <span class="total">$%.2f</span></li>
            </ul>
            
            <p>Puedes descargar tu factura en los siguientes formatos:</p>
            
            <div style="text-align: center; margin: 20px 0;">
                <a href="%s/v1/invoices/%s/files?file_type=pdf" class="button">📄 Descargar PDF</a>
                <a href="%s/v1/invoices/%s/files?file_type=xml" class="button">📋 Descargar XML</a>
            </div>
            
            <p><strong>Nota:</strong> Los enlaces expiran en 24 horas por seguridad.</p>
        </div>
        
        <div class="footer">
            <p>Este es un email automático del sistema de facturación electrónica.</p>
            <p>Si tienes alguna pregunta, por favor contacta a nuestro equipo de soporte.</p>
        </div>
    </div>
</body>
</html>`,
		invoice.DocumentNumber,
		invoice.CreatedAt.Format("02/01/2006"),
		customer.Name,
		emitter.Name,
		emitter.RUCTipo, emitter.RUCNumero, emitter.RUCDV, emitter.SucEm,
		invoice.DocumentType,
		invoice.TotalAmount,
		s.baseURL,
		invoice.ID,
		s.baseURL,
		invoice.ID)

	// Crear request para Resend
	request := &resend.SendEmailRequest{
		From:    s.fromEmail,
		To:      []string{customer.Email},
		Subject: subject,
		Html:    htmlContent,
	}

	// Enviar email
	result, err := s.client.Emails.Send(request)
	if err != nil {
		return fmt.Errorf("error sending email via Resend: %w", err)
	}

	s.logger.WithFields(logrus.Fields{
		"email_id": result.Id,
		"to":       customer.Email,
		"subject":  subject,
	}).Info("Email sent successfully via Resend")

	return nil
}
