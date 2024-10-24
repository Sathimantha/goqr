package secondaryfunctions

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/Sathimantha/goqr/certificate" // Adjust this import based on your structure
)

func GenerateCertificate(studentName, studentID string) (string, error) {
	currentDir, err := filepath.Abs(".")
	if err != nil {
		return "", fmt.Errorf("Error getting current directory: %v", err)
	}

	fontPath := "assets/Roboto-Regular.ttf" // Update this to the actual path of your TTF font

	// Initialize the certificate generator
	generator := certificate.NewGenerator(currentDir, filepath.Join(currentDir, "generated_files"), fontPath)

	// Generate the certificate
	path, err := generator.GenerateCertificate(studentName, studentID)
	if err != nil {
		return "", fmt.Errorf("Error generating certificate: %v", err)
	}

	log.Printf("Certificate saved at: %s\n", path)
	return path, nil
}
