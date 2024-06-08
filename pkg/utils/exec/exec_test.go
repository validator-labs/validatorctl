package exec

import (
	"strings"
	"testing"
)

func TestCmdExec(t *testing.T) {

	kubeOutput := "dGVzdDEyMw=="
	cmdStr := "echo ${out} | base64 -d"
	check := "test123"

	cmdStr = strings.ReplaceAll(cmdStr, "${out}", kubeOutput)
	cmds := GetCmds(cmdStr)
	if cmds == nil {
		t.Fatalf("Command parsing Failed. Please ensure command string is formatted correctly")
	}

	stdout, _, err := Execute(true, cmds...)

	if err != nil {
		t.Fatalf("Command Execution Failed. %v", err)
	}

	if stdout != check {
		t.Fatalf("Invalid Command Output. Expected:%s  Got:%s", check, stdout)
	}

}
