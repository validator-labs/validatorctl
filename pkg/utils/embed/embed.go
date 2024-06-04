package embed

import (
	"bytes"
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"text/tabwriter"
	"text/template"

	"github.com/Masterminds/sprig/v3"

	cfg "github.com/validator-labs/validatorctl/pkg/config"
	log "github.com/validator-labs/validatorctl/pkg/logging"
)

var Docker, Helm, Kind, Kubectl, Vmtoolsd string

//go:embed bin/docker
var docker []byte

//go:embed bin/helm
var helm []byte

//go:embed bin/kind
var kind []byte

//go:embed bin/kubectl
var kubectl []byte

//go:embed bin/vmtoolsd
var vmtoolsd []byte

func InitBinaries(c *cfg.Config) {
	if runtime.GOOS == "windows" {
		Docker = filepath.Join(c.WorkspaceLoc, "bin", "docker.exe")
		Helm = filepath.Join(c.WorkspaceLoc, "bin", "helm.exe")
		Kind = filepath.Join(c.WorkspaceLoc, "bin", "kind.exe")
		Kubectl = filepath.Join(c.WorkspaceLoc, "bin", "kubectl.exe")
	} else {
		Docker = filepath.Join(c.WorkspaceLoc, "bin", "docker")
		Helm = filepath.Join(c.WorkspaceLoc, "bin", "helm")
		Kind = filepath.Join(c.WorkspaceLoc, "bin", "kind")
		Kubectl = filepath.Join(c.WorkspaceLoc, "bin", "kubectl")
	}
	Vmtoolsd = filepath.Join(c.WorkspaceLoc, "bin", "vmtoolsd")

	if _, err := os.Stat(Docker); os.IsNotExist(err) {
		if err := os.WriteFile(Docker, docker, 0755); err != nil /* #nosec G306 */ {
			log.FatalCLI(err.Error())
		}
	}
	if _, err := os.Stat(Helm); os.IsNotExist(err) {
		if err := os.WriteFile(Helm, helm, 0755); err != nil /* #nosec G306 */ {
			log.FatalCLI(err.Error())
		}
	}
	if _, err := os.Stat(Kind); os.IsNotExist(err) {
		if err := os.WriteFile(Kind, kind, 0755); err != nil /* #nosec G306 */ {
			log.FatalCLI(err.Error())
		}
	}
	if _, err := os.Stat(Kubectl); os.IsNotExist(err) {
		if err := os.WriteFile(Kubectl, kubectl, 0755); err != nil /* #nosec G306 */ {
			log.FatalCLI(err.Error())
		}
	}
	if _, err := os.Stat(Vmtoolsd); os.IsNotExist(err) {
		if err := os.WriteFile(Vmtoolsd, vmtoolsd, 0755); err != nil /* #nosec G306 */ {
			log.FatalCLI(err.Error())
		}
	}
}

//go:embed resources/*
var resources embed.FS

// CopyFile copies a file from the embedded file system into a physical disk location.
func CopyFile(destPath, sourceDir, sourceName string) error {
	bytes, err := resources.ReadFile(toEmbeddedFilePath(sourceDir, sourceName))
	if err != nil {
		return err
	}
	if err := os.WriteFile(destPath, bytes, 0600); err != nil {
		return err
	}
	return nil
}

// ReadFile reads a file from the embedded file system.
func ReadFile(dir, filename string) ([]byte, error) {
	return resources.ReadFile(toEmbeddedFilePath(dir, filename))
}

// WriteFile Writes bytes to the specified file.
func WriteFile(outFilename string, data []byte) error {
	if err := os.WriteFile(outFilename, data, 0600); err != nil {
		log.Error("failed to write rendered template to disk: %v", err)
		return err
	}
	return nil
}

// RenderTemplate renders a template from the embedded file system and writes it to disk.
func RenderTemplate(args interface{}, dir, filename, outputPath string) error {
	data, err := RenderTemplateBytes(args, dir, filename)
	if err != nil {
		return err
	}
	if err := WriteFile(outputPath, data); err != nil {
		return err
	}
	return nil
}

// RenderTemplateBytes renders a template from the embedded file system and returns the resulting bytes.
func RenderTemplateBytes(args interface{}, dir, filename string) ([]byte, error) {
	var writer bytes.Buffer
	if err := render(args, &writer, dir, filename); err != nil {
		return nil, err
	}
	return writer.Bytes(), nil
}

// render renders a template from the embedded file system.
func render(args interface{}, writer *bytes.Buffer, dir, filename string) error {
	// Use sprig library functions for templates
	tfm := sprig.TxtFuncMap()
	tmpl, err := template.New(filename).Funcs(tfm).ParseFS(resources, toEmbeddedFilePath(dir, filename))
	if err != nil {
		return err
	}

	if err := tmpl.Option("missingkey=zero").Execute(writer, args); err != nil {
		log.Error("failed to parse template: args: %+v. Error: %v", args, err)
		return err
	}

	return nil
}

// PrintTableTemplate renders a tabular data template and flushes it to stdout.
func PrintTableTemplate(out io.Writer, args interface{}, dir, filename string) error {
	tabw := tabwriter.NewWriter(out, 8, 8, 8, ' ', 0)
	tmpl, err := template.ParseFS(resources, toEmbeddedFilePath(dir, filename))
	if err != nil {
		return err
	}
	if err := tmpl.Execute(tabw, args); err != nil {
		log.Error("failed to parse template: args: %+v. Error: %v", args, err)
		return err
	}
	return tabw.Flush()
}

// toEmbeddedFilePath retrieves the full path of a file within the embedded file system.
// Note that filepath.Join is NOT used here, as embed requires the '/' separator.
func toEmbeddedFilePath(dir, filename string) string {
	return fmt.Sprintf("resources/%s/%s", dir, filename)
}
