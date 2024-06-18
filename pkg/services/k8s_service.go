package services

import (
	"context"
	"os"
	"path"
	"regexp"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	prompt_utils "github.com/spectrocloud-labs/prompts-tui/prompts"

	log "github.com/validator-labs/validatorctl/pkg/logging"
	kube_utils "github.com/validator-labs/validatorctl/pkg/utils/kube"
)

func readConfigMap(k8sClient kubernetes.Interface, prompt, namespace string) (string, error) {
	cm, err := prompt_utils.ReadK8sName(prompt, "", true)
	if err != nil {
		return "", err
	}
	if cm != "" {
		if _, err := k8sClient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), cm, metav1.GetOptions{}); err != nil {
			log.InfoCLI("ConfigMap %s does not exist in the %s namespace. Please try again.", cm, namespace)
			return readConfigMap(k8sClient, prompt, namespace)
		}
	}
	return cm, nil
}

func GetSecretsWithKeys(k8sClient kubernetes.Interface, namespace string, keys []string) ([]corev1.Secret, error) {
	secrets := make([]corev1.Secret, 0)
	secretList, err := k8sClient.CoreV1().Secrets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, s := range secretList.Items {
		addSecret := true
		for _, k := range keys {
			if _, ok := s.Data[k]; !ok {
				addSecret = false
				break
			}
		}

		if addSecret {
			secrets = append(secrets, s)
		}
	}
	return secrets, nil
}

func GetSecretsWithRegexKeys(k8sClient kubernetes.Interface, namespace string, keyExpr string) ([]corev1.Secret, error) {
	pattern, err := regexp.Compile(keyExpr)
	if err != nil {
		return nil, err
	}

	secrets := make([]corev1.Secret, 0)
	secretList, err := k8sClient.CoreV1().Secrets(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, s := range secretList.Items {
		for key := range s.Data {
			if pattern.MatchString(key) {
				secrets = append(secrets, s)
				break
			}
		}
	}
	return secrets, nil
}

func ReadSecret(k8sClient kubernetes.Interface, namespace string, optional bool, keys []string) (*corev1.Secret, error) {
	name, err := prompt_utils.ReadK8sName("Secret Name", "", optional)
	if err != nil {
		return nil, err
	}
	var secret *corev1.Secret
	if name != "" {
		secret, err = k8sClient.CoreV1().Secrets(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if err != nil {
			log.InfoCLI("Secret %s does not exist in the %s namespace. Please try again.", name, namespace)
			return ReadSecret(k8sClient, namespace, optional, keys)
		}
		for _, k := range keys {
			if _, ok := secret.Data[k]; !ok {
				log.InfoCLI("Secret %s does not contain required key %s. Please try again.", name, k)
				return ReadSecret(k8sClient, namespace, optional, keys)
			}
		}
	}
	return secret, nil
}

func ReadServiceAccount(k8sClient kubernetes.Interface, namespace string) (string, error) {
	serviceAccount, err := prompt_utils.ReadK8sName("ServiceAccount Name", "", true)
	if err != nil {
		return "", err
	}
	if serviceAccount != "" {
		if _, err := k8sClient.CoreV1().ServiceAccounts(namespace).Get(context.TODO(), serviceAccount, metav1.GetOptions{}); err != nil {
			log.InfoCLI("ServiceAccount %s does not exist in the %s namespace. Please try again.", serviceAccount, namespace)
			return ReadServiceAccount(k8sClient, namespace)
		}
	}
	return serviceAccount, nil
}

func ReadKubeconfig() (kubernetes.Interface, string, error) {
	var err error
	kubeconfigPath := os.Getenv("KUBECONFIG")

	if kubeconfigPath != "" {
		log.InfoCLI("Using active KUBECONFIG: %s", kubeconfigPath)
	} else {
		var defaultKubeConfigPath string
		homeDir, err := os.UserHomeDir()
		if err == nil {
			defaultKubeConfigPath = path.Join(homeDir, ".kube", "config")
		} else {
			log.Warn("unable to determine user home directory path: %v", err)
		}
		kubeconfigPath, err = prompt_utils.ReadFilePath("KUBECONFIG path", defaultKubeConfigPath, "Invalid KUBECONFIG path", false)
		if err != nil {
			return nil, "", err
		}
		if err := os.Setenv("KUBECONFIG", kubeconfigPath); err != nil {
			return nil, "", err
		}
	}

	k8sClient, err := kube_utils.GetKubeClientset(kubeconfigPath)
	if err != nil {
		return nil, "", err
	}
	return k8sClient, kubeconfigPath, nil
}

func readNamespace(k8sClient kubernetes.Interface, prompt, defaultVal string) (string, error) {
	namespace, err := prompt_utils.ReadK8sName(prompt, defaultVal, false)
	if err != nil {
		return "", err
	}
	if _, err := k8sClient.CoreV1().Namespaces().Get(context.TODO(), namespace, metav1.GetOptions{}); err != nil {
		log.InfoCLI("Namespace %s does not exist. Please try again.", namespace)
		return readNamespace(k8sClient, prompt, defaultVal)
	}
	return namespace, nil
}
