package batch

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TDD RED: Test backup functionality

func TestBackupManager_CreateBackup(t *testing.T) {
	tests := []struct {
		name           string
		setupFile      bool   // Whether to create source file
		fileContent    string // Content to write to source file
		wantErr        bool
		wantBackupName string // Expected pattern in backup filename
	}{
		{
			name:           "successful backup of existing file",
			setupFile:      true,
			fileContent:    "test content",
			wantErr:        false,
			wantBackupName: "_backup_",
		},
		{
			name:           "backup filename includes timestamp",
			setupFile:      true,
			fileContent:    "test data",
			wantErr:        false,
			wantBackupName: "_backup_",
		},
		{
			name:        "error when source file does not exist",
			setupFile:   false,
			fileContent: "",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			sourceFile := filepath.Join(tmpDir, "test.xlsx")

			// Setup source file if needed
			if tt.setupFile {
				err := os.WriteFile(sourceFile, []byte(tt.fileContent), 0644)
				if err != nil {
					t.Fatalf("Failed to create test file: %v", err)
				}
			}

			// Create backup
			manager := NewBackupManager()
			backupPath, err := manager.CreateBackup(sourceFile)

			// Check error expectation
			if (err != nil) != tt.wantErr {
				t.Errorf("BackupManager.CreateBackup() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return // Expected error, test passed
			}

			// Verify backup file was created
			if _, statErr := os.Stat(backupPath); statErr != nil {
				t.Errorf("Backup file was not created at %s: %v", backupPath, statErr)
				return
			}

			// Verify backup filename contains expected pattern
			backupName := filepath.Base(backupPath)
			if !strings.Contains(backupName, tt.wantBackupName) {
				t.Errorf("Backup filename %q does not contain %q", backupName, tt.wantBackupName)
			}

			// Verify backup content matches original
			if tt.setupFile {
				backupContent, err := os.ReadFile(backupPath)
				if err != nil {
					t.Fatalf("Failed to read backup file: %v", err)
				}
				if string(backupContent) != tt.fileContent {
					t.Errorf("Backup content = %q, want %q", string(backupContent), tt.fileContent)
				}
			}
		})
	}
}

// Test backup filename format
func TestBackupManager_BackupFilenameFormat(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name         string
		originalFile string
		wantPattern  string // regex-like pattern to match
	}{
		{
			name:         "simple filename",
			originalFile: "test.xlsx",
			wantPattern:  "test_backup_",
		},
		{
			name:         "filename with spaces",
			originalFile: "My Budget 2025.xlsx",
			wantPattern:  "My Budget 2025_backup_",
		},
		{
			name:         "filename with special chars",
			originalFile: "Orçamento 2025.xlsx",
			wantPattern:  "Orçamento 2025_backup_",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sourceFile := filepath.Join(tmpDir, tt.originalFile)

			// Create source file
			err := os.WriteFile(sourceFile, []byte("test"), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			// Create backup
			manager := NewBackupManager()
			backupPath, err := manager.CreateBackup(sourceFile)

			if err != nil {
				t.Fatalf("BackupManager.CreateBackup() error = %v", err)
			}

			backupName := filepath.Base(backupPath)

			// Verify pattern
			if !strings.Contains(backupName, tt.wantPattern) {
				t.Errorf("Backup filename %q does not contain pattern %q", backupName, tt.wantPattern)
			}

			// Verify timestamp format (YYYYMMDD_HHMMSS)
			// Backup name should be like: "test_backup_20250415_143022.xlsx"
			parts := strings.Split(backupName, "_backup_")
			if len(parts) != 2 {
				t.Errorf("Backup filename format incorrect: %q", backupName)
				return
			}

			// Extract timestamp part (without extension)
			timestampPart := strings.TrimSuffix(parts[1], filepath.Ext(backupName))

			// Should be 15 chars: YYYYMMDD_HHMMSS
			if len(timestampPart) != 15 {
				t.Errorf("Timestamp part %q should be 15 characters (YYYYMMDD_HHMMSS)", timestampPart)
			}

			// Should contain underscore separator
			if !strings.Contains(timestampPart, "_") {
				t.Errorf("Timestamp should contain underscore separator: %q", timestampPart)
			}
		})
	}
}

