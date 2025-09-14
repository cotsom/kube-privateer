package config

import v1 "k8s.io/api/core/v1"

type CmdResult struct {
	Cmd    []string `json:"cmd"`
	Stdout string   `json:"stdout"`
	Stderr string   `json:"stderr"`
	Error  string   `json:"error,omitempty"`
}

type PodSpec struct {
	Name       string
	Image      string
	Privileged bool
	HostPID    bool
	HostPath   string
	ReadOnly   bool
	Caps       []v1.Capability
	Command    []string
	Labels     map[string]string
}
