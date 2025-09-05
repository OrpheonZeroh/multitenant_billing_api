package services

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hypernova-labs/dgi-service/internal/database"
	"github.com/hypernova-labs/dgi-service/internal/models"
	"github.com/sirupsen/logrus"
)

// HybridStorageService maneja el almacenamiento híbrido de archivos
type HybridStorageService struct {
	supabaseClient   *database.SupabaseClient
	invoiceFilesRepo *database.InvoiceFilesRepository
	logger           *logrus.Logger
	bucketName       string
}

// NewHybridStorageService crea una nueva instancia del servicio
func NewHybridStorageService(supabaseClient *database.SupabaseClient, invoiceFilesRepo *database.InvoiceFilesRepository, logger *logrus.Logger) *HybridStorageService {
	return &HybridStorageService{
		supabaseClient:   supabaseClient,
		invoiceFilesRepo: invoiceFilesRepo,
		logger:           logger,
		bucketName:       "invoice-files",
	}
}

// StoreInvoiceFiles almacena los archivos de una factura en Supabase y metadatos en BD local
func (s *HybridStorageService) StoreInvoiceFiles(ctx context.Context, invoiceID uuid.UUID, pdfData, xmlData []byte) (*models.InvoiceFilesResponse, error) {
	// Crear nombres de archivo únicos
	pdfFileName := fmt.Sprintf("invoices/%s/factura_%s.pdf", invoiceID, invoiceID)
	xmlFileName := fmt.Sprintf("invoices/%s/factura_%s.xml", invoiceID, invoiceID)

	// Subir PDF a Supabase
	pdfURL, err := s.supabaseClient.UploadFile(ctx, s.bucketName, pdfFileName, pdfData)
	if err != nil {
		return nil, fmt.Errorf("error uploading PDF to Supabase: %w", err)
	}

	// Subir XML a Supabase
	xmlURL, err := s.supabaseClient.UploadFile(ctx, s.bucketName, xmlFileName, xmlData)
	if err != nil {
		return nil, fmt.Errorf("error uploading XML to Supabase: %w", err)
	}

	// Convertir URLs S3 a URLs REST de Supabase para acceso público
	// Cambiar de: https://ambtmugdpopskzxdafdm.storage.supabase.co/storage/v1/s3/invoice-files/...
	// A: https://ambtmugdpopskzxdafdm.supabase.co/storage/v1/object/public/invoice-files/...
	
	// Extraer la parte del path del archivo
	pdfPath := strings.TrimPrefix(pdfURL, "https://ambtmugdpopskzxdafdm.storage.supabase.co/storage/v1/s3/")
	xmlPath := strings.TrimPrefix(xmlURL, "https://ambtmugdpopskzxdafdm.storage.supabase.co/storage/v1/s3/")
	
	// Crear URLs públicas usando la API REST de Supabase
	publicPDFURL := fmt.Sprintf("https://ambtmugdpopskzxdafdm.supabase.co/storage/v1/object/public/%s", pdfPath)
	publicXMLURL := fmt.Sprintf("https://ambtmugdpopskzxdafdm.supabase.co/storage/v1/object/public/%s", xmlPath)

	// Crear registro en BD local con URLs públicas de Supabase
	files := &models.InvoiceFiles{
		ID:          uuid.New(),
		InvoiceID:   invoiceID,
		PDFData:     nil, // No almacenamos datos en BD local
		XMLData:     nil, // No almacenamos datos en BD local
		PDFSize:     int64(len(pdfData)),
		XMLSize:     int64(len(xmlData)),
		PDFURL:      &publicPDFURL,
		XMLURL:      &publicXMLURL,
		GeneratedAt: time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Guardar metadatos en BD local
	if err := s.invoiceFilesRepo.CreateOrUpdate(files); err != nil {
		// Si falla, intentar limpiar archivos de Supabase
		s.supabaseClient.DeleteFile(ctx, s.bucketName, pdfFileName)
		s.supabaseClient.DeleteFile(ctx, s.bucketName, xmlFileName)
		return nil, fmt.Errorf("error saving invoice files metadata: %w", err)
	}

	// Retornar respuesta con URLs públicas de Supabase
	response := &models.InvoiceFilesResponse{
		XMLFE:        &publicXMLURL,
		XMLProtocolo: &publicXMLURL,
		CAFEPDFURL:   &publicPDFURL,
		Disposition:  "inline",
	}

	s.logger.WithFields(logrus.Fields{
		"invoice_id": invoiceID,
		"pdf_url":    publicPDFURL,
		"xml_url":    publicXMLURL,
		"pdf_size":   files.PDFSize,
		"xml_size":   files.XMLSize,
	}).Info("Invoice files stored in Supabase successfully")

	return response, nil
}

// GetInvoiceFiles obtiene los archivos de una factura desde Supabase
func (s *HybridStorageService) GetInvoiceFiles(ctx context.Context, invoiceID uuid.UUID) (*models.InvoiceFilesResponse, error) {
	// Obtener metadatos desde BD local
	files, err := s.invoiceFilesRepo.GetByInvoiceID(invoiceID)
	if err != nil {
		return nil, fmt.Errorf("error getting invoice files metadata: %w", err)
	}

	// Verificar que las URLs existan
	if files.PDFURL == nil || files.XMLURL == nil {
		return nil, fmt.Errorf("invoice files not found in Supabase")
	}

	// Retornar respuesta con URLs de Supabase
	response := &models.InvoiceFilesResponse{
		XMLFE:        files.XMLURL,
		XMLProtocolo: files.XMLURL,
		CAFEPDFURL:   files.PDFURL,
		Disposition:  "inline",
	}

	return response, nil
}

// DownloadInvoiceFile descarga un archivo específico desde Supabase
func (s *HybridStorageService) DownloadInvoiceFile(ctx context.Context, invoiceID uuid.UUID, fileType string) ([]byte, string, error) {
	// Obtener metadatos desde BD local
	files, err := s.invoiceFilesRepo.GetByInvoiceID(invoiceID)
	if err != nil {
		return nil, "", fmt.Errorf("error getting invoice files metadata: %w", err)
	}

	var fileName string
	var bucketPath string

	switch fileType {
	case "pdf":
		if files.PDFURL == nil {
			return nil, "", fmt.Errorf("PDF file not found for invoice %s", invoiceID)
		}
		fileName = fmt.Sprintf("factura_%s.pdf", invoiceID)
		bucketPath = fmt.Sprintf("invoices/%s/factura_%s.pdf", invoiceID, invoiceID)
	case "xml":
		if files.XMLURL == nil {
			return nil, "", fmt.Errorf("XML file not found for invoice %s", invoiceID)
		}
		fileName = fmt.Sprintf("factura_%s.xml", invoiceID)
		bucketPath = fmt.Sprintf("invoices/%s/factura_%s.xml", invoiceID, invoiceID)
	default:
		return nil, "", fmt.Errorf("invalid file type: %s", fileType)
	}

	// Descargar archivo desde Supabase
	fileData, err := s.supabaseClient.DownloadFile(ctx, s.bucketName, bucketPath)
	if err != nil {
		return nil, "", fmt.Errorf("error downloading file from Supabase: %w", err)
	}

	return fileData, fileName, nil
}

// DownloadFile descarga un archivo desde una URL de Supabase
func (s *HybridStorageService) DownloadFile(ctx context.Context, url string) ([]byte, error) {
	// Crear cliente HTTP
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// Crear request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	// Realizar request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error downloading file: %w", err)
	}
	defer resp.Body.Close()

	// Verificar status code
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("error downloading file: HTTP %d", resp.StatusCode)
	}

	// Leer contenido
	fileData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading file content: %w", err)
	}

	return fileData, nil
}

