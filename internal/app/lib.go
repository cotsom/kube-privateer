package app

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	config "main/internal/config"
	"os"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
)

func ExecCommands(ctx context.Context, clientset *kubernetes.Clientset, image string, commands [][]string, cfg *rest.Config, namespace string, stopper bool) ([]config.CmdResult, error) {
	pod, err := CreatePrivilegedPod(ctx, clientset, image, namespace)
	if err != nil {
		return nil, fmt.Errorf("create pod: %w", err)
	}

	defer func() {
		_ = clientset.CoreV1().Pods(namespace).Delete(context.Background(), pod.Name, metav1.DeleteOptions{})
	}()

	if err := WaitForPodRunning(ctx, clientset, pod.Name, namespace); err != nil {
		return nil, fmt.Errorf("wait pod running: %w", err)
	}

	var results []config.CmdResult

	for _, cmdline := range commands {
		stdout, stderr, err := ExecInPod(cfg, pod.Name, pod.Spec.Containers[0].Name, namespace, cmdline)
		r := config.CmdResult{
			Cmd:    cmdline,
			Stdout: stdout,
			Stderr: stderr,
		}
		if err != nil {
			r.Error = err.Error()
		}
		results = append(results, r)

		if stopper {
			fmt.Println("Command ", r.Cmd, " was executed")
			fmt.Println(r.Stdout, r.Stderr)
			fmt.Println("Press Enter to continue...")
			scanner := bufio.NewScanner(os.Stdin)
			scanner.Scan()
		}
	}

	return results, nil
}

func CreatePrivilegedPod(ctx context.Context, clientset *kubernetes.Clientset, img, ns string) (*v1.Pod, error) {
	priv := true
	hostPID := true
	mountReadOnly := false

	name := fmt.Sprintf("escape-test-%d", time.Now().Unix())

	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				"app": "escape-test",
			},
		},
		Spec: v1.PodSpec{
			Containers: []v1.Container{
				{
					Name:    "tester",
					Image:   img,
					Command: []string{"/bin/sleep", "3600"},
					SecurityContext: &v1.SecurityContext{
						Privileged: &priv,
						Capabilities: &v1.Capabilities{
							Add: []v1.Capability{
								v1.Capability("SYS_ADMIN"),
								v1.Capability("NET_ADMIN"),
								v1.Capability("SYS_PTRACE"),
							},
						},
					},
					VolumeMounts: []v1.VolumeMount{
						{
							Name:      "host-root",
							MountPath: "/hostroot",
							ReadOnly:  mountReadOnly,
						},
					},
				},
			},
			RestartPolicy: v1.RestartPolicyNever,
			HostPID:       hostPID,
			Volumes: []v1.Volume{
				{
					Name: "host-root",
					VolumeSource: v1.VolumeSource{
						HostPath: &v1.HostPathVolumeSource{
							Path: "/",
							Type: newHostPathType(v1.HostPathDirectory),
						},
					},
				},
			},
		},
	}

	return clientset.CoreV1().Pods(ns).Create(ctx, pod, metav1.CreateOptions{})
}

// waitForPodRunning waits until pod is in Running phase or context done.
func WaitForPodRunning(ctx context.Context, clientset *kubernetes.Clientset, podName, ns string) error {
	tick := time.NewTicker(500 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for pod running")
		case <-tick.C:
			p, err := clientset.CoreV1().Pods(ns).Get(context.Background(), podName, metav1.GetOptions{})
			if err != nil {
				// retry
				continue
			}
			if p.Status.Phase == v1.PodRunning {
				return nil
			}
			if p.Status.Phase == v1.PodFailed || p.Status.Phase == v1.PodSucceeded {
				return fmt.Errorf("pod finished with phase: %s", p.Status.Phase)
			}
		}
	}
}

// execInPod execs a single command and returns stdout/stderr.
func ExecInPod(config *restclient.Config, podName, container, namespace string, command []string) (string, string, error) {
	restClient := kubernetes.NewForConfigOrDie(config).CoreV1().RESTClient()
	req := restClient.Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&v1.PodExecOptions{
			Container: container,
			Command:   command,
			Stdin:     false,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		return "", "", fmt.Errorf("NewSPDYExecutor: %w", err)
	}

	var stdout, stderr bytes.Buffer
	err = exec.Stream(remotecommand.StreamOptions{
		Stdout: &stdout,
		Stderr: &stderr,
		Tty:    false,
	})
	return stdout.String(), stderr.String(), err
}

func Stopper(stopper bool, funcName string) {
	if stopper {
		fmt.Println("function ", funcName, " was executed")
		fmt.Println("Press Enter to continue...")
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
	}
}

func newHostPathType(t v1.HostPathType) *v1.HostPathType {
	tp := t
	return &tp
}
