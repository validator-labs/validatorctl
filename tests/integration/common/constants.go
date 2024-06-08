package common

import (
	"fmt"

	"github.com/validator-labs/validatorctl/tests/utils/test/mocks"
)

const (
	HubblePort = 8443
	ScarPort   = 8444
	MaasPort   = 8445
)

var (
	HubbleHost = fmt.Sprintf("https://127.0.0.1:%d", HubblePort)
	ScarHost   = fmt.Sprintf("https://127.0.0.1:%d", ScarPort)
	MaasHost   = fmt.Sprintf("https://127.0.0.1:%d/MAAS", MaasPort)
)

func NewFileEditor(options ...func(*mocks.MockFileEditor)) *mocks.MockFileEditor {
	editor := &mocks.MockFileEditor{
		FileContents: make([]string, 0),
	}
	for _, o := range options {
		o(editor)
	}
	return editor
}

func WithNoProxy() func(*mocks.MockFileEditor) {
	return func(e *mocks.MockFileEditor) {
		e.FileContents = append(e.FileContents, "127.0.0.1")
	}
}

func WithMirrorRegistries() func(*mocks.MockFileEditor) {
	return func(e *mocks.MockFileEditor) {
		e.FileContents = append(e.FileContents, "docker.io::fake-oci.com/v2//spectro-packs\ngcr.io::fake-oci.com/v2//spectro-packs\nghcr.io::fake-oci.com/v2//spectro-packs\nk8s.gcr.io::fake-oci.com/v2//spectro-packs\nregistry.k8s.io::fake-oci.com/v2//spectro-packs\nquay.io::fake-oci.com/v2//spectro-packs\n*::fake-oci.com/v2//spectro-packs")
	}
}

func WithSSHPublicKey() func(*mocks.MockFileEditor) {
	return func(e *mocks.MockFileEditor) {
		e.FileContents = append(e.FileContents, "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQCiDAxJ6zGg9ai/4CWDj7p9jwm3OAV/q3PTCsY2xMrX4/MuKTe+zuX2FoNyqQGOWAkZHnv/vRQ2vTPadnHpx+mruU7N6LjqGD1z8XujbayAlpQIFczytPJCNqQSGsoBxh6LAW3UJ4Xq+a3apwE8DsV1IkXDAnb6US8yueRsRD7mh+8eMdGtTCQfPfmiFCfaVYR9LEYEuSyeq8rvatYs55s5N+/QB45LrjyHbD070hMQGfkQoGZ6joD1fLF1O5Qm7c3jBg1jRUkRpZO7uJRr9MvT/Z+wl6+xlgQUNxh9+QHWYwjRrTbgZJIvAxhvjGT3CSZjlQkqyu9PT9e1L2lNjEy7 bob@macbook.local")
	}
}
