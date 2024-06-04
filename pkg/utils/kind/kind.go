package kind

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/pterm/pterm"

	"github.com/spectrocloud-labs/prompts-tui/prompts"

	cfg "github.com/validator-labs/validatorctl/pkg/config"
	log "github.com/validator-labs/validatorctl/pkg/logging"
	models "github.com/validator-labs/validatorctl/pkg/utils/extra"
	//"github.com/spectrocloud/palette-cli/pkg/repo"
	embed_utils "github.com/validator-labs/validatorctl/pkg/utils/embed"
	exec_utils "github.com/validator-labs/validatorctl/pkg/utils/exec"
)

var caCertRegex = regexp.MustCompile("/usr/local/share/ca-certificates")

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

func StartCluster(name, kindConfig, kubeconfig string) error {
	args := []string{
		"create", "cluster", "--name", name,
		"--kubeconfig", kubeconfig, "--config", kindConfig,
	}
	cmd := exec.Command(embed_utils.Kind, args...) //#nosec G204
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

func DeleteCluster(name string) error {
	args := []string{"delete", "cluster", "--name", name}
	cmd := exec.Command(embed_utils.Kind, args...) //#nosec G204
	_, stderr, err := exec_utils.Execute(false, cmd)
	if err != nil {
		return errors.Wrap(err, stderr)
	}
	log.InfoCLI("Deleted local Kind cluster: %s", name)
	return nil
}

func DefaultKindArgs() map[string]interface{} {
	return map[string]interface{}{
		"Env": models.V1Env{
			PodCIDR:        &cfg.DefaultPodCIDR,
			ServiceIPRange: &cfg.DefaultServiceIPRange,
		},
		"Image": fmt.Sprintf("%s:%s", cfg.KindImage, cfg.KindImageTag),
	}
}

func DefaultConfig(kindConfig string) error {
	return embed_utils.RenderTemplate(DefaultKindArgs(), cfg.Kind, cfg.ClusterConfigTemplate, kindConfig)
}

// AdvancedConfig renders a kind cluster configuration file with optional proxy and registry mirror customizations
func AdvancedConfig(env *models.V1Env /*, p *repo.ScarProps*/, kindConfig string) error {
	image := fmt.Sprintf("%s:%s", cfg.KindImage, cfg.KindImageTag)

	clusterConfigArgs := map[string]interface{}{
		"Env":   env,
		"Image": image,
	}

	/*// TODO: commented part out to get it compiling
	// update kind image to pull from registry mirror in airgapped envs
	if p != nil && p.ImageRegistryType != repo.RegistryTypeSpectro {
		image = os.Getenv("KIND_IMAGE")
		if image == "" {
			// For airgap & private registry cases, pull kind image from spectro-images-public/kindest/node
			image = p.ImageUrl(cfg.KindImageInternalRepo, cfg.KindImageTag)
		}
		ep := p.OCIImageRegistry.Endpoint(p.ImageRegistryType)
		basePath := p.OCIImageRegistry.BaseContentPath(p.ImageRegistryType)
		mirrorEndpoint := strings.TrimSuffix(fmt.Sprintf("%s/v2/%s", ep, basePath), "/")
		insecure := p.OCIImageRegistry.InsecureSkipVerify(p.ImageRegistryType)
		username, err := p.OCIImageRegistry.Username(p.ImageRegistryType)
		if err != nil {
			return err
		}
		password, err := p.OCIImageRegistry.Password(p.ImageRegistryType)
		if err != nil {
			return err
		}

		clusterConfigArgs["Image"] = image
		clusterConfigArgs["RegistryBaseContentPath"] = basePath
		clusterConfigArgs["RegistryEndpoint"] = ep
		clusterConfigArgs["RegistryInsecure"] = strconv.FormatBool(insecure)
		clusterConfigArgs["RegistryCaCertName"] = p.OCIImageRegistry.CACertName(p.ImageRegistryType)
		clusterConfigArgs["ReusedProxyCACert"] = p.OCIImageRegistry.ReusedProxyCACert(p.ImageRegistryType)
		clusterConfigArgs["RegistryUsername"] = username
		clusterConfigArgs["RegistryPassword"] = password
		clusterConfigArgs["RegistryMirrorEndpoint"] = mirrorEndpoint
		clusterConfigArgs["RegistryMirrors"] = strings.Split(p.OCIImageRegistry.OCIRegistryBasic.MirrorRegistries, ",")
	}
	*/

	return embed_utils.RenderTemplate(clusterConfigArgs, cfg.Kind, cfg.ClusterConfigTemplate, kindConfig)
}

func getClusters() ([]string, error) {
	cmd := exec.Command(embed_utils.Kind, "get", "clusters") //#nosec G204

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
	cmd := exec.Command(embed_utils.Docker, args...) //#nosec G204
	_, stderr, err := exec_utils.Execute(true, cmd)
	if err != nil {
		return errors.Wrap(err, stderr)
	}
	return nil
}
