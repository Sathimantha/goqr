package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

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
	fontPath := "assets/Roboto-Regular.ttf" // Update this to the actual path of your TTF font

	// Initialize the certificate generator
	generator = certificate.NewGenerator(currentDir, filepath.Join(currentDir, "generated_files"), fontPath)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Home handler called")
	http.ServeFile(w, r, filepath.Join(templateDir, "index.html"))
}

func verifyPageHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Verify page handler called")
	http.ServeFile(w, r, filepath.Join(templateDir, "verify.html"))
}

// API endpoint to search for a person
func searchPersonHandler(w http.ResponseWriter, r *http.Request) {
	searchTerm := r.URL.Query().Get("search")
	log.Printf("Search person handler called with search term: %s\n", searchTerm)

	if searchTerm == "" {
		http.Error(w, "Search term is required", http.StatusBadRequest)
		return
	}

	person := secondaryfunctions.GetPerson(searchTerm)
	if person == nil {
		http.Error(w, "Person not found", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"full_name":        person.FullName,
		"NID":              person.NID,
		"phone_no":         person.PhoneNo,
		"certificate_link": "/api/generate-certificate/" + person.StudentID, // Link to generate the certificate
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// API endpoint to generate a certificate
func generateCertificateHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	studentId := vars["studentId"]
	log.Printf("Generate certificate handler called with student ID: %s\n", studentId)

	person := secondaryfunctions.GetPerson(studentId)
	if person == nil {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	// Check if the certificate already exists
	if _, err := os.Stat(filepath.Join("generated_files", person.StudentID+".pdf")); os.IsNotExist(err) {
		// If the certificate does not exist, generate it
		if _, err := secondaryfunctions.GenerateCertificate(person.FullName, person.StudentID); err != nil {
			http.Error(w, "Failed to generate certificate", http.StatusInternalServerError)
			return
		}
	}

	// Serve the generated certificate
	http.ServeFile(w, r, filepath.Join("generated_files", person.StudentID+".pdf"))
}

// API endpoint to verify a student
func verifyStudentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	studentId := vars["studentId"]
	log.Printf("Verify student handler called with student ID: %s\n", studentId)

	if studentId == "" {
		http.Error(w, "Student ID is required", http.StatusBadRequest)
		return
	}

	person := secondaryfunctions.GetPerson(studentId)
	if person == nil {
		http.Error(w, "Student not found", http.StatusNotFound)
		return
	}

	response := map[string]interface{}{
		"full_name": person.FullName,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	// Check for command-line arguments
	if len(os.Args) > 1 {
		// Use flags to define the command-line options
		generateCertCmd := flag.NewFlagSet("generate-cert", flag.ExitOnError)
		studentIDFlag := generateCertCmd.String("id", "", "The Student ID")
		generateCertCmd.Parse(os.Args[2:]) // Parse flags after the command

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
		// If no command-line arguments, start the server as usual
		r := mux.NewRouter()
		r.Use(handlers.CORS(handlers.AllowedOrigins([]string{"*"})))

		r.HandleFunc("/", homeHandler).Methods("GET")
		r.HandleFunc("/verify", verifyPageHandler).Methods("GET")
		r.HandleFunc("/api/person", searchPersonHandler).Methods("GET")                                  // Search person
		r.HandleFunc("/api/generate-certificate/{studentId}", generateCertificateHandler).Methods("GET") // New endpoint for generating certificates
		r.HandleFunc("/api/verify/{studentId}", verifyStudentHandler).Methods("GET")                     // Verify student

		// Start the server and log the status
		log.Println("Starting server on :5001...")
		log.Fatal(http.ListenAndServe(":5001", r)) // This will block until the server is stopped
	}
}
