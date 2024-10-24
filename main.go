package main

import (
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
	// Get absolute path to the current directory
	currentDir, err := filepath.Abs(".")
	if err != nil {
		log.Fatalf("Error getting current directory: %v\n", err)
	}

	// Define the path to your font file
	fontPath := "assets/Roboto-Regular.ttf" // Update this to the actual path of your TTF font

	// Initialize the certificate generator
	log.Println("Initializing certificate generator...")
	generator = certificate.NewGenerator(currentDir, filepath.Join(currentDir, "generated_files"), fontPath)
	log.Println("Certificate generator initialized successfully.")

	// Initialize database (this will use the init() in secondaryfunctions)
	log.Println("Initializing database connection...")
	_ = secondaryfunctions.DBConfig // Just to trigger the init function
	log.Println("Database connection initialized.")
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Serving home page...")
	http.ServeFile(w, r, filepath.Join(templateDir, "index.html"))
	log.Println("Home page served.")
}

func verifyPageHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Serving verify page...")
	http.ServeFile(w, r, filepath.Join(templateDir, "verify.html"))
	log.Println("Verify page served.")
}

// Additional route handlers can be added here

func main() {
	log.Println("Setting up router...")
	r := mux.NewRouter()
	r.Use(handlers.CORS(handlers.AllowedOrigins([]string{"*"})))

	r.HandleFunc("/", homeHandler).Methods("GET")
	r.HandleFunc("/verify", verifyPageHandler).Methods("GET")

	// Start the server
	log.Println("Starting server on port 5001...")
	if err := http.ListenAndServe(":5001", r); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
	log.Println("Server started successfully.")
}
