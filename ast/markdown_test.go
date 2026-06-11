package ast

import (
    "testing"
)

func TestParseMarkdownAST(t *testing.T) {
    specMD := `
# Database Models
Below are user-related models.

| Field | Type | Nullable |
|---|---|---|
| id | uuid | false |
| email | string | false |

# API Routes
Users API interface.

* GET /api/v1/users
`
    chunks, err := ParseSpec([]byte(specMD))
    if err != nil {
        t.Fatalf("Failed to parse: %v", err)
    }
    if len(chunks) < 2 {
        t.Fatalf("Expected at least 2 associated chunks, got %d", len(chunks))
    }
    if chunks[0].Domain != "data_schema" {
        t.Errorf("Expected first chunk to be data_schema, got %s", chunks[0].Domain)
    }
    if chunks[1].Domain != "api_contract" {
        t.Errorf("Expected second chunk to be api_contract, got %s", chunks[1].Domain)
    }
}
