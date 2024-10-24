package secondaryfunctions

import (
	"database/sql"
	"fmt"
	"log"
	"time"

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
func AddRemark(studentID, newRemark string) error {
	// Fetch the current remark for the student
	var currentRemark string
	query := `SELECT remark FROM students WHERE student_id = ?`
	err := db.QueryRow(query, studentID).Scan(&currentRemark)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Student with ID %s not found when adding remark.\n", studentID)
			return fmt.Errorf("student not found")
		}
		log.Printf("Error fetching current remark: %v\n", err)
		return err
	}

	// Append the new remark with a timestamp
	timestamp := time.Now().Format(time.RFC3339)
	updatedRemark := fmt.Sprintf("%s\n%s - %s", currentRemark, timestamp, newRemark)

	// Update the remark in the database
	updateQuery := `UPDATE students SET remark = ? WHERE student_id = ?`
	_, err = db.Exec(updateQuery, updatedRemark, studentID)
	if err != nil {
		log.Printf("Error updating remark for student %s: %v\n", studentID, err)
		return err
	}

	log.Printf("Remark updated for student %s: %s\n", studentID, updatedRemark)
	return nil
}
