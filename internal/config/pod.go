package config

import (
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewPod(spec PodSpec) *v1.Pod {
	priv := spec.Privileged
	tp := v1.HostPathDirectory

	volumeMount := []v1.VolumeMount{}
	volume := []v1.Volume{}

	//mount root for privileged
	if spec.HostPath != "" {
		volume = []v1.Volume{
			{
				Name: "host-root",
				VolumeSource: v1.VolumeSource{
					HostPath: &v1.HostPathVolumeSource{
						Path: spec.HostPath,
						Type: &tp,
					},
				},
			},
		}

		volumeMount = []v1.VolumeMount{
			{
				Name:      "host-root",
				MountPath: "/hostroot",
				ReadOnly:  spec.ReadOnly,
			},
		}
	}

	pod := &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:   spec.Name,
			Labels: spec.Labels,
		},
		Spec: v1.PodSpec{
			HostPID:       spec.HostPID,
			RestartPolicy: v1.RestartPolicyNever,
			Containers: []v1.Container{
				{
					Name:    "tester",
					Image:   spec.Image,
					Command: spec.Command,
					SecurityContext: &v1.SecurityContext{
						Privileged: &priv,
						Capabilities: &v1.Capabilities{
							Add: spec.Caps,
						},
					},
					VolumeMounts: volumeMount,
				},
			},
			Volumes: volume,
		},
	}
	if pod.ObjectMeta.Name == "" {
		pod.ObjectMeta.Name = "kube-privateer-" + time.Now().Format("150405")
	}
	return pod
}