// InitializeBucket inicializa el bucket de Supabase si no existe
func (s *HybridStorageService) InitializeBucket(ctx context.Context) error {
	// Intentar crear el bucket (puede fallar si ya existe)
	err := s.supabaseClient.CreateBucket(ctx, s.bucketName, true)
	if err != nil {
		s.logger.WithError(err).Warn("Bucket may already exist, continuing...")
		
		// Si el bucket ya existe, intentar configurar permisos públicos
		if err := s.configurePublicAccess(ctx); err != nil {
			s.logger.WithError(err).Warn("Could not configure public access for existing bucket")
		}
	} else {
		// Si se creó exitosamente, verificar que los permisos públicos estén configurados
		if err := s.configurePublicAccess(ctx); err != nil {
			s.logger.WithError(err).Warn("Could not configure public access for new bucket")
		}
	}

	s.logger.WithField("bucket", s.bucketName).Info("Supabase storage bucket initialized")
	return nil
}

// configurePublicAccess configura el acceso público al bucket
func (s *HybridStorageService) configurePublicAccess(ctx context.Context) error {
	// Para Supabase, los archivos se suben como públicos por defecto
	// No necesitamos configurar políticas S3 complejas
	
	s.logger.WithField("bucket", s.bucketName).Info("Supabase storage configured for public access")
	return nil
}
