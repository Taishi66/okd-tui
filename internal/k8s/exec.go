package k8s

import (
	"fmt"
	"os/exec"
)

// LookPathFunc allows overriding exec.LookPath for testing.
var LookPathFunc = exec.LookPath

// BuildExecCmd builds an exec.Cmd to shell into a pod using oc or kubectl.
func (c *Client) BuildExecCmd(namespace, podName, containerName, shell string) (*exec.Cmd, error) {
	tool, err := findExecTool()
	if err != nil {
		return nil, err
	}

	args := []string{"exec", "-it", "-n", namespace, podName}
	if containerName != "" {
		args = append(args, "-c", containerName)
	}
	args = append(args, "--", shell)

	cmd := exec.Command(tool, args...)
	return cmd, nil
}

func findExecTool() (string, error) {
	if path, err := LookPathFunc("oc"); err == nil {
		return path, nil
	}
	if path, err := LookPathFunc("kubectl"); err == nil {
		return path, nil
	}
	return "", fmt.Errorf("ni 'oc' ni 'kubectl' trouvé dans le PATH")
}
