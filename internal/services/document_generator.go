package services

import (
	"bytes"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/hypernova-labs/dgi-service/internal/models"
	"github.com/jung-kurt/gofpdf"
	"github.com/sirupsen/logrus"
)

// DocumentGenerator maneja la generación de archivos PDF y XML
type DocumentGenerator struct {
	logger *logrus.Logger
}

// NewDocumentGenerator crea una nueva instancia del generador
func NewDocumentGenerator(logger *logrus.Logger) *DocumentGenerator {
	return &DocumentGenerator{
		logger: logger,
	}
}

// GenerateInvoiceFiles genera los archivos PDF y XML para una factura
func (d *DocumentGenerator) GenerateInvoiceFiles(invoice *models.Invoice, customer *models.Customer, emitter *models.Emitter, items []models.InvoiceItem) (*models.InvoiceFiles, error) {
	// Generar PDF
	pdfData, err := d.GenerateInvoicePDF(invoice, customer, emitter, items)
	if err != nil {
		return nil, fmt.Errorf("error generating PDF: %w", err)
	}

	// Generar XML
	xmlData, err := d.GenerateInvoiceXML(invoice, customer, emitter, items)
	if err != nil {
		return nil, fmt.Errorf("error generating XML: %w", err)
	}

	// Crear respuesta
	files := &models.InvoiceFiles{
		ID:          uuid.New(),
		InvoiceID:   invoice.ID,
		PDFData:     pdfData,
		XMLData:     xmlData,
		PDFSize:     int64(len(pdfData)),
		XMLSize:     int64(len(xmlData)),
		GeneratedAt: time.Now(),
		UpdatedAt:   time.Now(),
	}

	d.logger.WithFields(logrus.Fields{
		"invoice_id": invoice.ID,
		"pdf_size":   files.PDFSize,
		"xml_size":   files.XMLSize,
	}).Info("Invoice files generated successfully")

	return files, nil
}

