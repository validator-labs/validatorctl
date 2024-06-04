package exec

import (
	"bufio"
	"bytes"
	"io"
	"os/exec"
	"strings"

	"github.com/pkg/errors"

	log "github.com/validator-labs/validatorctl/pkg/logging"
)

// Execute enables monkey-patching cmd execution for integration tests
var Execute = execute

func GetCmds(commandStr string) []*exec.Cmd {

	cmdArr := strings.Split(commandStr, "|")
	cmds := make([]*exec.Cmd, 0)

	for _, command := range cmdArr {
		command = strings.TrimSpace(command)
		args := strings.Split(command, " ")
		if len(args) >= 1 {
			bin, args := args[0], args[1:]
			cmds = append(cmds, exec.Command(bin, args...)) //#nosec
		}
	}
	return cmds
}

func StreamingOutput(cmd *exec.Cmd) (string, string, error) {
	var errs string
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", "", err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", "", err
	}

	errScanner := bufio.NewScanner(stderr)
	go func() {
		for errScanner.Scan() {
			errs += errScanner.Text()
			log.Error("%v", errScanner.Text())
		}
	}()

	if err := cmd.Start(); err != nil {
		return "", "", err
	}

	stdoutb, err := io.ReadAll(stdout)
	if err != nil {
		return "", "", err
	}

	stderrb, err := io.ReadAll(stdout)
	if err != nil {
		return "", "", err
	}

	if err := cmd.Wait(); err != nil {
		if errs == "" {
			return "", "", err
		}
		return "", "", errors.WithMessage(err, errs)
	}
	outStr, errStr := string(stdoutb), string(stderrb)

	return outStr, errStr, nil
}

type WriterStringer interface {
	String() string
	Write(p []byte) (n int, err error)
}

// logWriter implements io.Writer while also logging to the terminal
type logWriter struct {
	buffer bytes.Buffer
}

func (l *logWriter) Write(p []byte) (n int, err error) {
	log.InfoCLI(string(p))
	return l.buffer.Write(p)
}

func (l *logWriter) String() string {
	return l.buffer.String()
}

func execute(logStdout bool, stack ...*exec.Cmd) (stdout, stderr string, err error) {
	var stdout_buffer WriterStringer
	if !logStdout {
		stdout_buffer = &bytes.Buffer{}
	} else {
		stdout_buffer = &logWriter{}
	}
	stderr_buffer := logWriter{}

	pipe_stack := make([]*io.PipeWriter, len(stack)-1)
	i := 0
	for ; i < len(stack)-1; i++ {
		stdin_pipe, stdout_pipe := io.Pipe()
		stack[i].Stdout = stdout_pipe
		stack[i].Stderr = &stderr_buffer
		stack[i+1].Stdin = stdin_pipe
		pipe_stack[i] = stdout_pipe
	}
	stack[i].Stdout = stdout_buffer
	stack[i].Stderr = &stderr_buffer

	if err := call(stack, pipe_stack); err != nil {
		return "", stderr_buffer.String(), err
	}
	return stdout_buffer.String(), stderr_buffer.String(), err
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
