package mocks

import (
	"os"
)

type CommandExecutor interface {
	Start() error
	Wait() error
}

type MockFileEditor struct {
	FileContents []string
	filename     string
}

func (m *MockFileEditor) Start() error {
	if err := os.WriteFile(m.filename, []byte(m.FileContents[0]), 0600); err != nil {
		return err
	}
	m.FileContents = m.FileContents[1:]
	return nil
}

func (m *MockFileEditor) Wait() error {
	return nil
}

func (m *MockFileEditor) GetCmdExecutor(vimPath string, filename string) CommandExecutor {
	m.filename = filename
	return m
}
