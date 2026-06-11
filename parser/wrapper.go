package parser

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"speccer/speccer_parser"
)

func ResolveInterpreter(workspaceRoot string) string {
	candidates := []string{
		filepath.Join(workspaceRoot, ".venv", "bin", "python3"),
		filepath.Join(workspaceRoot, "venv", "bin", "python3"),
		filepath.Join(workspaceRoot, "env", "bin", "python3"),
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return "python3"
}

func RunParser(workspaceRoot string, pyFiles []string) (map[string]interface{}, error) {
	interpreter := ResolveInterpreter(workspaceRoot)

	userCache, err := os.UserCacheDir()
	if err != nil {
		userCache = os.TempDir()
	}
	speccerCache := filepath.Join(userCache, "speccer")
	if err := os.MkdirAll(speccerCache, 0755); err != nil {
		return nil, fmt.Errorf("failed to make secure cache dir: %w", err)
	}

	scriptPath := filepath.Join(speccerCache, "parser.py")
	if err := os.WriteFile(scriptPath, speccer_parser.ParserScript, 0644); err != nil {
		return nil, fmt.Errorf("failed to extract embedded parser: %w", err)
	}

	args := append([]string{scriptPath}, pyFiles...)
	cmd := exec.Command(interpreter, args...)
	cmd.Dir = workspaceRoot

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("python execution error: %s, err: %w", stderrBuf.String(), err)
	}

	stdoutStr := stdoutBuf.String()
	startIdx := strings.Index(stdoutStr, "===SPECCER-JSON-START===")
	if startIdx == -1 {
		return nil, fmt.Errorf("unexpected python output. logs: %s", stderrBuf.String())
	}

	jsonPayload := strings.TrimSpace(stdoutStr[startIdx+len("===SPECCER-JSON-START==="):])
	var ir map[string]interface{}
	if err := json.Unmarshal([]byte(jsonPayload), &ir); err != nil {
		return nil, fmt.Errorf("failed to parse json IR: %w", err)
	}

	return ir, nil
}
