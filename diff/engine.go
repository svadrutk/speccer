package diff

import (
	"fmt"
	"regexp"
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
	}
	return rows
}

func parseBulletModels(content string) map[string][]map[string]string {
	models := make(map[string][]map[string]string)
	// Match: `ModelName` with a word after (e.g. "Table **:"), handles both
	// "**`ModelName` Table**:" (goldmark) and "* **`ModelName` Table**:" (raw)
	modelRe := regexp.MustCompile(`^\s*(?:\*+\s+)?\*\*` + "`" + `(.+?)` + "`" + `?\s+\S+\s*\*\*:`)
	// Match: `field`: TypeWord after stripping bold markers
	fieldRe := regexp.MustCompile("`" + `([^` + "`" + `]+)` + "`" + `\s*:\s*(\S+)`)

	lines := strings.Split(content, "\n")
	currentModel := ""

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if m := modelRe.FindStringSubmatch(line); m != nil {
			currentModel = strings.TrimSpace(m[1])
			if _, exists := models[currentModel]; !exists {
				models[currentModel] = nil
			}
			continue
		}

		cleanLine := strings.ReplaceAll(line, "**", "")
		if m := fieldRe.FindStringSubmatch(cleanLine); m != nil && currentModel != "" {
			nullable := strings.Contains(strings.ToLower(line), "nullable")
			models[currentModel] = append(models[currentModel], map[string]string{
				"field":    m[1],
				"type":     strings.TrimRight(m[2], "(,;"),
				"nullable": fmt.Sprintf("%v", nullable),
			})
		}
	}

	return models
}

func compareFieldToModel(specField map[string]string, modelName string, fieldsRaw map[string]interface{}, section string) []DriftFinding {
	var findings []DriftFinding

	fName := specField["field"]
	specType := specField["type"]
	specNullable := strings.ToLower(specField["nullable"]) == "true"

	if fName == "" {
		return nil
	}

	actual, exists := fieldsRaw[fName]
	if !exists {
		findings = append(findings, DriftFinding{
			Severity: "P1",
			Section:  section + " / " + modelName,
			Message:  fmt.Sprintf("missing field '%s' defined in spec but absent in workspace model '%s'", fName, modelName),
		})
		return findings
	}

	actualMap, ok := actual.(map[string]interface{})
	if !ok {
		return nil
	}

	actualType, _ := actualMap["type"].(string)
	actualNullable, _ := actualMap["nullable"].(bool)

	if strings.ToLower(actualType) != strings.ToLower(specType) {
		findings = append(findings, DriftFinding{
			Severity: "P1",
			Section:  section + " / " + modelName,
			Message:  fmt.Sprintf("field '%s' type mismatch. Spec expects '%s', code uses '%s'", fName, specType, actualType),
		})
	}

	if actualNullable != specNullable {
		findings = append(findings, DriftFinding{
			Severity: "P2",
			Section:  section + " / " + modelName,
			Message:  fmt.Sprintf("field '%s' nullable mismatch. Spec nullable=%v, code nullable=%v", fName, specNullable, actualNullable),
		})
	}

	return findings
}

func AnalyzeDrift(spec ast.SpecChunk, workspace map[string]interface{}) []DriftFinding {
	var findings []DriftFinding

	specFields := parseTableFields(spec.Content)
	if len(specFields) > 0 {
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
				findings = append(findings, compareFieldToModel(specField, modelName, fieldsRaw, spec.Heading)...)
			}
		}
		return findings
	}

	bulletModels := parseBulletModels(spec.Content)
	if len(bulletModels) > 0 {
		for modelName, specFields := range bulletModels {
			modelData, exists := workspace[modelName]
			if !exists {
				findings = append(findings, DriftFinding{
					Severity: "P1",
					Section:  spec.Heading + " / " + modelName,
					Message:  fmt.Sprintf("model '%s' defined in spec but not found in code", modelName),
				})
				continue
			}
			modelMap, ok := modelData.(map[string]interface{})
			if !ok {
				continue
			}
			fieldsRaw, ok := modelMap["fields"].(map[string]interface{})
			if !ok {
				continue
			}
			for _, specField := range specFields {
				findings = append(findings, compareFieldToModel(specField, modelName, fieldsRaw, spec.Heading)...)
			}
		}
	}

	return findings
}
