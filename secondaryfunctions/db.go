package secondaryfunctions

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var db *sql.DB

// ValidationPatterns holds the regex patterns for different types of search terms
var ValidationPatterns = struct {
	StudentID *regexp.Regexp
	Name      *regexp.Regexp
	NID       *regexp.Regexp
}{
	StudentID: regexp.MustCompile(`^[A-Z0-9]{1,10}$`),
	Name:      regexp.MustCompile(`^[^;'\\"#]{1,150}$`),
	NID:       regexp.MustCompile(`^[^;'\\"#]{5,30}$`),
}

func init() {
	var err error
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
	StudentID string
	FullName  string
	NID       string
	PhoneNo   string
	Remark    string
}

// LogError logs any type of error to the errors table
func LogError(errorType string, remark string) error {
	query := `
        INSERT INTO errors (
            timestamp, 
            error_type, 
            remark
        ) VALUES (?, ?, ?)
    `

	_, err := db.Exec(
		query,
		time.Now(),
		errorType,
		remark,
	)

	if err != nil {
		log.Printf("Error logging to errors table: %v\n", err)
		return err
	}

	return nil
}

func isValidSearchTerm(term, requestIP string) bool {
	// Check if term is empty or exceeds length limit
	if term == "" || len(term) > 150 {
		remark := fmt.Sprintf("Request IP: %s | Empty or oversized search term received: %s", requestIP, term)
		if err := LogError("validation_failure", remark); err != nil {
			log.Printf("Failed to log error: %v\n", err)
		}
		return false
	}

	// Remove allowed characters (letters, numbers, spaces, hyphens, periods)
	cleanedTerm := regexp.MustCompile(`[^A-Za-z0-9 .-]`).ReplaceAllString(term, "")

	// Check if the term is blank after cleanup
	if cleanedTerm == "" {
		remark := fmt.Sprintf("Request IP: %s | Search term resulted in blank after cleanup: %s", requestIP, term)
		if err := LogError("validation_failure", remark); err != nil {
			log.Printf("Failed to log error: %v\n", err)
		}
		return false
	}

	// Validate term against known patterns
	isValid := ValidationPatterns.StudentID.MatchString(term) ||
		ValidationPatterns.Name.MatchString(term) ||
		ValidationPatterns.NID.MatchString(term)

	if !isValid {
		remark := fmt.Sprintf("Request IP: %s | Invalid search term pattern: %s", requestIP, term)
		if err := LogError("validation_failure", remark); err != nil {
			log.Printf("Failed to log error: %v\n", err)
		}
	}

	return isValid
}

func GetPerson(searchTerm, requestIP string) *Person {
	log.Printf("Validating search term: %s from IP: %s\n", searchTerm, requestIP)

	if !isValidSearchTerm(searchTerm, requestIP) {
		log.Printf("Search term validation failed: %s\n", searchTerm)
		return nil
	}

	query := `
			SELECT student_id, full_name, NID, phone_no, remark
			FROM students
			WHERE student_id = ?
				OR LOWER(REGEXP_REPLACE(full_name, '[^A-Za-z0-9]', '')) = 
					LOWER(REGEXP_REPLACE(?, '[^A-Za-z0-9]', ''))
				OR LOWER(REGEXP_REPLACE(NID, '[^A-Za-z0-9]', '')) = 
					LOWER(REGEXP_REPLACE(?, '[^A-Za-z0-9]', ''))
				OR (
					REGEXP_REPLACE(NID, '[^0-9]', '') != '' 
					AND REGEXP_REPLACE(NID, '[^0-9]', '') = REGEXP_REPLACE(?, '[^0-9]', '')
				);
	`

	row := db.QueryRow(query, searchTerm, searchTerm, searchTerm, searchTerm)

	var person Person
	if err := row.Scan(&person.StudentID, &person.FullName, &person.NID, &person.PhoneNo, &person.Remark); err != nil {
		if err == sql.ErrNoRows {
			remark := fmt.Sprintf("Request IP: %s | No matching record found for search term: %s", requestIP, searchTerm)
			LogError("record_not_found", remark)
			return nil
		}
		remark := fmt.Sprintf("Request IP: %s | Database error while fetching person: %s | Error: %v",
			requestIP, searchTerm, err)
		LogError("database_error", remark)
		return nil
	}

	return &person
}

func AddRemark(studentID, newRemark, requestIP string) error {
	var currentRemark string
	query := `SELECT remark FROM students WHERE student_id = ?`
	err := db.QueryRow(query, studentID).Scan(&currentRemark)
	if err != nil {
		if err == sql.ErrNoRows {
			remark := fmt.Sprintf("Request IP: %s | Student not found when adding remark: %s",
				requestIP, studentID)
			LogError("student_not_found", remark)
			return fmt.Errorf("student not found")
		}
		remark := fmt.Sprintf("Request IP: %s | Database error while fetching current remark for student: %s | Error: %v",
			requestIP, studentID, err)
		LogError("database_error", remark)
		return err
	}

	timestamp := time.Now().Format(time.RFC3339)
	updatedRemark := fmt.Sprintf("%s\n%s - %s", currentRemark, timestamp, newRemark)

	updateQuery := `UPDATE students SET remark = ? WHERE student_id = ?`
	_, err = db.Exec(updateQuery, updatedRemark, studentID)
	if err != nil {
		remark := fmt.Sprintf("Request IP: %s | Error updating remark for student: %s | Error: %v",
			requestIP, studentID, err)
		LogError("database_error", remark)
		return err
	}

	log.Printf("Remark updated for student %s\n", studentID)
	return nil
}
