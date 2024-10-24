// certificate/generator.go
package certificate

import (
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"log"
	"os"
	"path/filepath"

	"github.com/jung-kurt/gofpdf"
	"github.com/skip2/go-qrcode"
)

// Generator handles certificate generation operations
type Generator struct {
	BaseDir   string
	OutputDir string
	FontPath  string // Add this field to hold the font path
}

// NewGenerator creates a new certificate generator
func NewGenerator(baseDir, outputDir, fontPath string) *Generator {
	return &Generator{
		BaseDir:   baseDir,
		OutputDir: outputDir,
		FontPath:  fontPath, // Initialize the font path
	}
}

// GenerateCertificate creates a certificate image, overlays text and QR code, and converts it to PDF.
func (g *Generator) GenerateCertificate(studentName, studentID string) (string, error) {
	// Create output directory if it doesn't exist
	if err := os.MkdirAll(g.OutputDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("failed to create output directory: %v", err)
	}

	// Load and process template
	templatePath := filepath.Join(g.BaseDir, "assets/Certificate_Template.jpg")
	rgba, err := g.loadAndProcessTemplate(templatePath, studentName)
	if err != nil {
		return "", fmt.Errorf("failed to process template: %v", err)
	}

	// Generate and overlay QR code
	if err := g.addQRCode(rgba, studentID); err != nil {
		return "", fmt.Errorf("failed to add QR code: %v", err)
	}

	// Save final certificate and convert to PDF
	return g.saveAsPDF(rgba, studentID)
}

func (g *Generator) loadAndProcessTemplate(templatePath string, studentName string) (*image.RGBA, error) {
	templateFile, err := os.Open(templatePath)
	if err != nil {
		return nil, fmt.Errorf("template not found at %s: %v", templatePath, err)
	}
	defer templateFile.Close()

	certificateTemplateImage, err := jpeg.Decode(templateFile)
	if err != nil {
		return nil, fmt.Errorf("failed to decode template: %v", err)
	}

	// Create new RGBA image
	rgba := image.NewRGBA(certificateTemplateImage.Bounds())
	draw.Draw(rgba, rgba.Bounds(), certificateTemplateImage, image.Point{}, draw.Src)

	// Add student name
	imgWidth := certificateTemplateImage.Bounds().Dx()
	fontScale := 150.0 //Max font size

	textRenderer := NewTextRenderer(rgba)
	// Pass the fontPath from the Generator struct to AddText
	textRenderer.AddText(studentName, imgWidth, 3000, fontScale, g.FontPath)

	return rgba, nil
}

func (g *Generator) addQRCode(rgba *image.RGBA, studentID string) error {
	url := fmt.Sprintf("https://cpcglobal.org/verify#%s", studentID)
	qrPath := filepath.Join(g.OutputDir, fmt.Sprintf("%s_qr.png", studentID))

	// Generate QR code
	if err := qrcode.WriteFile(url, qrcode.Medium, 600, qrPath); err != nil {
		return fmt.Errorf("failed to generate QR code: %v", err)
	}
	defer os.Remove(qrPath)

	// Load and overlay QR code
	qr, err := LoadQRCode(qrPath)
	if err != nil {
		return fmt.Errorf("failed to load QR code: %v", err)
	}

	imgWidth := rgba.Bounds().Dx()
	offset := image.Pt(imgWidth-qr.Bounds().Dx()-90, 90)
	draw.Draw(rgba, qr.Bounds().Add(offset), qr, image.Point{}, draw.Over)

	return nil
}

// saveAsPDF saves the final certificate as a PDF file.
func (g *Generator) saveAsPDF(rgba *image.RGBA, studentID string) (string, error) {
	// Save intermediate JPEG
	tempJPEGPath := filepath.Join(g.OutputDir, fmt.Sprintf("%s_final.jpg", studentID))
	if err := SaveAsJPEG(rgba, tempJPEGPath); err != nil {
		return "", fmt.Errorf("failed to save temporary JPEG: %v", err)
	}
	defer os.Remove(tempJPEGPath)

	// Convert to PDF with zero margins
	pdfPath := filepath.Join(g.OutputDir, fmt.Sprintf("%s.pdf", studentID))

	// Create new PDF with zero margins
	pdf := gofpdf.New("P", "mm", "A4", "")

	// Set margins to 0 (left, top, right)
	pdf.SetMargins(0, 0, 0)

	// Also remove auto page break to ensure no bottom margin
	pdf.SetAutoPageBreak(false, 0)

	// Add page and set zero margin for this specific page
	pdf.AddPage()

	// Get page dimensions in mm
	pageWidth, pageHeight := pdf.GetPageSize()

	// Place image to cover entire page
	pdf.Image(tempJPEGPath, 0, 0, pageWidth, pageHeight, false, "", 0, "")

	if err := pdf.OutputFileAndClose(pdfPath); err != nil {
		return "", fmt.Errorf("failed to create PDF: %v", err)
	}

	log.Printf("Certificate generated successfully for student ID: %s\n", studentID)
	return pdfPath, nil
}
