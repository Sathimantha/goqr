package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Sathimantha/goqr/certificate"
	"github.com/Sathimantha/goqr/secondaryfunctions"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

var (
	templateDir = filepath.Join(os.Getenv("PWD"), "templates")
	generator   *certificate.Generator
)

func init() {
	// Get absolute path to the current directory
	currentDir, err := filepath.Abs(".")
	if err != nil {
		log.Fatalf("Error getting current directory: %v\n", err)
	}

	// Define the path to your font file
	fontPath := "assets/Roboto-Regular.ttf"

	// Initialize the certificate generator
	generator = certificate.NewGenerator(currentDir, filepath.Join(currentDir, "generated_files"), fontPath)
}

func setupCORS(router *mux.Router) http.Handler {
	// Configure CORS
	headers := handlers.AllowedHeaders([]string{
		"X-Requested-With",
		"Content-Type",
		"Authorization",
		"Accept",
		"Origin",
	})
	methods := handlers.AllowedMethods([]string{
		"GET",
		"POST",
		"PUT",
		"DELETE",
		"OPTIONS",
	})
	// Allow specific origin
	origins := handlers.AllowedOrigins([]string{"*"})

	// Return handler with CORS middleware
	return handlers.CORS(headers, methods, origins)(router)
}

// Request handlers
func homeHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Home handler called")
	http.ServeFile(w, r, filepath.Join(templateDir, "index.html"))
}

func verifyPageHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Verify page handler called")
	http.ServeFile(w, r, filepath.Join(templateDir, "verify.html"))
}

func searchPersonHandler(w http.ResponseWriter, r *http.Request) {
	searchTerm := r.URL.Query().Get("search")
	log.Printf("Search person handler called with search term: %s\n", searchTerm)

	if searchTerm == "" {
		sendJSONError(w, "Search term is required", http.StatusBadRequest)
		return
	}

	person := secondaryfunctions.GetPerson(searchTerm)
	if person == nil {
		sendJSONError(w, "Person not found", http.StatusNotFound)
		return
	}

	// Obfuscate the phone number
	phoneNo := person.PhoneNo
	if len(phoneNo) > 4 {
		phoneNo = strings.Repeat("*", len(phoneNo)-4) + phoneNo[len(phoneNo)-4:]
	} else {
		phoneNo = phoneNo // Return as-is if the phone number is too short
	}

	response := map[string]interface{}{
		"full_name":        person.FullName,
		"NID":              person.NID,
		"phone_no":         phoneNo,
		"certificate_link": "/api/generate-certificate/" + person.StudentID,
	}
	sendJSONResponse(w, response, http.StatusOK)
}

func generateCertificateHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	studentId := vars["studentId"]
	log.Printf("Generate certificate handler called with student ID: %s\n", studentId)

	person := secondaryfunctions.GetPerson(studentId)
	if person == nil {
		sendJSONError(w, "Student not found", http.StatusNotFound)
		return
	}

	// Check if the certificate already exists
	certPath := filepath.Join("generated_files", person.StudentID+".pdf")
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		// If the certificate does not exist, generate it
		if _, err := secondaryfunctions.GenerateCertificate(person.FullName, person.StudentID); err != nil {
			log.Printf("Failed to generate certificate: %v", err)
			sendJSONError(w, "Failed to generate certificate", http.StatusInternalServerError)
			return
		}
	}

	// Extract client IP address from the request
	clientIP := strings.Split(r.RemoteAddr, ":")[0]

	// Save the stats with the client IP
	if err := SaveStats(person.StudentID, clientIP); err != nil {
		log.Printf("Error saving stats for %s: %v", person.StudentID, err)
		// Continue serving the file even if stats saving fails
	}

	// Set appropriate headers for file download
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.pdf", person.StudentID))

	// Serve the generated certificate
	http.ServeFile(w, r, certPath)
}

func verifyStudentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	studentId := vars["studentId"]
	log.Printf("Verify student handler called with student ID: %s\n", studentId)

	if studentId == "" {
		sendJSONError(w, "Student ID is required", http.StatusBadRequest)
		return
	}

	person := secondaryfunctions.GetPerson(studentId)
	if person == nil {
		sendJSONError(w, "Student not found", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"full_name": person.FullName,
		"NID":       person.NID,
	}
	sendJSONResponse(w, response, http.StatusOK)
}

func SaveStats(studentID string, clientIP string) error {
	if studentID == "" {
		return fmt.Errorf("student ID cannot be empty")
	}

	// Create the remark with the timestamp and client IP
	remark := fmt.Sprintf("Certificate downloaded via Go server at %s from IP %s", time.Now().Format(time.RFC3339), clientIP)

	// Log the attempt
	log.Printf("Attempting to save stats for student %s with remark: %s\n", studentID, remark)

	// Log the remark in the database
	err := secondaryfunctions.AddRemark(studentID, remark)
	if err != nil {
		log.Printf("Failed to save stats for student %s: %v\n", studentID, err)
		return fmt.Errorf("failed to save stats: %v", err)
	}

	log.Printf("Stats successfully saved for student %s\n", studentID)
	return nil
}

// Helper functions for JSON responses
func sendJSONResponse(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func sendJSONError(w http.ResponseWriter, message string, status int) {
	sendJSONResponse(w, map[string]string{"error": message}, status)
}

func main() {
	// Check for command-line arguments
	if len(os.Args) > 1 {
		// Use flags to define the command-line options
		generateCertCmd := flag.NewFlagSet("generate-cert", flag.ExitOnError)
		studentIDFlag := generateCertCmd.String("id", "", "The Student ID")
		generateCertCmd.Parse(os.Args[2:])

		switch os.Args[1] {
		case "generate-cert":
			if err := generateCertCmd.Parse(os.Args[2:]); err != nil {
				log.Fatalf("Error parsing flags: %v", err)
			}
			if *studentIDFlag == "" {
				log.Fatal("Student ID is required")
			}

			// Fetch person from database
			person := secondaryfunctions.GetPerson(*studentIDFlag)
			if person == nil {
				log.Fatalf("Student not found: %s", *studentIDFlag)
			}

			// Manually generate the certificate
			if _, err := secondaryfunctions.GenerateCertificate(person.FullName, person.StudentID); err != nil {
				log.Fatalf("Failed to generate certificate for %s: %v", person.FullName, err)
			}
			fmt.Printf("Certificate successfully generated for %s\n", person.FullName)
			return

		default:
			log.Fatalf("Unknown command: %s", os.Args[1])
		}
	} else {
		// Create and configure the router
		r := mux.NewRouter()

		// Add route handlers
		r.HandleFunc("/", homeHandler).Methods("GET", "OPTIONS")
		r.HandleFunc("/verify", verifyPageHandler).Methods("GET", "OPTIONS")
		r.HandleFunc("/api/person", searchPersonHandler).Methods("GET", "OPTIONS")
		r.HandleFunc("/api/generate-certificate/{studentId}", generateCertificateHandler).Methods("GET", "OPTIONS")
		r.HandleFunc("/api/verify/{studentId}", verifyStudentHandler).Methods("GET", "OPTIONS")

		// Add middleware to handle preflight requests
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Handle preflight requests
				if r.Method == "OPTIONS" {
					w.WriteHeader(http.StatusOK)
					return
				}
				next.ServeHTTP(w, r)
			})
		})

		// Setup CORS and create the final handler
		corsHandler := setupCORS(r)

		certFile := os.Getenv("CERT_FILE")
		keyFile := os.Getenv("KEY_FILE")
		// Ensure both cert and key files are defined
		if certFile == "" || keyFile == "" {
			log.Fatal("CERT_FILE and KEY_FILE must be defined in the .env file")
		}

		// Start the server with TLS
		log.Println("Starting server on :5000 with SSL...")
		log.Fatal(http.ListenAndServeTLS(":5000", certFile, keyFile, corsHandler))
	}
}
