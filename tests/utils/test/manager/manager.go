package manager

import (
	"context"
	"os"

	storagev1 "k8s.io/api/storage/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
	kubevirtv1 "kubevirt.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/validator-labs/validatorctl/tests/utils/test/testenv"
)

var (
	KubeVirtCRDs = []string{"crds/kubevirt"}
)

type TestManager interface {
	LoadTestEnv()
	InitEnvironment(input InitEnvironmentInput)
	SaveKubeconfig(path string)
}

type Manager struct {
	*testenv.TestData
	env     Env
	CleanUp func()
}

type Env struct {
	testEnv *envtest.Environment
	config  *rest.Config
	client  client.Client
}

func (m *Manager) LoadTestEnv() {
	// init test data
	testData, _ := testenv.GetTestData()

	m.TestData = testData
}

type InitEnvironmentInput struct {
	Name string
	CRDs []string
}

func (m *Manager) InitEnvironment(input InitEnvironmentInput) error {
	f := false
	testEnv := &envtest.Environment{
		CRDDirectoryPaths:  input.CRDs,
		UseExistingCluster: &f,
	}

	testEnv.ControlPlane.GetAPIServer().Configure().Append("disable-admission-plugins", "NamespaceLifecycle,ServiceAccount")

	if kconf := os.Getenv("KUBECONFIG"); kconf != "" {
		t := true
		testEnv.UseExistingCluster = &t
	}

	//+kubebuilder:scaffold:scheme
	if err := kubevirtv1.AddToScheme(scheme.Scheme); err != nil {
		return err
	}

	cfg, err := testEnv.Start()
	if err != nil {
		return err
	}

	// Add a 'default' test user if not using an existing kubeconfig
	if !*testEnv.UseExistingCluster {
		user := envtest.User{Name: "default", Groups: []string{"system:masters"}}
		if _, err := testEnv.AddUser(user, &rest.Config{}); err != nil {
			return err
		}
	}

	k8sClient, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		return err
	}

	m.env = Env{
		testEnv: testEnv,
		config:  cfg,
		client:  k8sClient,
	}

	return m.CreateK8sResources(k8sClient)
}

func (m *Manager) CreateK8sResources(k8sClient client.Client) error {
	sc := &storagev1.StorageClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "spectro-storage-class",
		},
		Provisioner: "dummy-provisioner",
	}
	return k8sClient.Create(context.TODO(), sc)
}

func (m *Manager) Client() client.Client {
	return m.env.client
}

func (m *Manager) SaveKubeconfig(path string) error {
	servers := make(map[string]*clientcmdapi.Cluster)
	localCluster := &clientcmdapi.Cluster{
		Server:                   m.env.config.Host,
		CertificateAuthorityData: m.env.config.CAData,
	}
	servers["local"] = localCluster

	contextsC := make(map[string]*clientcmdapi.Context)
	localContextC := &clientcmdapi.Context{
		Cluster:  "local",
		AuthInfo: "default",
	}
	contextsC["integration"] = localContextC

	users := make(map[string]*clientcmdapi.AuthInfo)
	users["default"] = &clientcmdapi.AuthInfo{
		ClientCertificateData: m.env.config.CertData,
		ClientKeyData:         m.env.config.KeyData,
	}

	configC := clientcmdapi.Config{
		Kind:           "Config",
		Clusters:       servers,
		Contexts:       contextsC,
		CurrentContext: "integration",
		AuthInfos:      users,
	}

	return clientcmd.WriteToFile(configC, path)
}

func (m *Manager) DestroyEnvironment() error {
	return m.env.testEnv.Stop()
}

func NewTestManager() *Manager {
	return &Manager{
		env: Env{},
	}
}

func (m *Manager) GetEnv(key string) *Env {
	return &m.env
}
