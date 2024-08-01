// Package embed provides utility functions for the embedded file system.
package embed

import (
	"embed"

	"github.com/spectrocloud-labs/embeddedfs/pkg/embeddedfs"
)

//go:embed resources/*
var resources embed.FS

// EFS is validatorctl's embedded file system
var EFS = embeddedfs.NewEmbeddedFS("resources", resources)
