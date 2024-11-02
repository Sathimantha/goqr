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
	currentDir, err := filepath.Abs(".")
	if err != nil {
		log.Fatalf("Error getting current directory: %v\n", err)
	}

	fontPath := "assets/Roboto-Regular.ttf"
	generator = certificate.NewGenerator(currentDir, filepath.Join(currentDir, "generated_files"), fontPath)
}

// handleCommandLine processes command-line arguments and executes appropriate actions
func handleCommandLine() error {
	if len(os.Args) <= 1 {
		return fmt.Errorf("no command provided")
	}

	// Define command-line flags
	generateCertCmd := flag.NewFlagSet("generate-cert", flag.ExitOnError)
	cleanupCmd := flag.NewFlagSet("cleanup", flag.ExitOnError)

	// Flags for generate-cert
	studentIDFlag := generateCertCmd.String("id", "", "The Student ID")

	// Flags for cleanup
	daysOldFlag := cleanupCmd.Int("days", 10, "Delete files older than specified days")

	// Process commands
	switch os.Args[1] {
	case "generate-cert":
		if err := generateCertCmd.Parse(os.Args[2:]); err != nil {
			return fmt.Errorf("error parsing generate-cert flags: %v", err)
		}
		if *studentIDFlag == "" {
			return fmt.Errorf("student ID is required")
		}

		return handleGenerateCert(*studentIDFlag)

	case "cleanup":
		if err := cleanupCmd.Parse(os.Args[2:]); err != nil {
			return fmt.Errorf("error parsing cleanup flags: %v", err)
		}

		return handleCleanup(*daysOldFlag)

	default:
		return fmt.Errorf("unknown command: %s", os.Args[1])
	}
}

// handleGenerateCert handles the certificate generation command
func handleGenerateCert(studentID string) error {
	person := secondaryfunctions.GetPerson(studentID, "CLI")
	if person == nil {
		return fmt.Errorf("student not found: %s", studentID)
	}

	if _, err := secondaryfunctions.GenerateCertificate(person.FullName, person.StudentID); err != nil {
		return fmt.Errorf("failed to generate certificate for %s: %v", person.FullName, err)
	}

	fmt.Printf("Certificate successfully generated for %s\n", person.FullName)
	return nil
}

// handleCleanup handles the cleanup command
func handleCleanup(days int) error {
	log.Printf("Starting cleanup of files older than %d days...", days)
	return secondaryfunctions.CleanupOldFiles(days)
}

// setupCORS configures CORS settings for the router
func setupCORS(router *mux.Router) http.Handler {
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
	origins := handlers.AllowedOrigins([]string{"https://cpcglobal.org"})

	return handlers.CORS(headers, methods, origins)(router)
}

// getClientIP extracts the client's IP address from the request
func getClientIP(r *http.Request) string {
	clientIP := strings.Split(r.RemoteAddr, ":")[0]
	if forwardedFor := r.Header.Get("X-Forwarded-For"); forwardedFor != "" {
		clientIP = strings.Split(forwardedFor, ",")[0]
	}
	return clientIP
}

