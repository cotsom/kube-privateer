# Disclaimer
⚠️ NOTICE: This project is not a penetration testing utility.

Kube-Privateer is designed specifically for running controlled test cases to evaluate monitoring tools (such as Tetragon), train Security Operations Centers (SOC), and verify the effectiveness of security alerts in Kubernetes environments.

# Features
* Container Escape Testing: Simulates privileged container escape scenarios
* Recon Testing: Evaluates role-based access control configurations and service account permissions
* TODO: node recon

# Installation
```
go build -o kube-privateer
```

# Usage
The tool provides two main testing modes:

## Escape Mode

The escape command creates a pod and executes various container escape techniques to test monitoring capabilities

`kube-privateer escape [flags]`


**What it tests:**

* System information gathering (id, uname, cgroup analysis)
* Process status examination
* Kernel module loading attempts
* Namespace escaping with nsenter
* Host filesystem mounting
* Core pattern manipulation
* Docker socket access

## Recon Mode
The rbac command tests role-based access control configurations and service account permissions

`kube-privateer recon [flags]`

**What it tests:**

* Volume and mount analysis
* Service account token discovery
* Kubernetes API access validation
* Permission enumeration
* Role and cluster role analysis
* Secrets access testing

## Flags
`--kubeconfig`: Path to kubeconfig file (default: KUBECONFIG env var or $HOME/.kube/config)

`--namespace, -n`: Namespace for test pod (default: "default")

`--image, -i`: Container image to use for test pod (default: "nicolaka/netshoot:latest")

`--timeout, -t`: Overall timeout for the test (default: 3 minutes)

`--stopper, -s`: Wait for user input after each command execution for step-by-step analysis (default: false)

`--privileged`: Create privileged pod with (Privileged = true), (HostPID = true), (HostPath = '/hostroot'), (Caps = 'SYS_ADMIN', 'NET_ADMIN', 'SYS_PTRACE')

## Privileged Pod Creation
The tool creates privileged pods with specific security configurations:

* Privileged container execution
* Host PID namespace access
* Enhanced capabilities (SYS_ADMIN, NET_ADMIN, SYS_PTRACE)
* Host root filesystem mounting at /hostroot

# Example Usage
```bash
# Run container escape tests with step-by-step execution in test namespace
kube-privateer escape --namespace test --stopper  
  
# Run recon tests with custom timeout and privileged pod
kube-privateer recon --timeout 5m --privileged
  
# Run escape tests with custom image  
kube-privateer escape --image custom/security-test:latest
```

# Output Format
All test results are returned in JSON format containing command execution details, including:

Executed commands
Standard output and error
Error information (if any)

## Requirements
* Go 1.24.7 or later
* Access to a Kubernetes cluster
* Appropriate RBAC permissions to create privileged pods
