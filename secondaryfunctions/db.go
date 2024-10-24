package secondaryfunctions

import (
	"database/sql"
	"log"

	_ "github.com/go-sql-driver/mysql" // Import MySQL driver for MariaDB
)

var db *sql.DB

func init() {
	var err error
	// Configure the Data Source Name (DSN) for MariaDB
	dsn := DBConfig.Username + ":" + DBConfig.Password + "@tcp(" + DBConfig.Host + ":" + DBConfig.Port + ")/" + DBConfig.Database
	log.Println("Connecting to the database...")
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("Error connecting to the database: %v", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatalf("Database is unreachable: %v", err)
	}

	log.Println("Database connection established successfully.")
}

// Person represents the student object in the database
type Person struct {
	StudentID string // student_id
	FullName  string // full_name
	NID       string // NID
	PhoneNo   string // phone_no
	Remark    string // remark
}

// Exported function to get a person by search term
func GetPerson(searchTerm string) *Person {
	log.Printf("Searching for person with search term: %s\n", searchTerm)
	query := `
		SELECT student_id, full_name, NID, phone_no, remark
		FROM students 
		WHERE student_id = ? OR LOWER(full_name) = LOWER(?) OR NID = ?
	`
	row := db.QueryRow(query, searchTerm, searchTerm, searchTerm)

	var person Person
	if err := row.Scan(&person.StudentID, &person.FullName, &person.NID, &person.PhoneNo, &person.Remark); err != nil {
		if err == sql.ErrNoRows {
			log.Println("Person not found.")
			return nil // Person not found
		}
		log.Printf("Error fetching person: %v\n", err)
		return nil
	}

	log.Printf("Person found: %+v\n", person)
	return &person
}

// Add other DB-related functions (logCertificateDownload, addRemark) here