// HTTP Handlers
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
	clientIP := getClientIP(r)
	log.Printf("Search person handler called with search term: %s\n", searchTerm)

	if searchTerm == "" {
		remark := fmt.Sprintf("Request IP: %s | Empty search term in request", clientIP)
		secondaryfunctions.LogError("invalid_request", remark)
		sendJSONError(w, "Search term is required", http.StatusBadRequest)
		return
	}

	person := secondaryfunctions.GetPerson(searchTerm, clientIP)
	if person == nil {
		sendJSONError(w, "Person not found", http.StatusNotFound)
		return
	}

	phoneNo := person.PhoneNo
	if len(phoneNo) > 4 {
		phoneNo = strings.Repeat("*", len(phoneNo)-4) + phoneNo[len(phoneNo)-4:]
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
	clientIP := getClientIP(r)
	log.Printf("Generate certificate handler called with student ID: %s\n", studentId)

	person := secondaryfunctions.GetPerson(studentId, clientIP)
	if person == nil {
		remark := fmt.Sprintf("Request IP: %s | Failed to generate certificate for student ID: %s | Student not found",
			clientIP, studentId)
		secondaryfunctions.LogError("certificate_generation_failure", remark)
		sendJSONError(w, "Student not found", http.StatusNotFound)
		return
	}

	certPath := filepath.Join("generated_files", person.StudentID+".pdf")
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		if _, err := secondaryfunctions.GenerateCertificate(person.FullName, person.StudentID); err != nil {
			remark := fmt.Sprintf("Request IP: %s | Failed to generate certificate for student: %s | Error: %v",
				clientIP, person.StudentID, err)
			secondaryfunctions.LogError("certificate_generation_error", remark)
			sendJSONError(w, "Failed to generate certificate", http.StatusInternalServerError)
			return
		}
	}

	if err := SaveStats(person.StudentID, clientIP); err != nil {
		log.Printf("Error saving stats for %s: %v", person.StudentID, err)
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.pdf", person.StudentID))
	http.ServeFile(w, r, certPath)
}

func verifyStudentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	studentId := vars["studentId"]
	clientIP := getClientIP(r)
	log.Printf("Verify student handler called with student ID: %s\n", studentId)

	if studentId == "" {
		remark := fmt.Sprintf("Request IP: %s | Empty student ID in verification request", clientIP)
		secondaryfunctions.LogError("invalid_request", remark)
		sendJSONError(w, "Student ID is required", http.StatusBadRequest)
		return
	}

	person := secondaryfunctions.GetPerson(studentId, clientIP)
	if person == nil {
		remark := fmt.Sprintf("Request IP: %s | Student not found during verification: %s",
			clientIP, studentId)
		secondaryfunctions.LogError("verification_failure", remark)
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
		remark := fmt.Sprintf("Request IP: %s | Attempt to save stats with empty student ID", clientIP)
		secondaryfunctions.LogError("invalid_stats_request", remark)
		return fmt.Errorf("student ID cannot be empty")
	}

	statsRemark := fmt.Sprintf("Certificate downloaded via Go server at %s from IP %s",
		time.Now().Format(time.RFC3339), clientIP)

	err := secondaryfunctions.AddRemark(studentID, statsRemark, clientIP)
	if err != nil {
		remark := fmt.Sprintf("Request IP: %s | Failed to save stats for student: %s | Error: %v",
			clientIP, studentID, err)
		secondaryfunctions.LogError("stats_save_failure", remark)
		return fmt.Errorf("failed to save stats: %v", err)
	}

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

// startServer initializes and starts the HTTP server
func startServer() error {
	r := mux.NewRouter()

	// Register routes
	registerRoutes(r)

	// Add CORS middleware
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}
			next.ServeHTTP(w, r)
		})
	})

	corsHandler := setupCORS(r)

	// Get SSL certificates
	certFile := os.Getenv("CERT_FILE")
	keyFile := os.Getenv("KEY_FILE")
	if certFile == "" || keyFile == "" {
		return fmt.Errorf("CERT_FILE and KEY_FILE must be defined in the .env file")
	}

	// Start server
	log.Println("Starting server on :5000 with SSL...")
	return http.ListenAndServeTLS(":5000", certFile, keyFile, corsHandler)
}

// registerRoutes sets up all the routes for the server
func registerRoutes(r *mux.Router) {
	r.HandleFunc("/", homeHandler).Methods("GET", "OPTIONS")
	r.HandleFunc("/verify", verifyPageHandler).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/person", searchPersonHandler).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/generate-certificate/{studentId}", generateCertificateHandler).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/verify/{studentId}", verifyStudentHandler).Methods("GET", "OPTIONS")
}

func main() {
	// Handle command-line arguments if present
	if len(os.Args) > 1 {
		if err := handleCommandLine(); err != nil {
			log.Fatalf("Command line error: %v", err)
		}
		return
	}

	// Start server if no command-line arguments
	if err := startServer(); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
