package tests

import (
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

var binaryPath string

func TestMain(m *testing.M) {
	// Build the binary once for all tests.
	tmp, err := os.MkdirTemp("", "datacur8-test-*")
	if err != nil {
		panic("creating temp dir: " + err.Error())
	}
	defer os.RemoveAll(tmp)

	binaryPath = filepath.Join(tmp, "datacur8")
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = filepath.Join(testsDir(), "..")
	if out, err := cmd.CombinedOutput(); err != nil {
		panic("building binary: " + err.Error() + "\n" + string(out))
	}

	os.Exit(m.Run())
}

func testsDir() string {
	// Use the location of this source file to find tests/.
	// When running via `go test ./tests/`, the working directory is the tests/ package dir.
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	return wd
}

func TestValidate(t *testing.T) {
	root := testsDir()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("reading tests dir: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		caseDir := filepath.Join(root, name)

		exitFile := filepath.Join(caseDir, "expected", "validate.exit")
		if _, err := os.Stat(exitFile); os.IsNotExist(err) {
			continue
		}

		t.Run(name, func(t *testing.T) {
			expectedCode := readExpectedExit(t, exitFile)

			args := []string{"validate"}
			argsFile := filepath.Join(caseDir, "expected", "validate.args")
			if data, err := os.ReadFile(argsFile); err == nil {
				for _, a := range strings.Fields(strings.TrimSpace(string(data))) {
					args = append(args, a)
				}
			}

			cmd := exec.Command(binaryPath, args...)
			cmd.Dir = caseDir
			var stdout, stderr strings.Builder
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()
			actualCode := 0
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					actualCode = exitErr.ExitCode()
				} else {
					t.Fatalf("running binary: %v", err)
				}
			}

			if actualCode != expectedCode {
				t.Errorf("validate exit code = %d, want %d\nstdout:\n%s\nstderr:\n%s",
					actualCode, expectedCode, stdout.String(), stderr.String())
			}

			// Compare stderr lines if expected/validate.stderr exists.
			stderrFile := filepath.Join(caseDir, "expected", "validate.stderr")
			if data, err := os.ReadFile(stderrFile); err == nil {
				compareLines(t, "validate stderr", stderr.String(), string(data))
			}

			// Compare stdout if expected/validate.stdout exists (for JSON format comparison).
			stdoutFile := filepath.Join(caseDir, "expected", "validate.stdout")
			if data, err := os.ReadFile(stdoutFile); err == nil {
				compareJSON(t, "validate stdout", stdout.String(), string(data))
			}
		})
	}
}

func TestExport(t *testing.T) {
	root := testsDir()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("reading tests dir: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		caseDir := filepath.Join(root, name)

		expectedExportDir := filepath.Join(caseDir, "expected", "export")
		if _, err := os.Stat(expectedExportDir); os.IsNotExist(err) {
			continue
		}

		t.Run(name, func(t *testing.T) {
			// Run export in a temp copy to avoid polluting test case dirs.
			tmpDir := t.TempDir()
			copyDir(t, caseDir, tmpDir)

			cmd := exec.Command(binaryPath, "export")
			cmd.Dir = tmpDir
			var stdout, stderr strings.Builder
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					if exitErr.ExitCode() != 0 {
						t.Fatalf("export failed with exit code %d\nstdout:\n%s\nstderr:\n%s",
							exitErr.ExitCode(), stdout.String(), stderr.String())
					}
				} else {
					t.Fatalf("running export: %v", err)
				}
			}

			// Compare all files under expected/export/ with actual outputs.
			err = filepath.WalkDir(expectedExportDir, func(path string, d fs.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return err
				}

				relPath, _ := filepath.Rel(expectedExportDir, path)
				expectedContent, err := os.ReadFile(path)
				if err != nil {
					t.Errorf("reading expected file %s: %v", relPath, err)
					return nil
				}

				actualPath := filepath.Join(tmpDir, relPath)
				actualContent, err := os.ReadFile(actualPath)
				if err != nil {
					t.Errorf("expected export file %s not found in output", relPath)
					return nil
				}

				if string(actualContent) != string(expectedContent) {
					t.Errorf("export file %s differs\n--- expected ---\n%s\n--- actual ---\n%s",
						relPath, string(expectedContent), string(actualContent))
				}

				return nil
			})
			if err != nil {
				t.Errorf("walking expected export dir: %v", err)
			}
		})
	}
}

