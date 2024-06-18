package kube

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"time"

	"golang.org/x/exp/slices"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	log "github.com/validator-labs/validatorctl/pkg/logging"
	exec_utils "github.com/validator-labs/validatorctl/pkg/utils/exec"
)

type KubectlCmd struct {
	Cmd      []string
	Delay    *time.Duration
	DelayMsg string
}

type Crd string

func KubectlCommand(params []string, kConfig string) (out, stderr string, err error) {
	params = append(params, fmt.Sprintf("--kubeconfig=%s", kConfig))
	cmd := exec.Command(exec_utils.Kubectl, params...) //#nosec

	if slices.Contains(params, "secret") {
		log.InfoCLI("\n==== Kubectl Command ==== Create Secret")
	} else {
		log.InfoCLI("\n==== Kubectl Command ====")
		log.InfoCLI(cmd.String())
	}

	out, stderr, err = exec_utils.Execute(true, cmd)
	return
}

func GetKubeClientset(kubeconfigPath string) (kubernetes.Interface, error) {
	config, err := getConfigFromKubeconfig(kubeconfigPath, "")
	if err != nil {
		return nil, err
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}

func GetGroupVersion(group, version string) schema.GroupVersion {
	return schema.GroupVersion{Group: group, Version: version}
}

func GetCRDClient(groupVersion schema.GroupVersion, crd Crd) (dynamic.NamespaceableResourceInterface, error) {
	dynClient, err := getDynamicClient()
	if err != nil {
		return nil, err
	}

	version := schema.GroupVersionResource{
		Group:    groupVersion.Group,
		Version:  groupVersion.Version,
		Resource: string(crd),
	}

	return dynClient.Resource(version), nil
}

func GetAPIConfig(kubeconfig string) (*clientcmdapi.Config, error) {
	bytes, err := os.ReadFile(kubeconfig) //#nosec
	if err != nil {
		return nil, err
	}
	clientCfg, err := clientcmd.NewClientConfigFromBytes(bytes)
	if err != nil {
		return nil, err
	}
	apiCfg, err := clientCfg.RawConfig()
	if err != nil {
		return nil, err
	}
	return &apiCfg, nil
}

func getDynamicClient() (dynamic.Interface, error) {
	config, err := getConfig()
	if err != nil {
		return nil, err
	}

	return getDynamicClientForConfig(config)
}

func getConfig() (*rest.Config, error) {
	// If an env variable is specified with the config locaiton, use that
	if len(os.Getenv("KUBECONFIG")) > 0 {
		return clientcmd.BuildConfigFromFlags("", os.Getenv("KUBECONFIG"))
	}
	// If no explicit location, try the in-cluster config
	if c, err := rest.InClusterConfig(); err == nil {
		return c, nil
	}
	// If no in-cluster config, try the default location in the user's home directory
	if usr, err := user.Current(); err == nil {
		if c, err := clientcmd.BuildConfigFromFlags(
			"", filepath.Join(usr.HomeDir, ".kube", "config")); err == nil {
			return c, nil
		}
	}

	return nil, fmt.Errorf("could not locate a kubeconfig")
}

func getDynamicClientForConfig(config *rest.Config) (dynamic.Interface, error) {
	dynClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return dynClient, nil
}

func getConfigFromKubeconfig(kubeconfig, masterURL string) (*rest.Config, error) {
	// If a flag is specified with the config location, use that
	if len(kubeconfig) > 0 {
		return clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	}
	// If an env variable is specified with the config locaiton, use that
	if len(os.Getenv("KUBECONFIG")) > 0 {
		return clientcmd.BuildConfigFromFlags(masterURL, os.Getenv("KUBECONFIG"))
	}
	// If no explicit location, try the in-cluster config
	if c, err := rest.InClusterConfig(); err == nil {
		return c, nil
	}
	// If no in-cluster config, try the default location in the user's home directory
	if usr, err := user.Current(); err == nil {
		if c, err := clientcmd.BuildConfigFromFlags(
			"", filepath.Join(usr.HomeDir, ".kube", "config")); err == nil {
			return c, nil
		}
	}

	return nil, fmt.Errorf("could not locate a kubeconfig")
}
