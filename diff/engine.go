package diff

import (
    "fmt"
    "strings"
    "speccer/ast"
)

type DriftFinding struct {
    Severity string `json:"severity"`
    Section  string `json:"section"`
    Message  string `json:"message"`
}

func parseTableFields(content string) []map[string]string {
    lines := strings.Split(content, "\n")
    nonEmpty := make([]string, 0, len(lines))
    for _, l := range lines {
        l = strings.TrimSpace(l)
        if l != "" {
            nonEmpty = append(nonEmpty, l)
        }
    }

    var headers []string
    var rows []map[string]string

    if len(nonEmpty) == 0 {
        return nil
    }

    hasPipes := false
    for _, l := range nonEmpty {
        if strings.HasPrefix(l, "|") {
            hasPipes = true
            break
        }
    }

    if hasPipes {
        for _, line := range nonEmpty {
            if strings.HasPrefix(line, "|---") {
                continue
            }
            if strings.HasPrefix(line, "|") && strings.HasSuffix(line, "|") {
                raw := line[1 : len(line)-1]
                cells := strings.Split(raw, "|")
                trimmed := make([]string, len(cells))
                for i, c := range cells {
                    trimmed[i] = strings.TrimSpace(c)
                }
                if headers == nil {
                    headers = trimmed
                } else {
                    row := make(map[string]string)
                    for i, h := range headers {
                        if i < len(trimmed) {
                            row[strings.ToLower(h)] = trimmed[i]
                        }
                    }
                    if len(row) > 0 {
                        rows = append(rows, row)
                    }
                }
            }
        }
    } else {
        if len(nonEmpty) < 2 {
            return nil
        }
        headers = nonEmpty[:3]
        data := nonEmpty[3:]
        cols := len(headers)
        for i := 0; i+cols <= len(data); i += cols {
            row := make(map[string]string)
            for j, h := range headers {
                row[strings.ToLower(h)] = data[i+j]
            }
            rows = append(rows, row)
        }
    }
    return rows
}

func AnalyzeDrift(spec ast.SpecChunk, workspace map[string]interface{}) []DriftFinding {
    var findings []DriftFinding

    specFields := parseTableFields(spec.Content)
    if len(specFields) == 0 {
        return findings
    }

    for modelName, modelData := range workspace {
        modelMap, ok := modelData.(map[string]interface{})
        if !ok {
            continue
        }
        fieldsRaw, ok := modelMap["fields"].(map[string]interface{})
        if !ok {
            continue
        }

        for _, specField := range specFields {
            fName := specField["field"]
            specType := specField["type"]
            specNullable := strings.ToLower(specField["nullable"]) == "true"

            if fName == "" {
                continue
            }

            actual, exists := fieldsRaw[fName]
            if !exists {
                findings = append(findings, DriftFinding{
                    Severity: "P1",
                    Section:  spec.Heading + " / " + modelName,
                    Message:  fmt.Sprintf("missing field '%s' defined in spec but absent in workspace model '%s'", fName, modelName),
                })
                continue
            }

            actualMap, ok := actual.(map[string]interface{})
            if !ok {
                continue
            }

            actualType, _ := actualMap["type"].(string)
            actualNullable, _ := actualMap["nullable"].(bool)

            if strings.ToLower(actualType) != strings.ToLower(specType) {
                findings = append(findings, DriftFinding{
                    Severity: "P1",
                    Section:  spec.Heading + " / " + modelName,
                    Message:  fmt.Sprintf("field '%s' type mismatch. Spec expects '%s', code uses '%s'", fName, specType, actualType),
                })
            }

            if actualNullable != specNullable {
                findings = append(findings, DriftFinding{
                    Severity: "P2",
                    Section:  spec.Heading + " / " + modelName,
                    Message:  fmt.Sprintf("field '%s' nullable mismatch. Spec nullable=%v, code nullable=%v", fName, specNullable, actualNullable),
                })
            }
        }
    }

    return findings
}
