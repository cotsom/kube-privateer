package app

import (
	"context"
	"encoding/json"
	"fmt"
	config "main/internal/config"
	"os"
	"strings"

	authv1 "k8s.io/api/authorization/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

func GetKubeconfig(kubeconfig string) (*rest.Config, *kubernetes.Clientset, error) {
	if kubeconfig == "" {
		if env := os.Getenv("KUBECONFIG"); env != "" {
			kubeconfig = env
		} else {
			home := os.Getenv("HOME")
			kubeconfig = fmt.Sprintf("%s/.kube/config", home)
		}
	}
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, nil, fmt.Errorf("build kubeconfig from %q: %w", kubeconfig, err)
	}
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("create kubernetes clientset: %w", err)
	}

	return cfg, clientset, nil
}

func GetKubeSecrets(ctx context.Context, clientset *kubernetes.Clientset, results []config.CmdResult) []config.CmdResult {
	secrets, err := clientset.CoreV1().Secrets(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil {
		results = append(results, config.CmdResult{
			Cmd:   []string{"kubectl", "get", "secrets", "--all-namespaces"},
			Error: err.Error(),
		})
		return results
	}

	var lines []string
	for _, s := range secrets.Items {
		keys := len(s.Data)
		lines = append(lines, fmt.Sprintf("NAMESPACE:%s\nNAMESPACE:%s\nTYPE:%s\nKEYS:%d\n", s.Namespace, s.Name, string(s.Type), keys))
	}

	results = append(results, config.CmdResult{
		Cmd:    []string{"kubectl", "get", "secrets", "--all-namespaces", "-o", "wide"},
		Stdout: strings.Join(lines, "\n"),
	})

	return results
}

func KubeCanIList(ctx context.Context, namespace string, clientset *kubernetes.Clientset, results []config.CmdResult) []config.CmdResult {
	ssrr := &authv1.SelfSubjectRulesReview{
		Spec: authv1.SelfSubjectRulesReviewSpec{
			Namespace: namespace,
		},
	}
	res, err := clientset.AuthorizationV1().SelfSubjectRulesReviews().Create(ctx, ssrr, metav1.CreateOptions{})
	if err != nil {
		results = append(results, config.CmdResult{
			Cmd:   []string{"kubectl", "auth", "can-i", "--list"},
			Error: err.Error(),
		})
	} else {
		b, _ := json.MarshalIndent(res.Status, "", "  ")
		results = append(results, config.CmdResult{
			Cmd:    []string{"kubectl", "auth", "can-i", "--list"},
			Stdout: string(b),
		})
	}
	return results
}

func GetKubeRoles(ctx context.Context, namespace string, clientset *kubernetes.Clientset, results []config.CmdResult) []config.CmdResult {
	//Role
	rbs, err := clientset.RbacV1().RoleBindings(metav1.NamespaceAll).List(ctx, metav1.ListOptions{})
	if err != nil {
		results = append(results, config.CmdResult{
			Cmd:   []string{"kubectl", "get", "rolebindings", "-all-namespaces"},
			Error: err.Error(),
		})
	}

	roleRef := ""
	for _, rb := range rbs.Items {
		roleRef = fmt.Sprintf("%s\n%s/%s", roleRef, rb.RoleRef.Kind, rb.RoleRef.Name)
	}
	results = append(results, config.CmdResult{
		Cmd:    []string{"kubectl", "auth", "can-i", "--list"},
		Stdout: roleRef,
	})

	//ClusterRole
	crbs, err := clientset.RbacV1().ClusterRoleBindings().List(ctx, metav1.ListOptions{})
	if err != nil {
		results = append(results, config.CmdResult{
			Cmd:   []string{"kubectl", "get", "clusterrolebindings", "-all-namespaces"},
			Error: err.Error(),
		})
	}

	clusterRoleRef := ""
	for _, crb := range crbs.Items {
		clusterRoleRef = fmt.Sprintf("%s\n%s/%s", clusterRoleRef, crb.RoleRef.Kind, crb.RoleRef.Name)
	}
	results = append(results, config.CmdResult{
		Cmd:    []string{"kubectl", "auth", "can-i", "--list"},
		Stdout: clusterRoleRef,
	})

	return results
}
