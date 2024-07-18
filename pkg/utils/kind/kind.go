// Package kind provides functions to manage Kind clusters
package kind

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/pterm/pterm"

	"github.com/spectrocloud-labs/prompts-tui/prompts"

	"github.com/validator-labs/validatorctl/pkg/components"
	cfg "github.com/validator-labs/validatorctl/pkg/config"
	log "github.com/validator-labs/validatorctl/pkg/logging"
	embed_utils "github.com/validator-labs/validatorctl/pkg/utils/embed"
	exec_utils "github.com/validator-labs/validatorctl/pkg/utils/exec"
)

var caCertRegex = regexp.MustCompile("/usr/local/share/ca-certificates")

// ValidateClusters checks for existing Kind clusters and prompts the user to proceed or abort
func ValidateClusters(action string) error {
	if os.Getenv("DISABLE_KIND_CLUSTER_CHECK") != "" {
		return nil
	}
	clusters, err := getClusters()
	if err != nil {
		return err
	}
	if clusters != nil {
		prompt := fmt.Sprintf(
			"Existing kind cluster(s) %s detected. This may cause too many open files errors. Proceed with %s",
			clusters, action,
		)
		ok, err := prompts.ReadBool(prompt, true)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("%s aborted", action)
		}
	}
	return nil
}

// StartCluster creates a new Kind cluster with the given name and configuration
func StartCluster(name, kindConfig, kubeconfig string) error {
	args := []string{
		"create", "cluster", "--name", name,
		"--kubeconfig", kubeconfig, "--config", kindConfig,
	}
	cmd := exec.Command(exec_utils.Kind, args...) //#nosec G204
	_, stderr, err := exec_utils.Execute(true, cmd)
	if err != nil {
		return errors.Wrap(err, stderr)
	}
	update, err := requiresCaCertUpdate(kindConfig)
	if err != nil {
		return errors.Wrap(err, "failed to determine if kind cluster requires CA cert updates")
	}
	if update {
		return updateCaCerts(name)
	}
	return nil
}

// DeleteCluster deletes the Kind cluster with the given name
func DeleteCluster(name string) error {
	args := []string{"delete", "cluster", "--name", name}
	cmd := exec.Command(exec_utils.Kind, args...) //#nosec G204
	_, stderr, err := exec_utils.Execute(false, cmd)
	if err != nil {
		return errors.Wrap(err, stderr)
	}
	log.InfoCLI("Deleted local Kind cluster: %s", name)
	return nil
}

// RenderKindConfig renders a kind cluster configuration file with optional proxy and registry mirror customizations
func RenderKindConfig(env *components.Env, hauler *components.Hauler, kindConfig string) error {
	image := fmt.Sprintf("%s:%s", cfg.KindImage, cfg.KindImageTag)

	clusterConfigArgs := map[string]interface{}{
		"Env":   env,
		"Image": image,
	}

	// air-gapped configuration
	if hauler != nil {
		ep := hauler.Endpoint()
		clusterConfigArgs["Image"] = hauler.KindImage(image)
		clusterConfigArgs["RegistryEndpoint"] = ep
		clusterConfigArgs["RegistryInsecure"] = strconv.FormatBool(hauler.InsecureSkipTLSVerify)
		clusterConfigArgs["RegistryMirrors"] = defaultMirrorRegistries(ep)
		clusterConfigArgs["ReusedProxyCACert"] = hauler.ReuseProxyCACert

		if hauler.CACert != nil {
			clusterConfigArgs["RegistryCACertName"] = hauler.CACert.Name
		}
		if hauler.BasicAuth != nil {
			clusterConfigArgs["RegistryUsername"] = hauler.BasicAuth.Username
			clusterConfigArgs["RegistryPassword"] = hauler.BasicAuth.Password
		}
	}

	return embed_utils.RenderTemplate(clusterConfigArgs, cfg.Kind, cfg.ClusterConfigTemplate, kindConfig)
}

// defaultMirrorRegistries returns a comma-separated string of default registry mirrors
func defaultMirrorRegistries(registryEndpoint string) []string {
	if registryEndpoint == "" {
		return nil
	}
	mirrorRegistries := make([]string, 0)
	for _, registry := range cfg.RegistryMirrors {
		// Add OCI format suffix (/v2)
		registryMirrorEndpoint := fmt.Sprintf("%s/v2", registryEndpoint)
		mirrorRegistries = append(mirrorRegistries,
			fmt.Sprintf("%s%s%s", registry, cfg.RegistryMirrorSeparator, registryMirrorEndpoint),
		)
	}
	return mirrorRegistries
}

func getClusters() ([]string, error) {
	cmd := exec.Command(exec_utils.Kind, "get", "clusters") //#nosec G204

	stdout, stderr, err := exec_utils.Execute(false, cmd)
	if err != nil {
		return nil, errors.Wrap(err, stderr)
	}
	if os.Getenv("IS_TEST") == "true" && len(stdout) > 0 {
		log.HeaderCustom("WARNING: integration tests will fail until you 'export DISABLE_KIND_CLUSTER_CHECK=true' or delete all kind clusters", pterm.BgRed, pterm.FgBlack)
	}
	if len(stdout) > 0 {
		return strings.Split(strings.TrimSpace(stdout), "\n"), nil
	}
	return nil, nil
}

func requiresCaCertUpdate(kindConfig string) (bool, error) {
	bytes, err := os.ReadFile(kindConfig) //#nosec G304
	if err != nil {
		return false, errors.Wrap(err, "failed to read kind cluster configuration file")
	}
	return caCertRegex.Match(bytes), nil
}

func updateCaCerts(name string) error {
	args := []string{
		"exec", fmt.Sprintf("%s-control-plane", name),
		"sh", "-c", "update-ca-certificates && systemctl restart containerd",
	}
	cmd := exec.Command(exec_utils.Docker, args...) //#nosec G204
	_, stderr, err := exec_utils.Execute(true, cmd)
	if err != nil {
		return errors.Wrap(err, stderr)
	}
	return nil
}
