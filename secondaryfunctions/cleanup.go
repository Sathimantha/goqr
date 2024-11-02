// secondaryfunctions/cleanup.go

package secondaryfunctions

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

var datePattern = regexp.MustCompile(`\d{4}-\d{2}-\d{2}`)

// CleanupOldFiles checks and deletes certificate files older than specified days
func CleanupOldFiles(daysOld int) error {
	// Query to get all students with remarks
	query := `SELECT student_id, remark FROM students WHERE remark IS NOT NULL AND remark != ''`

	rows, err := db.Query(query)
	if err != nil {
		return fmt.Errorf("error querying database: %v", err)
	}
	defer rows.Close()

	deletedCount := 0
	currentTime := time.Now()

	for rows.Next() {
		var studentID, remark string
		if err := rows.Scan(&studentID, &remark); err != nil {
			log.Printf("Error scanning row: %v", err)
			continue
		}

		// Find all dates in the remark
		dates := datePattern.FindAllString(remark, -1)
		if len(dates) == 0 {
			continue
		}

		// Parse the most recent date
		var mostRecent time.Time
		for _, dateStr := range dates {
			date, err := time.Parse("2006-01-02", dateStr)
			if err != nil {
				continue
			}
			if date.After(mostRecent) {
				mostRecent = date
			}
		}

		// If we found no valid dates, skip this record
		if mostRecent.IsZero() {
			continue
		}

		// Check if the most recent date is older than specified days
		if currentTime.Sub(mostRecent).Hours() > float64(daysOld*24) {
			certPath := filepath.Join("generated_files", studentID+".pdf")

			// Check if file exists
			if _, err := os.Stat(certPath); err == nil {
				// Delete the file
				if err := os.Remove(certPath); err != nil {
					log.Printf("Error deleting file %s: %v", certPath, err)
					continue
				}
				deletedCount++
				log.Printf("Deleted certificate for %s (last accessed: %s)",
					studentID, mostRecent.Format("2006-01-02"))
			}
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating over rows: %v", err)
	}

	log.Printf("Cleanup completed. Deleted %d files older than %d days", deletedCount, daysOld)
	return nil
}
