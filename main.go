package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Sathimantha/goqr/certificate"
	"github.com/Sathimantha/goqr/secondaryfunctions"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var (
	templateDir = filepath.Join(os.Getenv("PWD"), "templates")
	generator   *certificate.Generator
)

var (
	certificateGenerationTracker = struct {
		sync.RWMutex
		inProgress map[string]bool
		completed  map[string]time.Time
	}{
		inProgress: make(map[string]bool),
		completed:  make(map[string]time.Time),
	}
)

// Add WebSocket upgrader
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Or implement proper origin checking
	},
}

// Add WebSocket client tracking
var (
	clientConnections = struct {
		sync.RWMutex
		clients map[string]map[*websocket.Conn]bool // studentID -> connections
	}{
		clients: make(map[string]map[*websocket.Conn]bool),
	}
)

func init() {
	currentDir, err := filepath.Abs(".")
	if err != nil {
		log.Fatalf("Error getting current directory: %v\n", err)
	}

	fontPath := "assets/Roboto-Regular.ttf"
	generator = certificate.NewGenerator(currentDir, filepath.Join(currentDir, "generated_files"), fontPath)

	go func() {
		for {
			time.Sleep(time.Hour)
			certificateGenerationTracker.Lock()
			for id, completedTime := range certificateGenerationTracker.completed {
				if time.Since(completedTime) > 24*time.Hour {
					delete(certificateGenerationTracker.completed, id)
				}
			}
			certificateGenerationTracker.Unlock()
		}
	}()
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
	studentIDFlag := generateCertCmd.String("id", "", "The Student ID or range (e.g., 'ST001' or 'ST001-ST010')")

	// Flags for cleanup
	daysOldFlag := cleanupCmd.Int("days", 10, "Delete files older than specified days")

	// Process commands
	switch os.Args[1] {
	case "generate-cert":
		if err := generateCertCmd.Parse(os.Args[2:]); err != nil {
			return fmt.Errorf("error parsing generate-cert flags: %v", err)
		}
		if *studentIDFlag == "" {
			return fmt.Errorf("student ID or range is required")
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

func parseIDRange(idRange string) (start, end string, err error) {
	parts := strings.Split(idRange, "-")
	if len(parts) == 1 {
		// Single ID
		return parts[0], parts[0], nil
	} else if len(parts) == 2 {
		// Range of IDs
		start, end = strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
		if start == "" || end == "" {
			return "", "", fmt.Errorf("invalid ID range format: %s", idRange)
		}
		return start, end, nil
	}
	return "", "", fmt.Errorf("invalid ID range format: %s", idRange)
}

func generateNextID(currentID string) (string, error) {
	// Extract the numeric part
	i := len(currentID) - 1
	for i >= 0 && (currentID[i] >= '0' && currentID[i] <= '9') {
		i--
	}
	prefix := currentID[:i+1]
	numStr := currentID[i+1:]

	// Convert to number and increment
	num, err := strconv.Atoi(numStr)
	if err != nil {
		return "", fmt.Errorf("invalid ID format: %s", currentID)
	}

	// Generate next ID with same padding
	padding := len(numStr)
	nextNum := num + 1
	nextNumStr := fmt.Sprintf("%0*d", padding, nextNum)

	return prefix + nextNumStr, nil
}

// handleGenerateCert handles the certificate generation command
func handleGenerateCert(idRange string) error {
	start, end, err := parseIDRange(idRange)
	if err != nil {
		return err
	}

	// For single ID case
	if start == end {
		return generateSingleCertificate(start)
	}

	// For range of IDs
	currentID := start
	for {
		if err := generateSingleCertificate(currentID); err != nil {
			return fmt.Errorf("failed at ID %s: %v", currentID, err)
		}

		if currentID == end {
			break
		}

		nextID, err := generateNextID(currentID)
		if err != nil {
			return fmt.Errorf("failed to generate next ID after %s: %v", currentID, err)
		}

		currentID = nextID
	}

	return nil
}

// Modify initiateAsyncCertificateGeneration to notify clients
func initiateAsyncCertificateGeneration(studentID string, clientIP string) {
	certificateGenerationTracker.RLock()
	if _, inProgress := certificateGenerationTracker.inProgress[studentID]; inProgress {
		certificateGenerationTracker.RUnlock()
		return
	}
	if completedTime, exists := certificateGenerationTracker.completed[studentID]; exists {
		if time.Since(completedTime) < time.Hour {
			certificateGenerationTracker.RUnlock()
			return
		}
	}
	certificateGenerationTracker.RUnlock()

	certificateGenerationTracker.Lock()
	if certificateGenerationTracker.inProgress[studentID] {
		certificateGenerationTracker.Unlock()
		return
	}
	certificateGenerationTracker.inProgress[studentID] = true
	certificateGenerationTracker.Unlock()

	// Start async generation
	go func() {
		defer func() {
			certificateGenerationTracker.Lock()
			delete(certificateGenerationTracker.inProgress, studentID)
			certificateGenerationTracker.completed[studentID] = time.Now()
			certificateGenerationTracker.Unlock()

			// Notify all connected clients
			notifyClients(studentID, "complete")
		}()

		_, err := secondaryfunctions.GenerateCertificate(studentID, clientIP)
		if err != nil {
			remark := fmt.Sprintf("Request IP: %s | Failed to pre-generate certificate for student: %s | Error: %v",
				clientIP, studentID, err)
			secondaryfunctions.LogError("certificate_pregeneration_failure", remark)
			notifyClients(studentID, "error")
			return
		}
	}()
}

// Add function to notify WebSocket clients
func notifyClients(studentID string, status string) {
	message := map[string]string{
		"type":   "certificate_status",
		"status": status,
	}

	clientConnections.RLock()
	if clients, exists := clientConnections.clients[studentID]; exists {
		for conn := range clients {
			err := conn.WriteJSON(message)
			if err != nil {
				log.Printf("Failed to send WebSocket message: %v", err)
			}
		}
	}
	clientConnections.RUnlock()
}

// Add WebSocket connection handler
func websocketHandler(w http.ResponseWriter, r *http.Request) {
	studentID := r.URL.Query().Get("studentId")
	if studentID == "" {
		http.Error(w, "Student ID is required", http.StatusBadRequest)
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}

	// Register client connection
	clientConnections.Lock()
	if _, exists := clientConnections.clients[studentID]; !exists {
		clientConnections.clients[studentID] = make(map[*websocket.Conn]bool)
	}
	clientConnections.clients[studentID][conn] = true
	clientConnections.Unlock()

	// Cleanup on disconnect
	defer func() {
		clientConnections.Lock()
		delete(clientConnections.clients[studentID], conn)
		if len(clientConnections.clients[studentID]) == 0 {
			delete(clientConnections.clients, studentID)
		}
		clientConnections.Unlock()
		conn.Close()
	}()

	// Keep connection alive and handle client messages
	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func generateSingleCertificate(studentID string) error {
	person := secondaryfunctions.GetPerson(studentID, "CLI")
	if person == nil {
		return fmt.Errorf("student not found: %s", studentID)
	}

	if _, err := secondaryfunctions.GenerateCertificate(person.FullName, person.StudentID); err != nil {
		return fmt.Errorf("failed to generate certificate for %s: %v", person.FullName, err)
	}

	fmt.Printf("Certificate successfully generated for %s (%s)\n", person.FullName, person.StudentID)
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

	origins := handlers.AllowedOrigins([]string{
		"https://cpcglobal.org",
		"https://cdn.cpcglobal.org"})

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

	// Initiate async certificate generation
	initiateAsyncCertificateGeneration(person.StudentID, clientIP)

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

// DownloadStatus represents the status of a certificate download
type DownloadStatus struct {
	StartTime time.Time
	FileSize  int64
	Completed bool
}

// downloadTracker maintains a map of active downloads
var downloadTracker = struct {
	sync.RWMutex
	downloads map[string]*DownloadStatus
}{
	downloads: make(map[string]*DownloadStatus),
}

// Initialize random seed
func init() {
	rand.Seed(time.Now().UnixNano())
}

func generateCertificateHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	studentId := vars["studentId"]
	clientIP := getClientIP(r)
	log.Printf("Generate certificate handler called with student ID: %s\n", studentId)

	// First verify that the student exists
	person := secondaryfunctions.GetPerson(studentId, clientIP)
	if person == nil {
		remark := fmt.Sprintf("Request IP: %s | Failed to generate certificate for student ID: %s | Student not found",
			clientIP, studentId)
		secondaryfunctions.LogError("certificate_generation_failure", remark)
		sendJSONError(w, "Student not found", http.StatusNotFound)
		return
	}

	// Check if certificate is currently being generated
	certificateGenerationTracker.RLock()
	if certificateGenerationTracker.inProgress[studentId] {
		certificateGenerationTracker.RUnlock()
		// Return a 202 Accepted status with a message
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{
			"message": "Certificate generation in progress, please try again in a few moments",
		})
		return
	}
	certificateGenerationTracker.RUnlock()

	// Check if the certificate file already exists
	certPath := filepath.Join("generated_files", studentId+".pdf")
	if _, err := os.Stat(certPath); os.IsNotExist(err) {
		// Certificate doesn't exist, start generation
		certificateGenerationTracker.Lock()
		certificateGenerationTracker.inProgress[studentId] = true
		certificateGenerationTracker.Unlock()

		// Generate certificate
		if _, err := secondaryfunctions.GenerateCertificate(person.FullName, person.StudentID); err != nil {
			certificateGenerationTracker.Lock()
			delete(certificateGenerationTracker.inProgress, studentId)
			certificateGenerationTracker.Unlock()

			remark := fmt.Sprintf("Request IP: %s | Failed to generate certificate for student: %s | Error: %v",
				clientIP, person.StudentID, err)
			secondaryfunctions.LogError("certificate_generation_error", remark)
			sendJSONError(w, "Failed to generate certificate", http.StatusInternalServerError)
			return
		}

		certificateGenerationTracker.Lock()
		delete(certificateGenerationTracker.inProgress, studentId)
		certificateGenerationTracker.completed[studentId] = time.Now()
		certificateGenerationTracker.Unlock()
	}

	// Get file information
	fileInfo, err := os.Stat(certPath)
	if err != nil {
		sendJSONError(w, "Failed to get certificate information", http.StatusInternalServerError)
		return
	}
	fileSize := fileInfo.Size()

	// Generate a unique download ID
	downloadID := fmt.Sprintf("%s-%s-%d", person.StudentID, time.Now().Format("20060102150405"), rand.Int63())

	// Initialize download tracking
	downloadTracker.Lock()
	downloadTracker.downloads[downloadID] = &DownloadStatus{
		StartTime: time.Now(),
		FileSize:  fileSize,
		Completed: false,
	}
	downloadTracker.Unlock()

	// Set headers for download
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s.pdf", person.StudentID))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileSize))
	w.Header().Set("X-Download-ID", downloadID)

	// Create a wrapped response writer to track completion
	downloadWriter := &downloadResponseWriter{
		ResponseWriter: w,
		downloadID:     downloadID,
		written:        0,
		totalSize:      fileSize,
	}

	// Serve the file
	http.ServeFile(downloadWriter, r, certPath)

	// After serving, check if download was completed
	downloadTracker.RLock()
	status := downloadTracker.downloads[downloadID]
	downloadTracker.RUnlock()

	if status != nil && status.Completed {
		if err := SaveStats(person.StudentID, clientIP); err != nil {
			log.Printf("Error saving stats for %s: %v", person.StudentID, err)
		}

		// Log successful download
		log.Printf("Certificate download completed successfully for student ID: %s, Download ID: %s",
			person.StudentID, downloadID)
	} else {
		// Log incomplete download
		remark := fmt.Sprintf("Request IP: %s | Incomplete certificate download for student: %s | Download ID: %s",
			clientIP, person.StudentID, downloadID)
		secondaryfunctions.LogError("incomplete_download", remark)
	}

	// Clean up download tracking after a delay
	go func() {
		time.Sleep(1 * time.Hour) // Keep tracking info for 1 hour
		downloadTracker.Lock()
		delete(downloadTracker.downloads, downloadID)
		downloadTracker.Unlock()
	}()
}

// downloadResponseWriter wraps http.ResponseWriter to track download progress
type downloadResponseWriter struct {
	http.ResponseWriter
	downloadID string
	written    int64
	totalSize  int64
}

func (w *downloadResponseWriter) Write(p []byte) (int, error) {
	n, err := w.ResponseWriter.Write(p)
	if err != nil {
		return n, err
	}

	w.written += int64(n)

	// Check if download is complete
	if w.written >= w.totalSize {
		downloadTracker.Lock()
		if status := downloadTracker.downloads[w.downloadID]; status != nil {
			status.Completed = true
		}
		downloadTracker.Unlock()
	}

	return n, nil
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

	// Save verification record as a remark
	verificationRemark := fmt.Sprintf("Certificate verified via Go server at %s from IP %s",
		time.Now().Format(time.RFC3339), clientIP)

	if err := secondaryfunctions.AddRemark(studentId, verificationRemark, clientIP); err != nil {
		remark := fmt.Sprintf("Request IP: %s | Failed to save verification record for student: %s | Error: %v",
			clientIP, studentId, err)
		secondaryfunctions.LogError("verification_record_failure", remark)
		// Continue with the response even if logging fails
		log.Printf("Failed to save verification record: %v", err)
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

	// Log server listening status
	listeningRemark := fmt.Sprintf("Server listening on :5000 with SSL at %s\nCertificate File: %s\nKey File: %s",
		time.Now().Format(time.RFC3339),
		certFile,
		keyFile)
	if err := secondaryfunctions.LogError("server_listening", listeningRemark); err != nil {
		log.Printf("Failed to log server listening status: %v", err)
	}

	log.Println("Starting server on :5000 with SSL...")
	return http.ListenAndServeTLS(":5000", certFile, keyFile, corsHandler)
}

// registerRoutes sets up all the routes for the server
func registerRoutes(r *mux.Router) {
	r.HandleFunc("/", homeHandler).Methods("GET", "OPTIONS")
	r.HandleFunc("/verify", verifyPageHandler).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/person", searchPersonHandler).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/generate-certificate/{studentId}", generateCertificateHandler).Methods("GET", "HEAD", "OPTIONS")
	r.HandleFunc("/api/verify/{studentId}", verifyStudentHandler).Methods("GET", "OPTIONS")
	r.HandleFunc("/ws", websocketHandler)
}

func main() {
	// Log program startup
	startupRemark := fmt.Sprintf("Server started at %s\nEnvironment:\n"+
		"Template Directory: %s\n"+
		"Certificate File: %s\n"+
		"Key File: %s",
		time.Now().Format(time.RFC3339),
		templateDir,
		os.Getenv("CERT_FILE"),
		os.Getenv("KEY_FILE"))

	if err := secondaryfunctions.LogError("server_startup", startupRemark); err != nil {
		log.Printf("Failed to log server startup: %v", err)
	}

	// Handle command-line arguments if present
	if len(os.Args) > 1 {
		if err := handleCommandLine(); err != nil {
			log.Fatalf("Command line error: %v", err)
		}
		return
	}

	// Initialize scheduled cleanup before starting the server
	secondaryfunctions.InitScheduledCleanup(10)

	// Start server if no command-line arguments
	if err := startServer(); err != nil {
		shutdownRemark := fmt.Sprintf("Server shutdown with error at %s: %v",
			time.Now().Format(time.RFC3339), err)
		secondaryfunctions.LogError("server_shutdown", shutdownRemark)
		log.Fatalf("Server error: %v", err)
	}
}
