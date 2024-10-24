package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/Sathimantha/goqr/certificate"
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
	http.ServeFile(w, r, filepath.Join(templateDir, "index.html"))
}

func verifyPageHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, filepath.Join(templateDir, "verify.html"))
}

// Additional route handlers can be added here

func main() {
	r := mux.NewRouter()
	r.Use(handlers.CORS(handlers.AllowedOrigins([]string{"*"})))

	r.HandleFunc("/", homeHandler).Methods("GET")
	r.HandleFunc("/verify", verifyPageHandler).Methods("GET")

	// Start the server
	log.Fatal(http.ListenAndServe(":5000", r))
}