func TestTidy(t *testing.T) {
	root := testsDir()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("reading tests dir: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		caseDir := filepath.Join(root, name)

		expectedTidyDir := filepath.Join(caseDir, "expected", "tidy")
		if _, err := os.Stat(expectedTidyDir); os.IsNotExist(err) {
			continue
		}

		t.Run(name, func(t *testing.T) {
			// Run tidy in a temp copy.
			tmpDir := t.TempDir()
			copyDir(t, caseDir, tmpDir)

			cmd := exec.Command(binaryPath, "tidy")
			cmd.Dir = tmpDir
			var stdout, stderr strings.Builder
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()
			if err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					if exitErr.ExitCode() != 0 {
						t.Fatalf("tidy failed with exit code %d\nstderr:\n%s",
							exitErr.ExitCode(), stderr.String())
					}
				} else {
					t.Fatalf("running tidy: %v", err)
				}
			}

			// Compare files under expected/tidy/ with the tidied files.
			err = filepath.WalkDir(expectedTidyDir, func(path string, d fs.DirEntry, err error) error {
				if err != nil || d.IsDir() {
					return err
				}

				relPath, _ := filepath.Rel(expectedTidyDir, path)
				expectedContent, err := os.ReadFile(path)
				if err != nil {
					t.Errorf("reading expected tidy file %s: %v", relPath, err)
					return nil
				}

				actualPath := filepath.Join(tmpDir, relPath)
				actualContent, err := os.ReadFile(actualPath)
				if err != nil {
					t.Errorf("expected tidy file %s not found in output", relPath)
					return nil
				}

				if string(actualContent) != string(expectedContent) {
					t.Errorf("tidy file %s differs\n--- expected ---\n%s\n--- actual ---\n%s",
						relPath, string(expectedContent), string(actualContent))
				}

				return nil
			})
			if err != nil {
				t.Errorf("walking expected tidy dir: %v", err)
			}
		})
	}
}

// readExpectedExit reads and parses the expected exit code file.
func readExpectedExit(t *testing.T, path string) int {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading exit file: %v", err)
	}
	code, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		t.Fatalf("parsing exit code from %s: %v", path, err)
	}
	return code
}

// compareLines compares actual and expected output line-by-line, ignoring order.
func compareLines(t *testing.T, label, actual, expected string) {
	t.Helper()
	actualLines := nonEmptyLines(actual)
	expectedLines := nonEmptyLines(expected)

	sort.Strings(actualLines)
	sort.Strings(expectedLines)

	if len(actualLines) != len(expectedLines) {
		t.Errorf("%s: line count differs: got %d, want %d\ngot:\n%s\nwant:\n%s",
			label, len(actualLines), len(expectedLines),
			strings.Join(actualLines, "\n"), strings.Join(expectedLines, "\n"))
		return
	}

	for i := range actualLines {
		if actualLines[i] != expectedLines[i] {
			t.Errorf("%s: line %d differs (sorted)\ngot:  %s\nwant: %s",
				label, i, actualLines[i], expectedLines[i])
		}
	}
}

func nonEmptyLines(s string) []string {
	var lines []string
	for _, line := range strings.Split(s, "\n") {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

// compareJSON compares actual and expected JSON output.
// Both are parsed as JSON and compared structurally.
func compareJSON(t *testing.T, label, actual, expected string) {
	t.Helper()
	var actualJSON, expectedJSON any
	if err := json.Unmarshal([]byte(actual), &actualJSON); err != nil {
		t.Errorf("%s: failed to parse actual JSON: %v\nraw:\n%s", label, err, actual)
		return
	}
	if err := json.Unmarshal([]byte(expected), &expectedJSON); err != nil {
		t.Errorf("%s: failed to parse expected JSON: %v\nraw:\n%s", label, err, expected)
		return
	}

	actualNorm, _ := json.Marshal(actualJSON)
	expectedNorm, _ := json.Marshal(expectedJSON)

	if string(actualNorm) != string(expectedNorm) {
		actualPretty, _ := json.MarshalIndent(actualJSON, "", "  ")
		expectedPretty, _ := json.MarshalIndent(expectedJSON, "", "  ")
		t.Errorf("%s: JSON differs\n--- expected ---\n%s\n--- actual ---\n%s",
			label, string(expectedPretty), string(actualPretty))
	}
}

// copyDir recursively copies src to dst, skipping the expected/ directory.
func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(src, path)

		// Skip the expected/ directory.
		if d.IsDir() && relPath == "expected" {
			return filepath.SkipDir
		}

		destPath := filepath.Join(dst, relPath)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0o755)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(destPath, data, 0o644)
	})
	if err != nil {
		t.Fatalf("copying directory %s to %s: %v", src, dst, err)
	}
}
