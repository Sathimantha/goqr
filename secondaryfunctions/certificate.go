package secondaryfunctions

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/Sathimantha/goqr/certificate" // Adjust this import based on your structure
)

func GenerateCertificate(studentName, studentID string) (string, error) {
	currentDir, err := filepath.Abs(".")
	if err != nil {
		return "", fmt.Errorf("Error getting current directory: %v", err)
	}

	fontPath := "assets/Roboto-Regular.ttf" // Update this to the actual path of your TTF font
	outputDir := filepath.Join(currentDir, "generated_files")
	certificatePath := filepath.Join(outputDir, studentID+".pdf") // Assuming you name the file using studentID

	// Check if the certificate already exists and delete it
	if _, err := os.Stat(certificatePath); err == nil {
		if err := os.Remove(certificatePath); err != nil {
			return "", fmt.Errorf("Error deleting existing certificate: %v", err)
		}
		log.Printf("Deleted existing certificate: %s\n", certificatePath)
	} else if !os.IsNotExist(err) {
		return "", fmt.Errorf("Error checking for existing certificate: %v", err)
	}

	// Initialize the certificate generator
	generator := certificate.NewGenerator(currentDir, outputDir, fontPath)

	// Generate the certificate
	path, err := generator.GenerateCertificate(studentName, studentID)
	if err != nil {
		return "", fmt.Errorf("Error generating certificate: %v", err)
	}

	log.Printf("Certificate saved at: %s\n", path)
	return path, nil
}
