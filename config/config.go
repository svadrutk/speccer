package config

import (
    "encoding/json"
    "fmt"
)

type DatabaseConfig struct {
    Engine      string   `json:"engine"`
    SourceFiles []string `json:"source_files"`
}

type APIConfig struct {
    Framework     string   `json:"framework"`
    SourceFiles   []string `json:"source_files"`
    PydanticFiles []string `json:"pydantic_files,omitempty"`
}

type Config struct {
    Database DatabaseConfig `json:"database"`
    API      APIConfig      `json:"api"`
}

func Parse(data []byte) (*Config, error) {
    var cfg Config
    if err := json.Unmarshal(data, &cfg); err != nil {
        return nil, fmt.Errorf("invalid speccer.json: %w", err)
    }
    if cfg.Database.Engine == "" {
        return nil, fmt.Errorf("database.engine is required")
    }
    if cfg.API.Framework == "" {
        return nil, fmt.Errorf("api.framework is required")
    }
    return &cfg, nil
}
