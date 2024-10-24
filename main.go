package main

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/Sathimantha/goqr/certificate"
)

func main() {
	// Get absolute path to the current directory
	currentDir, err := filepath.Abs(".")
	if err != nil {
		log.Fatalf("Error getting current directory: %v\n", err)
	}

	// Define the path to your font file
	fontPath := "assets/Roboto-Regular.ttf" // Update this to the actual path of your TTF font

	// Initialize the certificate generator with current directory as base
	generator := certificate.NewGenerator(currentDir, filepath.Join(currentDir, "generated_files"), fontPath)

	studentName := "WIJEKOON RAJAKEERTHI RATHNAYAKE MUDIYANSELAGE KARANDAGOLLE WALAWWE DINITHI THARUSHA POTHUWILA"
	studentID := "123456"

	path, err := generator.GenerateCertificate(studentName, studentID)
	if err != nil {
		log.Fatalf("Error generating certificate: %v\n", err)
	}
	fmt.Printf("Certificate saved at: %s\n", path)
}
