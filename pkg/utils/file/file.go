package file

import (
	"archive/tar"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/spectrocloud-labs/prompts-tui/prompts"

	log "github.com/validator-labs/validatorctl/pkg/logging"
	"github.com/validator-labs/validatorctl/tests/utils/test/mocks"
)

var (
	editorBinary = "vi"
	editorPath   = ""
)

func init() {
	visual := os.Getenv("VISUAL")
	editor := os.Getenv("EDITOR")
	if visual != "" {
		editorBinary = visual
		log.InfoCLI("Detected VISUAL env var. Overrode default editor (vi) with %s.", visual)
	} else if editor != "" {
		editorBinary = editor
		log.InfoCLI("Detected EDITOR env var. Overrode default editor (vi) with %s.", editor)
	}
	var err error
	editorPath, err = exec.LookPath(editorBinary)
	if err != nil {
		log.InfoCLI("Error: %s not found on PATH. Either install vi or export VISUAL or EDITOR to an editor of your choosing.", editorBinary)
		os.Exit(1)
	}
}

var (
	GetCmdExecutor = getEditorExecutor
	FileReader     = os.ReadFile
)

func getEditorExecutor(editor, filename string) mocks.CommandExecutor {
	cmd := exec.Command(editor, filename)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

func EditFile(initialContent []byte) ([]byte, error) {
	tmpFile, err := os.CreateTemp(os.TempDir(), "validator")
	if err != nil {
		return nil, err
	}
	filename := tmpFile.Name()
	if err := tmpFile.Close(); err != nil {
		return nil, err
	}

	if initialContent != nil {
		if err := os.WriteFile(filename, initialContent, 0600); err != nil {
			return nil, err
		}
	}

	cmd := GetCmdExecutor(editorPath, filename)
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	if err := cmd.Wait(); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(filename) //#nosec
	if err != nil {
		return nil, err
	}
	if err := os.Remove(filename); err != nil {
		return nil, err
	}
	return data, nil
}

// EditFileValidated prompts a user to edit a file with a predefined prompt, initial content, and separator.
// An optional validation function can be specified to validate the content of each line.
// Entries within the file must be newline-separated. Additionally, a minimum number of entries can be specified.
// The values on each line are joined by the separator and returned to the caller.
func EditFileValidated(prompt, content, separator string, validate func(input string) error, minEntries int) (string, error) {
	if separator == "" {
		return "", errors.New("a non-empty separator is required")
	}

	for {
		var partsBytes []byte
		if content != "" {
			parts := bytes.Split([]byte(content), []byte(separator))
			partsBytes = bytes.Join(parts, []byte("\n"))
		}

		partsBytes, err := EditFile(append([]byte(prompt), partsBytes...))
		if err != nil {
			return content, err
		}
		lines := strings.Split(string(partsBytes), "\n")

		// Parse final lines, skipping comments and optionally validating each line
		finalLines := make([]string, 0)
		for _, l := range lines {
			l = strings.TrimSpace(l)
			if l != "" && !strings.HasPrefix(l, "#") {
				if validate != nil {
					if err = validate(l); err != nil {
						break
					}
				}
				finalLines = append(finalLines, l)
			}
		}

		if err != nil && errors.Is(err, prompts.ValidationError) {
			// for integration tests, return the error
			if os.Getenv("IS_TEST") == "true" {
				return "", err
			}
			// otherwise, we assume the validation function logged
			// a meaningful error message and let the user try again
			time.Sleep(5 * time.Second)
			continue
		}
		if minEntries > 0 && len(finalLines) < minEntries {
			log.InfoCLI("Error editing file: %d or more entries are required", minEntries)
			time.Sleep(5 * time.Second)
			continue
		}

		content = strings.TrimRight(strings.Join(finalLines, separator), separator)
		return content, err
	}
}

func FindFileInTar(r io.Reader, suffix string) ([]byte, error) {
	tarReader := tar.NewReader(r)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		switch header.Typeflag {
		case tar.TypeReg:
			if !strings.HasSuffix(header.Name, suffix) {
				log.Warn("FindFileInTar: skipping file: %s", header.Name)
				continue
			}
			data, err := io.ReadAll(tarReader)
			if err != nil {
				return nil, err
			}
			return data, nil
		default:
			log.Warn("FindFileInTar: ignoring file of type: %v in %s", header.Typeflag, header.Name)
		}
	}
	return nil, fmt.Errorf("FindFileInTar: no file with suffix %s was found", suffix)
}

func ReadFile(filepath string) ([]byte, error) {
	return FileReader(filepath)
}
