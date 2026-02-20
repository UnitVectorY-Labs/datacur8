package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

func TestValidateCases(t *testing.T) {
	// Locate the test binary relative to this test file.
	_, thisFile, _, _ := runtime.Caller(0)
	repoRoot := filepath.Dir(filepath.Dir(thisFile))
	binary := filepath.Join(repoRoot, "datacur8-test-binary")
	testsDir := filepath.Join(repoRoot, "tests")

	if _, err := os.Stat(binary); os.IsNotExist(err) {
		t.Fatalf("test binary not found at %s", binary)
	}

	entries, err := os.ReadDir(testsDir)
	if err != nil {
		t.Fatalf("cannot read tests dir: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		caseDir := filepath.Join(testsDir, name)

		exitFile := filepath.Join(caseDir, "expected", "validate.exit")
		if _, err := os.Stat(exitFile); os.IsNotExist(err) {
			continue // skip directories without expected exit code
		}

		t.Run(name, func(t *testing.T) {
			// Read expected exit code.
			raw, err := os.ReadFile(exitFile)
			if err != nil {
				t.Fatalf("reading expected exit code: %v", err)
			}
			expectedCode, err := strconv.Atoi(strings.TrimSpace(string(raw)))
			if err != nil {
				t.Fatalf("parsing expected exit code: %v", err)
			}

			// Build command args: validate + any extra args from validate.args.
			args := []string{"validate"}
			argsFile := filepath.Join(caseDir, "expected", "validate.args")
			if argsData, err := os.ReadFile(argsFile); err == nil {
				for _, a := range strings.Fields(strings.TrimSpace(string(argsData))) {
					args = append(args, a)
				}
			}

			cmd := exec.Command(binary, args...)
			cmd.Dir = caseDir
			output, err := cmd.CombinedOutput()

			actualCode := 0
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					actualCode = exitErr.ExitCode()
				} else {
					t.Fatalf("running binary: %v\noutput: %s", err, output)
				}
			}

			if actualCode != expectedCode {
				t.Errorf("exit code = %d, want %d\noutput:\n%s", actualCode, expectedCode, output)
			}
		})
	}
}
