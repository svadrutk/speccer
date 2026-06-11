package diff

import (
    "strings"
    "testing"
    "speccer/ast"
)

func TestDiffSchema(t *testing.T) {
    specChunk := ast.SpecChunk{
        Heading: "Database Models",
        Domain:  "data_schema",
        Content: `| Field | Type | Nullable |
|---|---|---|
| id | uuid | false |
| email | string | false |
`,
    }

    workspaceIR := map[string]interface{}{
        "User": map[string]interface{}{
            "name": "User",
            "fields": map[string]interface{}{
                "id":    map[string]interface{}{"type": "int", "nullable": false},
                "email": map[string]interface{}{"type": "string", "nullable": false},
            },
        },
    }

    findings := AnalyzeDrift(specChunk, workspaceIR)
    if len(findings) == 0 {
        t.Fatalf("Expected drift findings, got none")
    }

    foundTypeMismatch := false
    for _, f := range findings {
        if strings.Contains(f.Message, "type mismatch") && strings.Contains(f.Message, "id") {
            foundTypeMismatch = true
        }
    }
    if !foundTypeMismatch {
        t.Errorf("Expected type mismatch finding for field 'id', got: %v", findings)
    }
}

func TestDiffSchemaNoDrift(t *testing.T) {
    specChunk := ast.SpecChunk{
        Heading: "Database Models",
        Domain:  "data_schema",
        Content: `| Field | Type | Nullable |
|---|---|---|
| id | int | false |
| email | string | false |
`,
    }

    workspaceIR := map[string]interface{}{
        "User": map[string]interface{}{
            "name": "User",
            "fields": map[string]interface{}{
                "id":    map[string]interface{}{"type": "int", "nullable": false},
                "email": map[string]interface{}{"type": "string", "nullable": false},
            },
        },
    }

    findings := AnalyzeDrift(specChunk, workspaceIR)
    if len(findings) != 0 {
        t.Errorf("Expected no findings for matching schema, got: %v", findings)
    }
}

func TestDiffSchemaMissingField(t *testing.T) {
    specChunk := ast.SpecChunk{
        Heading: "Database Models",
        Domain:  "data_schema",
        Content: `| Field | Type | Nullable |
|---|---|---|
| id | uuid | false |
| email | string | false |
`,
    }

    workspaceIR := map[string]interface{}{
        "User": map[string]interface{}{
            "name": "User",
            "fields": map[string]interface{}{
                "id": map[string]interface{}{"type": "int", "nullable": false},
            },
        },
    }

    findings := AnalyzeDrift(specChunk, workspaceIR)
    foundMissing := false
    for _, f := range findings {
        if strings.Contains(f.Message, "missing field") && strings.Contains(f.Message, "email") {
            foundMissing = true
        }
    }
    if !foundMissing {
        t.Errorf("Expected missing field finding for 'email', got: %v", findings)
    }
}

func TestDiffSchemaNullableMismatch(t *testing.T) {
    specChunk := ast.SpecChunk{
        Heading: "Database Models",
        Domain:  "data_schema",
        Content: `| Field | Type | Nullable |
|---|---|---|
| email | string | true |
`,
    }

    workspaceIR := map[string]interface{}{
        "User": map[string]interface{}{
            "name": "User",
            "fields": map[string]interface{}{
                "email": map[string]interface{}{"type": "string", "nullable": false},
            },
        },
    }

    findings := AnalyzeDrift(specChunk, workspaceIR)
    foundNullable := false
    for _, f := range findings {
        if strings.Contains(f.Message, "nullable mismatch") && strings.Contains(f.Message, "email") {
            foundNullable = true
        }
    }
    if !foundNullable {
        t.Errorf("Expected nullable mismatch finding for 'email', got: %v", findings)
    }
}

func TestDiffSchemaNoTable(t *testing.T) {
    specChunk := ast.SpecChunk{
        Heading: "System Overview",
        Domain:  "system_prose",
        Content: "This is just prose with no table.\n",
    }

    workspaceIR := map[string]interface{}{
        "User": map[string]interface{}{
            "name": "User",
            "fields": map[string]interface{}{
                "id": map[string]interface{}{"type": "int", "nullable": false},
            },
        },
    }

    findings := AnalyzeDrift(specChunk, workspaceIR)
    if len(findings) != 0 {
        t.Errorf("Expected no findings for prose-only chunk, got: %v", findings)
    }
}

func TestDiffSchemaEmptySpec(t *testing.T) {
    specChunk := ast.SpecChunk{
        Heading: "Database Models",
        Domain:  "data_schema",
        Content: "",
    }

    workspaceIR := map[string]interface{}{
        "User": map[string]interface{}{
            "name": "User",
            "fields": map[string]interface{}{
                "id": map[string]interface{}{"type": "int", "nullable": false},
            },
        },
    }

    findings := AnalyzeDrift(specChunk, workspaceIR)
    if len(findings) != 0 {
        t.Errorf("Expected no findings for empty spec, got: %v", findings)
    }
}
