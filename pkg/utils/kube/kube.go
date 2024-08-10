// Package kube provides functions to interact with Kubernetes clusters
package kube

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"time"

	"golang.org/x/exp/slices"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	log "github.com/validator-labs/validatorctl/pkg/logging"
	exec_utils "github.com/validator-labs/validatorctl/pkg/utils/exec"
)

// KubectlCmd represents a kubectl command
type KubectlCmd struct {
	Cmd      []string
	Delay    *time.Duration
	DelayMsg string
}

// KubectlCommand executes a kubectl command with the given parameters
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

// GetKubeClientset returns a Kubernetes clientset
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

// GetGroupVersion returns a GroupVersion object
func GetGroupVersion(group, version string) schema.GroupVersion {
	return schema.GroupVersion{Group: group, Version: version}
}

// GetCRDClient returns a dynamic client for the given CRD
func GetCRDClient(groupVersion schema.GroupVersion, groupResource schema.GroupResource) (dynamic.NamespaceableResourceInterface, error) {
	dynClient, err := getDynamicClient()
	if err != nil {
		return nil, err
	}

	version := schema.GroupVersionResource{
		Group:    groupVersion.Group,
		Version:  groupVersion.Version,
		Resource: groupResource.Resource,
	}

	return dynClient.Resource(version), nil
}

// GetAPIConfig returns the API configuration from the kubeconfig file
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
	// If an env variable is specified with the config location, use that
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
		cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
		if err != nil {
			return cfg, err
		}
		return cfg, os.Setenv("KUBECONFIG", kubeconfig)
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

// ToUnstructured converts an arbitrary struct to an unstructured object.
func ToUnstructured(obj interface{}) (*unstructured.Unstructured, error) {
	m, err := toMap(obj)
	if err != nil {
		return nil, err
	}
	u := &unstructured.Unstructured{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(m, u); err != nil {
		return nil, fmt.Errorf("failed to convert map to Unstructured: %w", err)
	}
	return u, nil
}

// toMap converts an arbitrary struct to an unstructured map.
func toMap(obj interface{}) (map[string]interface{}, error) {
	out, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to convert obj to unstructured: %w", err)
	}
	return out, nil
}
