package scaffold

import (
    "os"
    "path/filepath"
    "testing"
)

func TestScaffoldPreviewAndWrite(t *testing.T) {
    tempDir, err := os.MkdirTemp("", "scaffold_test_*")
    if err != nil {
        t.Fatalf("Failed to make temp dir: %v", err)
    }
    defer os.RemoveAll(tempDir)

    targetFile := filepath.Join(tempDir, "models.py")
    err = os.WriteFile(targetFile, []byte("# Hand-written model"), 0644)
    if err != nil {
        t.Fatalf("Failed to write initial file")
    }

    // Run preview first
    preview, err := GenerateStubs("User", "sqlalchemy", false, targetFile)
    if err != nil {
        t.Fatalf("Failed preview: %v", err)
    }
    if !preview.DryRun {
        t.Errorf("Expected dry run true")
    }

    // Run write-to-disk
    preview, err = GenerateStubs("User", "sqlalchemy", true, targetFile)
    if err != nil {
        t.Fatalf("Failed write: %v", err)
    }

    // Verify backup created in .speccer_bak/ directory
    bakFile := filepath.Join(tempDir, ".speccer_bak", "models.py.bak")
    if _, err := os.Stat(bakFile); os.IsNotExist(err) {
        t.Errorf("Expected backup file %s to exist", bakFile)
    }
}

func TestScaffoldPreviewDryRunDoesNotWrite(t *testing.T) {
    tempDir, err := os.MkdirTemp("", "scaffold_dryrun_*")
    if err != nil {
        t.Fatalf("Failed to make temp dir: %v", err)
    }
    defer os.RemoveAll(tempDir)

    targetFile := filepath.Join(tempDir, "models.py")
    err = os.WriteFile(targetFile, []byte("# Original content"), 0644)
    if err != nil {
        t.Fatalf("Failed to write initial file")
    }

    preview, err := GenerateStubs("User", "sqlalchemy", false, targetFile)
    if err != nil {
        t.Fatalf("Failed preview: %v", err)
    }
    if !preview.DryRun {
        t.Errorf("Expected dry run true")
    }
    if preview.Path != targetFile {
        t.Errorf("Expected path %s, got %s", targetFile, preview.Path)
    }
    if preview.Content == "" {
        t.Errorf("Expected non-empty content in preview")
    }

    // Verify original file is untouched
    data, err := os.ReadFile(targetFile)
    if err != nil {
        t.Fatalf("Failed to read original file: %v", err)
    }
    if string(data) != "# Original content" {
        t.Errorf("Expected original content unchanged, got: %s", string(data))
    }
}

func TestScaffoldWriteToDiskNoExistingFile(t *testing.T) {
    tempDir, err := os.MkdirTemp("", "scaffold_newfile_*")
    if err != nil {
        t.Fatalf("Failed to make temp dir: %v", err)
    }
    defer os.RemoveAll(tempDir)

    targetFile := filepath.Join(tempDir, "new_model.py")

    preview, err := GenerateStubs("Product", "sqlalchemy", true, targetFile)
    if err != nil {
        t.Fatalf("Failed write: %v", err)
    }
    if preview.DryRun {
        t.Errorf("Expected dry run false for actual write")
    }

    // Verify file was created
    if _, err := os.Stat(targetFile); os.IsNotExist(err) {
        t.Errorf("Expected file %s to exist", targetFile)
    }

    // Verify no backup was created (no original to backup)
    bakFile := filepath.Join(tempDir, ".speccer_bak", "new_model.py.bak")
    if _, err := os.Stat(bakFile); !os.IsNotExist(err) {
        t.Errorf("Expected no backup for new file, but backup exists")
    }
}

func TestScaffoldContentMatch(t *testing.T) {
    preview, err := GenerateStubs("User", "sqlalchemy", false, "/tmp/dummy.py")
    if err != nil {
        t.Fatalf("Failed preview: %v", err)
    }
    expected := "class User(Base):\n    __tablename__ = 'users'\n    # TODO: Define fields\n"
    if preview.Content != expected {
        t.Errorf("Expected content:\n%s\nGot:\n%s", expected, preview.Content)
    }
}


