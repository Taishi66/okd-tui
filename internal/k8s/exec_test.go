package k8s

import (
	"fmt"
	"strings"
	"testing"
)

func TestBuildExecCmd_WithOc(t *testing.T) {
	original := LookPathFunc
	defer func() { LookPathFunc = original }()

	LookPathFunc = func(name string) (string, error) {
		if name == "oc" {
			return "/usr/bin/oc", nil
		}
		return "", fmt.Errorf("not found")
	}

	c := &Client{namespace: "default"}
	cmd, err := c.BuildExecCmd("myns", "web-1", "", "/bin/sh")
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Path != "/usr/bin/oc" {
		t.Errorf("Path = %q, want /usr/bin/oc", cmd.Path)
	}
	args := strings.Join(cmd.Args, " ")
	if !strings.Contains(args, "exec -it -n myns web-1 -- /bin/sh") {
		t.Errorf("Args = %q, missing expected args", args)
	}
}

func TestBuildExecCmd_WithContainer(t *testing.T) {
	original := LookPathFunc
	defer func() { LookPathFunc = original }()

	LookPathFunc = func(name string) (string, error) {
		if name == "kubectl" {
			return "/usr/bin/kubectl", nil
		}
		return "", fmt.Errorf("not found")
	}

	c := &Client{namespace: "default"}
	cmd, err := c.BuildExecCmd("myns", "web-1", "sidecar", "/bin/bash")
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Path != "/usr/bin/kubectl" {
		t.Errorf("Path = %q, want /usr/bin/kubectl", cmd.Path)
	}
	args := strings.Join(cmd.Args, " ")
	if !strings.Contains(args, "-c sidecar") {
		t.Errorf("Args = %q, missing -c sidecar", args)
	}
	if !strings.Contains(args, "-- /bin/bash") {
		t.Errorf("Args = %q, missing -- /bin/bash", args)
	}
}

func TestBuildExecCmd_NoToolAvailable(t *testing.T) {
	original := LookPathFunc
	defer func() { LookPathFunc = original }()

	LookPathFunc = func(string) (string, error) {
		return "", fmt.Errorf("not found")
	}

	c := &Client{namespace: "default"}
	_, err := c.BuildExecCmd("myns", "web-1", "", "/bin/sh")
	if err == nil {
		t.Fatal("expected error when no tool available")
	}
	if !strings.Contains(err.Error(), "oc") || !strings.Contains(err.Error(), "kubectl") {
		t.Errorf("error = %q, should mention oc and kubectl", err.Error())
	}
}

func TestBuildExecCmd_PrefersOcOverKubectl(t *testing.T) {
	original := LookPathFunc
	defer func() { LookPathFunc = original }()

	LookPathFunc = func(name string) (string, error) {
		if name == "oc" {
			return "/usr/bin/oc", nil
		}
		if name == "kubectl" {
			return "/usr/bin/kubectl", nil
		}
		return "", fmt.Errorf("not found")
	}

	c := &Client{namespace: "default"}
	cmd, err := c.BuildExecCmd("myns", "web-1", "", "/bin/sh")
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Path != "/usr/bin/oc" {
		t.Errorf("Path = %q, want /usr/bin/oc (should prefer oc)", cmd.Path)
	}
}
