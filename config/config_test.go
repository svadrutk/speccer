package config

import (
    "testing"
)

func TestParseConfig(t *testing.T) {
    jsonData := `{
        "database": {
            "engine": "sqlalchemy",
            "source_files": ["app/models/**/*.py"]
        },
        "api": {
            "framework": "fastapi",
            "source_files": ["app/routers/**/*.py"],
            "pydantic_files": ["app/schemas/**/*.py"]
        }
    }`
    cfg, err := Parse([]byte(jsonData))
    if err != nil {
        t.Fatalf("Failed to parse config: %v", err)
    }
    if cfg.Database.Engine != "sqlalchemy" {
        t.Errorf("Expected engine sqlalchemy, got %s", cfg.Database.Engine)
    }
    if len(cfg.Database.SourceFiles) != 1 || cfg.Database.SourceFiles[0] != "app/models/**/*.py" {
        t.Errorf("Source files mismatch")
    }
}
