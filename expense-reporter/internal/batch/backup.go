package batch

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// BackupManager handles backup operations for the workbook
type BackupManager struct{}

// NewBackupManager creates a new backup manager
func NewBackupManager() *BackupManager {
	return &BackupManager{}
}

// CreateBackup creates a timestamped backup of the workbook file
// Returns the path to the backup file
// Format: <original_name>_backup_YYYYMMDD_HHMMSS.xlsx
func (b *BackupManager) CreateBackup(workbookPath string) (string, error) {
	// Check if source file exists
	if _, err := os.Stat(workbookPath); err != nil {
		return "", fmt.Errorf("source file does not exist: %w", err)
	}

	// Generate backup filename
	backupPath := b.generateBackupPath(workbookPath)

	// Copy file
	err := b.copyFile(workbookPath, backupPath)
	if err != nil {
		return "", fmt.Errorf("failed to create backup: %w", err)
	}

	return backupPath, nil
}

// generateBackupPath creates a backup file path with timestamp
func (b *BackupManager) generateBackupPath(originalPath string) string {
	dir := filepath.Dir(originalPath)
	filename := filepath.Base(originalPath)
	ext := filepath.Ext(filename)
	nameWithoutExt := strings.TrimSuffix(filename, ext)

	// Format: YYYYMMDD_HHMMSS
	timestamp := time.Now().Format("20060102_150405")

	backupName := fmt.Sprintf("%s_backup_%s%s", nameWithoutExt, timestamp, ext)
	return filepath.Join(dir, backupName)
}

// copyFile copies a file from src to dst
func (b *BackupManager) copyFile(src, dst string) error {
	// Open source file
	sourceFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	// Create destination file
	destFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer destFile.Close()

	// Copy content
	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	// Ensure all data is written to disk
	err = destFile.Sync()
	if err != nil {
		return fmt.Errorf("failed to sync destination file: %w", err)
	}

	return nil
}