// Test backup is in same directory as original
func TestBackupManager_SameDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	sourceFile := filepath.Join(tmpDir, "test.xlsx")

	err := os.WriteFile(sourceFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	manager := NewBackupManager()
	backupPath, err := manager.CreateBackup(sourceFile)

	if err != nil {
		t.Fatalf("BackupManager.CreateBackup() error = %v", err)
	}

	sourceDir := filepath.Dir(sourceFile)
	backupDir := filepath.Dir(backupPath)

	if sourceDir != backupDir {
		t.Errorf("Backup directory %q should match source directory %q", backupDir, sourceDir)
	}
}

// Test backup timestamp is valid and parseable
func TestBackupManager_TimestampIsRecent(t *testing.T) {
	tmpDir := t.TempDir()
	sourceFile := filepath.Join(tmpDir, "test.xlsx")

	err := os.WriteFile(sourceFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	manager := NewBackupManager()
	backupPath, err := manager.CreateBackup(sourceFile)

	if err != nil {
		t.Fatalf("BackupManager.CreateBackup() error = %v", err)
	}

	backupName := filepath.Base(backupPath)

	// Extract timestamp from filename
	// Format: filename_backup_20250415_143022.xlsx
	parts := strings.Split(backupName, "_backup_")
	if len(parts) != 2 {
		t.Fatalf("Cannot parse backup filename: %q", backupName)
	}

	timestampPart := strings.TrimSuffix(parts[1], filepath.Ext(backupName))

	// Parse timestamp: YYYYMMDD_HHMMSS
	parsedTime, err := time.Parse("20060102_150405", timestampPart)
	if err != nil {
		t.Fatalf("Failed to parse timestamp %q: %v", timestampPart, err)
	}

	// Verify timestamp is from this year (basic sanity check)
	currentYear := time.Now().Year()
	if parsedTime.Year() != currentYear {
		t.Errorf("Backup timestamp year %d should match current year %d", parsedTime.Year(), currentYear)
	}

	// Verify timestamp is not in the future (allowing 1 hour for timezone differences)
	if parsedTime.After(time.Now().Add(1 * time.Hour)) {
		t.Errorf("Backup timestamp %v should not be in the future", parsedTime)
	}

	// Verify timestamp is not too old (within last 24 hours)
	if parsedTime.Before(time.Now().Add(-24 * time.Hour)) {
		t.Errorf("Backup timestamp %v should be recent (within 24 hours)", parsedTime)
	}
}

// Test multiple backups don't overwrite each other
func TestBackupManager_MultipleBackups(t *testing.T) {
	tmpDir := t.TempDir()
	sourceFile := filepath.Join(tmpDir, "test.xlsx")

	err := os.WriteFile(sourceFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	manager := NewBackupManager()

	// Create first backup
	backup1, err := manager.CreateBackup(sourceFile)
	if err != nil {
		t.Fatalf("First backup failed: %v", err)
	}

	// Wait a moment to ensure different timestamp
	time.Sleep(time.Second)

	// Modify source file
	err = os.WriteFile(sourceFile, []byte("modified"), 0644)
	if err != nil {
		t.Fatalf("Failed to modify source file: %v", err)
	}

	// Create second backup
	backup2, err := manager.CreateBackup(sourceFile)
	if err != nil {
		t.Fatalf("Second backup failed: %v", err)
	}

	// Verify both backups exist
	if _, err := os.Stat(backup1); err != nil {
		t.Error("First backup should still exist")
	}

	if _, err := os.Stat(backup2); err != nil {
		t.Error("Second backup should exist")
	}

	// Verify they have different names
	if backup1 == backup2 {
		t.Error("Multiple backups should have different filenames")
	}

	// Verify first backup has original content
	content1, _ := os.ReadFile(backup1)
	if string(content1) != "test" {
		t.Errorf("First backup content = %q, want %q", string(content1), "test")
	}

	// Verify second backup has modified content
	content2, _ := os.ReadFile(backup2)
	if string(content2) != "modified" {
		t.Errorf("Second backup content = %q, want %q", string(content2), "modified")
	}
}