// GenerateInvoicePDF genera un archivo PDF para la factura
func (d *DocumentGenerator) GenerateInvoicePDF(invoice *models.Invoice, customer *models.Customer, emitter *models.Emitter, items []models.InvoiceItem) ([]byte, error) {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()

	// Configurar colores corporativos
	pdf.SetFillColor(41, 128, 185)  // Azul corporativo
	pdf.SetTextColor(44, 62, 80)    // Texto oscuro
	pdf.SetDrawColor(52, 73, 94)    // Bordes

	// Header con color de fondo
	pdf.SetFillColor(41, 128, 185)
	pdf.Rect(0, 0, 210, 40, "F")
	
	// Título principal en blanco
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Arial", "B", 24)
	pdf.Cell(190, 15, "FACTURA")
	pdf.Ln(15)
	
	// Número de factura
	pdf.SetFont("Arial", "B", 16)
	pdf.Cell(190, 10, fmt.Sprintf("#%s", invoice.DocumentNumber))
	pdf.Ln(10)
	
	// Fecha
	pdf.SetFont("Arial", "", 12)
	pdf.Cell(190, 8, fmt.Sprintf("Fecha: %s", invoice.CreatedAt.Format("02/01/2006")))
	pdf.Ln(8)

	// Resetear color de texto
	pdf.SetTextColor(44, 62, 80)
	pdf.SetFillColor(255, 255, 255)

	// Información del emisor (izquierda)
	pdf.SetY(50)
	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(95, 8, "EMISOR")
	pdf.Ln(8)
	
	// Manejar campos de forma segura
	emitterName := emitter.Name
	emitterAddress := "N/A"
	if emitter.AddressLine != nil {
		emitterAddress = *emitter.AddressLine
	}
	
	emitterRUC := fmt.Sprintf("%s-%s-%s-%s", emitter.RUCTipo, emitter.RUCNumero, emitter.RUCDV, emitter.SucEm)
	
	pdf.SetFont("Arial", "", 10)
	pdf.Cell(95, 6, emitterName)
	pdf.Ln(6)
	pdf.Cell(95, 6, fmt.Sprintf("RUC: %s", emitterRUC))
	pdf.Ln(6)
	pdf.Cell(95, 6, emitterAddress)
	pdf.Ln(6)
	
	// Email del emisor
	if emitter.Email != "" {
		pdf.Cell(95, 6, fmt.Sprintf("Email: %s", emitter.Email))
		pdf.Ln(6)
	}

	// Información del cliente (derecha)
	pdf.SetY(50)
	pdf.SetX(105)
	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(95, 8, "CLIENTE")
	pdf.Ln(8)
	
	// Manejar campos de forma segura
	customerName := customer.Name
	customerEmail := customer.Email
	customerAddress := "N/A"
	if customer.AddressLine != nil {
		customerAddress = *customer.AddressLine
	}
	
	customerRUC := "N/A"
	if customer.TaxID != nil {
		customerRUC = *customer.TaxID
	}
	
	pdf.SetFont("Arial", "", 10)
	pdf.Cell(95, 6, customerName)
	pdf.Ln(6)
	pdf.Cell(95, 6, fmt.Sprintf("RUC: %s", customerRUC))
	pdf.Ln(6)
	pdf.Cell(95, 6, customerEmail)
	pdf.Ln(6)
	pdf.Cell(95, 6, customerAddress)
	pdf.Ln(6)

	// Información adicional de la factura
	pdf.SetY(120)
	pdf.SetX(10)
	pdf.SetFont("Arial", "B", 12)
	pdf.Cell(190, 8, fmt.Sprintf("Punto de Facturación: %s", invoice.PtoFacDF))
	pdf.Ln(8)

	// Tabla de items con estilo mejorado
	pdf.SetY(140)
	
	// Header de la tabla
	pdf.SetFillColor(236, 240, 241)  // Gris claro
	pdf.SetTextColor(44, 62, 80)
	pdf.SetFont("Arial", "B", 10)
	
	// Columnas de la tabla
	colWidths := []float64{20, 70, 25, 30, 35}
	colHeaders := []string{"Línea", "Descripción", "Cantidad", "Precio Unit.", "Total"}
	
	for i, header := range colHeaders {
		pdf.CellFormat(colWidths[i], 10, header, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(10)

	// Items de la factura
	pdf.SetFillColor(255, 255, 255)
	pdf.SetFont("Arial", "", 9)
	rowHeight := 8.0
	
	for i, item := range items {
		// Alternar colores de fila
		if i%2 == 0 {
			pdf.SetFillColor(248, 249, 250)
		} else {
			pdf.SetFillColor(255, 255, 255)
		}
		
		pdf.CellFormat(colWidths[0], rowHeight, fmt.Sprintf("%d", item.LineNo), "1", 0, "C", true, 0, "")
		pdf.CellFormat(colWidths[1], rowHeight, item.Description, "1", 0, "L", true, 0, "")
		pdf.CellFormat(colWidths[2], rowHeight, fmt.Sprintf("%.2f", item.Quantity), "1", 0, "R", true, 0, "")
		pdf.CellFormat(colWidths[3], rowHeight, fmt.Sprintf("%.2f", item.UnitPrice), "1", 0, "R", true, 0, "")
		pdf.CellFormat(colWidths[4], rowHeight, fmt.Sprintf("%.2f", item.LineTotal), "1", 0, "R", true, 0, "")
		pdf.Ln(rowHeight)
	}

	// Totales con estilo mejorado
	totalY := pdf.GetY() + 10
	pdf.SetY(totalY)
	
	// Línea separadora
	pdf.SetDrawColor(189, 195, 199)
	pdf.Line(120, totalY, 200, totalY)
	pdf.Ln(5)
	
	// Totales
	pdf.SetFont("Arial", "B", 12)
	pdf.SetX(120)
	pdf.Cell(50, 8, "Subtotal:")
	pdf.Cell(30, 8, fmt.Sprintf("$%.2f", invoice.Subtotal))
	pdf.Ln(8)
	
	pdf.SetX(120)
	pdf.Cell(50, 8, "ITBMS (7%):")
	pdf.Cell(30, 8, fmt.Sprintf("$%.2f", invoice.ITBMSAmount))
	pdf.Ln(8)
	
	// Total final destacado
	pdf.SetFillColor(41, 128, 185)
	pdf.SetTextColor(255, 255, 255)
	pdf.SetX(120)
	pdf.Cell(50, 12, "TOTAL:")
	pdf.Cell(30, 12, fmt.Sprintf("$%.2f", invoice.TotalAmount))
	pdf.Ln(12)

	// Footer
	pdf.SetY(270)
	pdf.SetTextColor(149, 165, 166)
	pdf.SetFont("Arial", "", 8)
	pdf.Cell(190, 6, "Esta factura fue generada electrónicamente por el sistema DGI Service")
	pdf.Ln(6)
	pdf.Cell(190, 6, fmt.Sprintf("Generado el: %s", time.Now().Format("02/01/2006 15:04:05")))

	// Generar bytes del PDF usando buffer
	var buf bytes.Buffer
	err := pdf.Output(&buf)
	if err != nil {
		return nil, fmt.Errorf("error generating PDF: %w", err)
	}
	
	return buf.Bytes(), nil
}

// GenerateInvoiceXML genera un archivo XML para la factura
func (d *DocumentGenerator) GenerateInvoiceXML(invoice *models.Invoice, customer *models.Customer, emitter *models.Emitter, items []models.InvoiceItem) ([]byte, error) {
	// XML básico para la factura
	xmlContent := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<factura>
    <emisor>
        <ruc>%s-%s-%s-%s</ruc>
        <nombre>%s</nombre>
        <direccion>%s</direccion>
    </emisor>
    <cliente>
        <ruc>%s</ruc>
        <nombre>%s</nombre>
        <email>%s</email>
        <direccion>%s</direccion>
    </cliente>
    <documento>
        <numero>%s</numero>
        <fecha>%s</fecha>
        <ptoFac>%s</ptoFac>
        <tipo>%s</tipo>
    </documento>
    <items>`,
		emitter.RUCTipo, emitter.RUCNumero, emitter.RUCDV, emitter.SucEm,
		emitter.Name,
		func() string {
			if emitter.AddressLine != nil {
				return *emitter.AddressLine
			}
			return "N/A"
		}(),
		func() string {
			if customer.TaxID != nil {
				return *customer.TaxID
			}
			return "N/A"
		}(),
		customer.Name,
		customer.Email,
		func() string {
			if customer.AddressLine != nil {
				return *customer.AddressLine
			}
			return "N/A"
		}(),
		invoice.DocumentNumber,
		invoice.CreatedAt.Format("2006-01-02"),
		invoice.PtoFacDF,
		invoice.DocumentType,
	)

	// Agregar items
	for _, item := range items {
		xmlContent += fmt.Sprintf(`
        <item>
            <linea>%d</linea>
            <descripcion>%s</descripcion>
            <cantidad>%.2f</cantidad>
            <precioUnitario>%.2f</precioUnitario>
            <total>%.2f</total>
        </item>`,
			item.LineNo,
			item.Description,
			item.Quantity,
			item.UnitPrice,
			item.LineTotal,
		)
	}

	// Cerrar XML
	xmlContent += `
    </items>
    <totales>
        <subtotal>%.2f</subtotal>
        <itbms>%.2f</itbms>
        <total>%.2f</total>
    </totales>
</factura>`

	xmlContent = fmt.Sprintf(xmlContent, invoice.Subtotal, invoice.ITBMSAmount, invoice.TotalAmount)

	return []byte(xmlContent), nil
}
