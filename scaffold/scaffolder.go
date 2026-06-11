package scaffold

import (
    "fmt"
    "io"
    "os"
    "path/filepath"
    "strings"
)

type ScaffoldPreview struct {
    DryRun  bool   `json:"dry_run"`
    Path    string `json:"path"`
    Content string `json:"content"`
}

func GenerateStubs(modelName, engine string, writeToDisk bool, targetPath string) (*ScaffoldPreview, error) {
    content := fmt.Sprintf(`class %s(Base):
    __tablename__ = '%ss'
    # TODO: Define fields
`, modelName, strings.ToLower(modelName))

    if !writeToDisk {
        return &ScaffoldPreview{
            DryRun:  true,
            Path:    targetPath,
            Content: content,
        }, nil
    }

    // Backup existing file to .speccer_bak/ directory
    if _, err := os.Stat(targetPath); err == nil {
        bakDir := filepath.Join(filepath.Dir(targetPath), ".speccer_bak")
        if err := os.MkdirAll(bakDir, 0755); err != nil {
            return nil, fmt.Errorf("failed to create backup dir: %w", err)
        }
        bakPath := filepath.Join(bakDir, filepath.Base(targetPath)+".bak")
        src, err := os.Open(targetPath)
        if err != nil {
            return nil, err
        }
        defer src.Close()
        dst, err := os.Create(bakPath)
        if err != nil {
            return nil, err
        }
        defer dst.Close()
        if _, err := io.Copy(dst, src); err != nil {
            return nil, err
        }
    }

    if err := os.WriteFile(targetPath, []byte(content), 0644); err != nil {
        return nil, err
    }

    return &ScaffoldPreview{
        DryRun:  false,
        Path:    targetPath,
        Content: content,
    }, nil
}
