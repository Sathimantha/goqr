package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Sathimantha/goqr/certificate"
	"github.com/Sathimantha/goqr/secondaryfunctions" // Import the secondaryfunctions package
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

var (
	templateDir = filepath.Join(os.Getenv("PWD"), "templates")
	generator   *certificate.Generator
)

func init() {
	// ... [existing init code] ...
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	// ... [existing home handler code] ...
}

func verifyPageHandler(w http.ResponseWriter, r *http.Request) {
	// ... [existing verify handler code] ...
}

// API endpoint to search for a person
func searchPersonHandler(w http.ResponseWriter, r *http.Request) {
	searchTerm := r.URL.Query().Get("search")
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
		"certificate_link": "/path/to/certificate/" + person.StudentID, // Adjust as needed
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// API endpoint to verify a student
func verifyStudentHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	studentId := vars["studentId"]

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
	r := mux.NewRouter()
	r.Use(handlers.CORS(handlers.AllowedOrigins([]string{"*"})))

	r.HandleFunc("/", homeHandler).Methods("GET")
	r.HandleFunc("/verify", verifyPageHandler).Methods("GET")
	r.HandleFunc("/api/person", searchPersonHandler).Methods("GET")              // New endpoint
	r.HandleFunc("/api/verify/{studentId}", verifyStudentHandler).Methods("GET") // New endpoint

	// Start the server
	log.Fatal(http.ListenAndServe(":5001", r))
}
