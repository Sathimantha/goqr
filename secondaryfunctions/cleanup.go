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

type CleanupStats struct {
	FilesScanned   int
	FilesDeleted   int
	BytesFreed     int64
	ErrorCount     int
	StartTime      time.Time
	Duration       time.Duration
	OldestFileDate time.Time
	NewestFileDate time.Time
}

// CleanupOldFiles checks and deletes certificate files older than specified days
func CleanupOldFiles(daysOld int) error {
	stats := &CleanupStats{
		StartTime: time.Now(),
	}

	// Query to get all students with remarks
	query := `SELECT student_id, remark FROM students WHERE remark IS NOT NULL AND remark != ''`

	rows, err := db.Query(query)
	if err != nil {
		logCleanupError("Database query failed", err, stats)
		return fmt.Errorf("error querying database: %v", err)
	}
	defer rows.Close()

	currentTime := time.Now()

	for rows.Next() {
		var studentID, remark string
		if err := rows.Scan(&studentID, &remark); err != nil {
			stats.ErrorCount++
			log.Printf("Error scanning row: %v", err)
			continue
		}

		stats.FilesScanned++

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

			// Track oldest and newest files for statistics
			if stats.OldestFileDate.IsZero() || date.Before(stats.OldestFileDate) {
				stats.OldestFileDate = date
			}
			if date.After(stats.NewestFileDate) {
				stats.NewestFileDate = date
			}
		}

		// If we found no valid dates, skip this record
		if mostRecent.IsZero() {
			continue
		}

		// Check if the most recent date is older than specified days
		if currentTime.Sub(mostRecent).Hours() > float64(daysOld*24) {
			certPath := filepath.Join("generated_files", studentID+".pdf")

			// Check if file exists and get its size
			if fileInfo, err := os.Stat(certPath); err == nil {
				// Add file size to bytes freed
				stats.BytesFreed += fileInfo.Size()

				// Delete the file
				if err := os.Remove(certPath); err != nil {
					stats.ErrorCount++
					log.Printf("Error deleting file %s: %v", certPath, err)
					continue
				}
				stats.FilesDeleted++
				log.Printf("Deleted certificate for %s (last accessed: %s)",
					studentID, mostRecent.Format("2006-01-02"))
			}
		}
	}

	if err := rows.Err(); err != nil {
		logCleanupError("Error iterating over rows", err, stats)
		return fmt.Errorf("error iterating over rows: %v", err)
	}

	stats.Duration = time.Since(stats.StartTime)
	logCleanupSuccess(daysOld, stats)

	return nil
}

// logCleanupError logs cleanup errors to the errors table
func logCleanupError(message string, err error, stats *CleanupStats) {
	errorRemark := fmt.Sprintf("Cleanup error: %s\nError details: %v\nStats at time of error:\n"+
		"Files scanned: %d\nFiles deleted: %d\nBytes freed: %d\nErrors encountered: %d\n"+
		"Duration: %v",
		message, err, stats.FilesScanned, stats.FilesDeleted, stats.BytesFreed,
		stats.ErrorCount, time.Since(stats.StartTime))

	LogError("cleanup_error", errorRemark)
}

// logCleanupSuccess logs successful cleanup operations to the errors table
func logCleanupSuccess(daysOld int, stats *CleanupStats) {
	successRemark := fmt.Sprintf("Automated cleanup completed successfully:\n"+
		"Cleanup age threshold: %d days\n"+
		"Files scanned: %d\n"+
		"Files deleted: %d\n"+
		"Storage freed: %.2f MB\n"+
		"Errors encountered: %d\n"+
		"Duration: %v\n"+
		"Oldest file found: %s\n"+
		"Newest file found: %s",
		daysOld,
		stats.FilesScanned,
		stats.FilesDeleted,
		float64(stats.BytesFreed)/(1024*1024), // Convert to MB
		stats.ErrorCount,
		stats.Duration,
		stats.OldestFileDate.Format("2006-01-02"),
		stats.NewestFileDate.Format("2006-01-02"))

	LogError("cleanup_success", successRemark)
}

// InitScheduledCleanup starts the cleanup scheduler
func InitScheduledCleanup(daysOld int) {
	go func() {
		for {
			// Wait until midnight
			now := time.Now()
			next := now.Add(24 * time.Hour)
			next = time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, next.Location())
			time.Sleep(next.Sub(now))

			// Run cleanup
			if err := CleanupOldFiles(daysOld); err != nil {
				log.Printf("Scheduled cleanup failed: %v", err)
			}
		}
	}()
}
