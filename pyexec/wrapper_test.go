package pyexec

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractAndLocateInterpreter(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "speccer_test_*")
	if err != nil {
		t.Fatalf("Failed to make temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	venvBin := filepath.Join(tempDir, ".venv", "bin")
	err = os.MkdirAll(venvBin, 0755)
	if err != nil {
		t.Fatalf("Failed to make venv directory structure")
	}

	dummyInterpreter := filepath.Join(venvBin, "python3")
	err = os.WriteFile(dummyInterpreter, []byte("#!/bin/sh\necho 'python'"), 0755)
	if err != nil {
		t.Fatalf("Failed to write dummy interpreter: %v", err)
	}

	interpreterPath := ResolveInterpreter(tempDir)
	if interpreterPath != dummyInterpreter {
		t.Errorf("Expected interpreter %s, got %s", dummyInterpreter, interpreterPath)
	}
}
