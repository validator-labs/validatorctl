package kube

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"regexp"
	"time"

	"golang.org/x/exp/slices"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	//"github.com/spectrocloud/gomi/pkg/k8s" // TODO: see if this is actually needed. maybe i dont need the functions
	log "github.com/validator-labs/validatorctl/pkg/logging"
	embed_utils "github.com/validator-labs/validatorctl/pkg/utils/embed"
	exec_utils "github.com/validator-labs/validatorctl/pkg/utils/exec"
)

type KubectlCmd struct {
	Cmd      []string
	Delay    *time.Duration
	DelayMsg string
}

func KubectlCommand(params []string, kConfig string) (out, stderr string, err error) {
	params = append(params, fmt.Sprintf("--kubeconfig=%s", kConfig))
	cmd := exec.Command(embed_utils.Kubectl, params...) //#nosec

	if slices.Contains(params, "secret") {
		log.InfoCLI("\n==== Kubectl Command ==== Create Secret")
	} else {
		log.InfoCLI("\n==== Kubectl Command ====")
		log.InfoCLI(cmd.String())
	}

	out, stderr, err = exec_utils.Execute(true, cmd)
	return
}

func KubectlDelayCommand(cmd KubectlCmd, kConfig string) (out, stderr string, err error) {
	out, stderr, err = KubectlCommand(cmd.Cmd, kConfig)
	if cmd.Delay != nil {
		log.InfoCLI("waiting %v to %s", cmd.Delay, cmd.DelayMsg)
		time.Sleep(*cmd.Delay)
	}
	return
}

func GetKubeClientset(kubeconfigPath string) (kubernetes.Interface, error) {
	return getClientFromKubeconfig(kubeconfigPath, "")
}

func GetGroupVersion(group, version string) schema.GroupVersion {
	return schema.GroupVersion{Group: group, Version: version}
}

func GetCRDClient(groupVersion schema.GroupVersion, crd Crd) (dynamic.NamespaceableResourceInterface, error) {
	return getCrdClient(groupVersion, crd)
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

type Client interface {
	BuildConfig(kubeconfig string) (*rest.Config, error)
	NewClient(config *rest.Config) (kubernetes.Interface, error)
	IsImported(client kubernetes.Interface) bool
}

type KubeClient struct {
}

func (kc KubeClient) BuildConfig(kubeconfig string) (*rest.Config, error) {
	return clientcmd.BuildConfigFromFlags("", kubeconfig)
}

func (kc KubeClient) NewClient(config *rest.Config) (kubernetes.Interface, error) {
	return kubernetes.NewForConfig(config)
}

func (kc KubeClient) IsImported(client kubernetes.Interface) bool {
	namespacePattern := `cluster-([0-9a-z]{24})`
	namespaceMatch := regexp.MustCompile(namespacePattern)

	namespaces, err := client.CoreV1().Namespaces().List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return false
	}

	var clusterInfo, hubbleInfo *v1.ConfigMap
	var clusterInfoErr, hubbleInfoErr error
	imported := false

	for _, ns := range namespaces.Items {
		if namespaceMatch.MatchString(ns.Name) {

			hubbleInfo, clusterInfoErr = client.CoreV1().ConfigMaps(ns.Name).Get(context.TODO(), "hubble-info", metav1.GetOptions{})
			clusterInfo, hubbleInfoErr = client.CoreV1().ConfigMaps(ns.Name).Get(context.TODO(), "cluster-info", metav1.GetOptions{})

			if hubbleInfo != nil && hubbleInfoErr == nil && clusterInfo != nil && clusterInfoErr == nil {
				log.InfoCLI("Cluster has already been imported as %s at %s.", clusterInfo.Data["clusterName"], hubbleInfo.Data["apiEndpoint"])
				imported = true
			}
		}
	}
	return imported
}

// TODO: -------------------- Everything below is from spectrocloud/gomi/pkg/k8s --------------------

type Crd string

func getCrdClient(groupVersion schema.GroupVersion, crd Crd) (dynamic.NamespaceableResourceInterface, error) {
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

func getClientFromKubeconfig(kubeconfig, masterURL string) (*kubernetes.Clientset, error) {
	config, err := getConfigFromKubeconfig(kubeconfig, masterURL)
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
