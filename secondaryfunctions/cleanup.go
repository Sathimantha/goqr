// secondaryfunctions/cleanup.go

package secondaryfunctions

import (
	"fmt"
	"log"
	"math"
	"math/rand"
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

	// Directory path for generated files
	generatedFilesDir := "generated_files"

	// Read all files in the directory
	files, err := os.ReadDir(generatedFilesDir)
	if err != nil {
		logCleanupError("Failed to read directory", err, stats)
		return fmt.Errorf("error reading directory: %v", err)
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
	remainingFiles := []string{}
	preservedFiles := make(map[string]bool)

	// Mark files to preserve based on database remarks
	for rows.Next() {
		var studentID, remark string
		if err := rows.Scan(&studentID, &remark); err != nil {
			stats.ErrorCount++
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

		certPath := filepath.Join(generatedFilesDir, studentID+".pdf")

		// If file is older than specified days, preserve it
		if currentTime.Sub(mostRecent).Hours() <= float64(daysOld*24) {
			preservedFiles[certPath] = true
		}
	}

	// Process all files in the directory
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".pdf" {
			filePath := filepath.Join(generatedFilesDir, file.Name())

			// Skip if file is already preserved
			if preservedFiles[filePath] {
				continue
			}

			remainingFiles = append(remainingFiles, filePath)
		}
	}

	// Randomly delete 20% of remaining files
	randomDeletionCount := int(math.Ceil(float64(len(remainingFiles)) * 0.2))
	log.Printf("Total remaining files: %d, Files to delete randomly: %d",
		len(remainingFiles), randomDeletionCount)

	if randomDeletionCount > 0 {
		// Shuffle the remaining files
		rand.Shuffle(len(remainingFiles), func(i, j int) {
			remainingFiles[i], remainingFiles[j] = remainingFiles[j], remainingFiles[i]
		})

		// Delete the first randomDeletionCount files
		for i := 0; i < randomDeletionCount && i < len(remainingFiles); i++ {
			filePath := remainingFiles[i]
			if fileInfo, err := os.Stat(filePath); err == nil {
				// Add file size to bytes freed
				stats.BytesFreed += fileInfo.Size()

				// Delete the file
				if err := os.Remove(filePath); err != nil {
					stats.ErrorCount++
					log.Printf("Error deleting random file %s: %v", filePath, err)
					continue
				}
				stats.FilesDeleted++
				log.Printf("Randomly deleted file: %s", filePath)
			}
		}
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
	// Run cleanup immediately when starting
	if err := CleanupOldFiles(daysOld); err != nil {
		log.Printf("Initial cleanup failed: %v", err)
	}

	go func() {
		for {
			// Wait until next midnight
			now := time.Now()
			next := now.Add(24 * time.Hour)
			next = time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, next.Location())
			duration := next.Sub(now)

			log.Printf("Next cleanup scheduled for: %v (in %v)", next.Format(time.RFC3339), duration)

			time.Sleep(duration)

			// Run cleanup
			if err := CleanupOldFiles(daysOld); err != nil {
				log.Printf("Scheduled cleanup failed: %v", err)
			}
		}
	}()
}
