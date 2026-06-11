package main

import (
    "encoding/json"
    "fmt"
    "speccer/ast"
    "speccer/diff"
)

func main() {
    spec := `
# Song Identification Models

| Field | Type | Nullable |
|---|---|---|
| artist | string | false |
| title | string | false |
| enable_discogs | bool | false |
| enable_spotify | bool | false |
| enable_soundcloud | bool | false |
| timeout_per_service | int | false |
| total_timeout | int | false |

# Discogs Schema

| Field | Type | Nullable |
|---|---|---|
| release_id | int | false |
| master_id | int | false |
| title | string | false |
| label | string | false |
| catalog_number | string | false |
| country | string | false |
| year | int | false |
| genres | list[string] | false |
| styles | list[string] | false |
| cover_image_url | string | false |
| formats | list[dict] | false |
| tracklist | list[dict] | false |
`

    chunks, err := ast.ParseSpec([]byte(spec))
    if err != nil {
        fmt.Printf("ERROR parsing spec: %v\n", err)
        return
    }

    // Real IR from track-id-api codebase
    irJSON := `{"SongIdentificationRequest": {"name": "SongIdentificationRequest", "fields": {"artist": {"type": "string", "nullable": false}, "title": {"type": "string", "nullable": false}, "enable_discogs": {"type": "bool", "nullable": false}, "enable_spotify": {"type": "bool", "nullable": false}, "enable_soundcloud": {"type": "bool", "nullable": false}, "enable_gpt_disambiguation": {"type": "bool", "nullable": false}, "force_refresh": {"type": "bool", "nullable": false}, "timeout_per_service": {"type": "int", "nullable": false}, "total_timeout": {"type": "int", "nullable": false}}}, "DiscogsMetadata": {"name": "DiscogsMetadata", "fields": {"release_id": {"type": "int", "nullable": false}, "master_id": {"type": "unresolved_type", "nullable": false}, "title": {"type": "string", "nullable": false}, "label": {"type": "unresolved_type", "nullable": false}, "catalog_number": {"type": "unresolved_type", "nullable": false}, "country": {"type": "unresolved_type", "nullable": false}, "year": {"type": "unresolved_type", "nullable": false}, "genres": {"type": "list[string]", "nullable": false}, "styles": {"type": "list[string]", "nullable": false}, "cover_image_url": {"type": "unresolved_type", "nullable": false}, "formats": {"type": "list[dict]", "nullable": false}, "tracklist": {"type": "list[dict]", "nullable": false}}}}`

    var ir map[string]interface{}
    if err := json.Unmarshal([]byte(irJSON), &ir); err != nil {
        fmt.Printf("ERROR parsing IR JSON: %v\n", err)
        return
    }

    fmt.Println("=== Drift Analysis Results ===")
    for _, chunk := range chunks {
        if chunk.Domain != "data_schema" {
            continue
        }
        fmt.Printf("\n--- %s ---\n", chunk.Heading)
        findings := diff.AnalyzeDrift(chunk, ir)
        if len(findings) == 0 {
            fmt.Println("  ✓ No drift detected")
        } else {
            for _, f := range findings {
                fmt.Printf("  [%s] %s\n", f.Severity, f.Message)
            }
        }
    }
}
