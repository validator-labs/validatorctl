// Package exec provides utility functions for executing shell commands.
package exec

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"

	log "github.com/validator-labs/validatorctl/pkg/logging"
)

// Binary represents a binary executable.
type Binary struct {
	name string
	path *string
}

var (
	// Execute enables monkey-patching cmd execution for integration tests.
	Execute = execute

	// Docker references to the docker binary.
	Docker string
	// DockerBin is a Binary struct that references the docker binary.
	DockerBin = Binary{"docker", &Docker}

	// Helm references to the helm binary.
	Helm string
	// HelmBin is a Binary struct that references the helm binary.
	HelmBin = Binary{"helm", &Helm}

	// Kind references to the kind binary.
	Kind string
	// KindBin is a Binary struct that references the kind binary.
	KindBin = Binary{"kind", &Kind}

	// Kubectl references to the kubectl binary.
	Kubectl string
	// KubectlBin is a Binary struct that references the kubectl binary.
	KubectlBin = Binary{"kubectl", &Kubectl}

	// Nslookup references to the nslookup binary.
	Nslookup string
	// NslookupBin is a Binary struct that references the nslookup binary.
	NslookupBin = Binary{"nslookup", &Nslookup}

	// Ping references to the ping binary.
	Ping string
	// PingBin is a Binary struct that references the ping binary.
	PingBin = Binary{"ping", &Ping}
)

// CheckBinaries checks if the required binaries are installed and available on the PATH and returns an error if any are missing.
func CheckBinaries(binaries []Binary) error {
	hasAllBinaries := true
	for _, binary := range binaries {
		path, err := exec.LookPath(binary.name)
		if err != nil {
			hasAllBinaries = false
			log.ErrorCLI(
				fmt.Sprintf("%s is not installed.\nPlease install the missing dependency and ensure it's available on your PATH.", binary.name),
				"PATH", os.Getenv("PATH"),
			)
		}
		*binary.path = path
	}
	if !hasAllBinaries {
		return fmt.Errorf("failed to verify required binaries; one or more binaries are missing")
	}
	return nil
}

// WriterStringer is an interface that combines the io.Writer and fmt.Stringer interfaces
type WriterStringer interface {
	String() string
	Write(p []byte) (n int, err error)
}

// logWriter implements io.Writer while also logging to the terminal
type logWriter struct {
	buffer bytes.Buffer
}

func (l *logWriter) Write(p []byte) (n int, err error) {
	log.InfoCLI("%s", string(p))
	return l.buffer.Write(p)
}

func (l *logWriter) String() string {
	return l.buffer.String()
}

func execute(logStdout bool, stack ...*exec.Cmd) (stdout, stderr string, err error) {
	var stdoutBuffer WriterStringer
	if !logStdout {
		stdoutBuffer = &bytes.Buffer{}
	} else {
		stdoutBuffer = &logWriter{}
	}
	stderrBuffer := logWriter{}

	pipeStack := make([]*io.PipeWriter, len(stack)-1)
	i := 0
	for ; i < len(stack)-1; i++ {
		stdinPipe, stdoutPipe := io.Pipe()
		stack[i].Stdout = stdoutPipe
		stack[i].Stderr = &stderrBuffer
		stack[i+1].Stdin = stdinPipe
		pipeStack[i] = stdoutPipe
	}
	stack[i].Stdout = stdoutBuffer
	stack[i].Stderr = &stderrBuffer

	if err := call(stack, pipeStack); err != nil {
		return "", stderrBuffer.String(), err
	}
	return stdoutBuffer.String(), stderrBuffer.String(), err
}

func call(stack []*exec.Cmd, pipes []*io.PipeWriter) (err error) {

	if stack[0].Process == nil {
		if err = stack[0].Start(); err != nil {
			return err
		}
	}
	if len(stack) > 1 {
		if err = stack[1].Start(); err != nil {
			return err
		}
		defer func() {
			if err == nil {
				err := pipes[0].Close()
				if err != nil {
					log.Error("Error closing pipe: %v", err)
					return
				}
				err = call(stack[1:], pipes[1:])
				if err != nil {
					log.Error("Error calling stack: %v", err)
					return
				}
			} else {
				err = stack[1].Wait()
			}
		}()
	}
	return stack[0].Wait()
}
