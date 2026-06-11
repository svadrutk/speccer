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

func TestParseBulletModels(t *testing.T) {
	content := `* **` + "`" + `sessions` + "`" + ` Table**:
    * ` + "`" + `playlist_id` + "`" + `: UUID (nullable for single-artist legacy sessions).
    * ` + "`" + `playlist_name` + "`" + `: Text.
    * ` + "`" + `collection_metadata` + "`" + `: JSONB
    * ` + "`" + `source_platform` + "`" + `: Enum
* **` + "`" + `songs` + "`" + ` Table**:
    * **` + "`" + `isrc` + "`" + `**: Text
    * ` + "`" + `genres` + "`" + `: Text Array (fetched from Artist metadata on Spotify).`

	models := parseBulletModels(content)

	if len(models) != 2 {
		t.Fatalf("Expected 2 models, got %d: %v", len(models), models)
	}

	sessions, ok := models["sessions"]
	if !ok {
		t.Fatalf("Expected model 'sessions', got: %v", models)
	}
	if len(sessions) != 4 {
		t.Fatalf("Expected 4 fields in 'sessions', got %d: %v", len(sessions), sessions)
	}

	if sessions[0]["field"] != "playlist_id" || sessions[0]["type"] != "UUID" {
		t.Errorf("Expected playlist_id/UUID, got %v", sessions[0])
	}
	if sessions[0]["nullable"] != "true" {
		t.Errorf("Expected playlist_id nullable=true, got %v", sessions[0]["nullable"])
	}

	songs, ok := models["songs"]
	if !ok {
		t.Fatalf("Expected model 'songs', got: %v", models)
	}
	if len(songs) != 2 {
		t.Fatalf("Expected 2 fields in 'songs', got %d: %v", len(songs), songs)
	}
	if songs[0]["field"] != "isrc" || songs[0]["type"] != "Text" {
		t.Errorf("Expected isrc/Text, got %v", songs[0])
	}
}

func TestAnalyzeDriftWithBullets(t *testing.T) {
	specChunk := ast.SpecChunk{
		Heading: "2. Database Schema Changes (`Supabase`)",
		Domain:  "data_schema",
		Content: `* **` + "`" + `sessions` + "`" + ` Table**:
    * ` + "`" + `playlist_id` + "`" + `: integer
    * ` + "`" + `email` + "`" + `: string
`,
	}

	workspaceIR := map[string]interface{}{
		"sessions": map[string]interface{}{
			"name": "sessions",
			"fields": map[string]interface{}{
				"playlist_id": map[string]interface{}{"type": "uuid", "nullable": false},
				"email":       map[string]interface{}{"type": "string", "nullable": false},
			},
		},
	}

	findings := AnalyzeDrift(specChunk, workspaceIR)
	if len(findings) != 1 {
		t.Fatalf("Expected 1 finding (type mismatch on playlist_id: integer vs uuid), got %d: %v", len(findings), findings)
	}
	if !strings.Contains(findings[0].Message, "type mismatch") {
		t.Errorf("Expected type mismatch finding, got: %v", findings[0].Message)
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
