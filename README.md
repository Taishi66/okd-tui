# okd-tui

A fast Terminal User Interface for OKD/OpenShift clusters. Browse projects, pods, deployments, events, and stream logs — all from your terminal.

## Installation

### Option A: Go install (recommended)

```bash
go install github.com/Taishi66/okd-tui/cmd@latest
```

This places the `okd-tui` binary in your `$GOPATH/bin` (or `~/go/bin`). Make sure this directory is in your `PATH`.

### Option B: Build from source

```bash
git clone https://github.com/Taishi66/okd-tui.git
cd okd-tui
make install
```

## Prerequisites

Before using okd-tui, you need a valid connection to an OKD/OpenShift cluster.

### 1. Install the OKD CLI (`oc`)

If you don't have it yet, follow the [official install guide](https://docs.okd.io/latest/cli_reference/openshift_cli/getting-started-cli.html).

### 2. Log in to your cluster

```bash
oc login https://api.your-cluster-domain.com:6443
```

Replace with the API URL of your OKD instance. You can find it in your OKD web console under **Help > Command Line Tools**, or ask your cluster administrator.

You will be prompted for your credentials (username/password or token). Once authenticated, `oc` writes the connection info to `~/.kube/config`.

### 3. Verify the connection

```bash
oc whoami     # should print your username
oc project    # should print the active project/namespace
```

## Usage

```bash
okd-tui
```

> **Note:** If your session expires (token timeout), okd-tui displays a reconnection message. Run `oc login` again in another terminal, then press `r` inside the TUI to reconnect.

## Keybindings

### Navigation

| Key | Action |
|-----|--------|
| `j` / `↓` | Move down |
| `k` / `↑` | Move up |
| `g` / `G` | Jump to top / bottom |
| `Ctrl+D` / `Ctrl+U` | Page down / up |
| `Tab` | Next view |
| `1`-`4` | Switch view (Projects, Pods, Deployments, Events) |
| `/` | Filter |
| `t` | Sort column |
| `r` | Refresh |
| `?` | Help |
| `q` | Quit |

### Pod actions

| Key | Action |
|-----|--------|
| `Enter` | View logs |
| `s` | Shell into pod |
| `d` | Delete pod |
| `y` | View YAML |
| `p` | Previous container logs |

### Deployment actions

| Key | Action |
|-----|--------|
| `+` / `-` | Scale up / down |
| `s` | Set replica count |
| `y` | View YAML |

### Log view

| Key | Action |
|-----|--------|
| `w` | Toggle line wrap |
| `p` | Toggle previous logs |

## Configuration

Optional config file at `~/.config/okd-tui/config.yaml`:

```yaml
prod_patterns:
  - prod
  - production
  - prd
  - live

readonly_namespaces: []

cache:
  pods: 5s
  namespaces: 30s
  deployments: 10s
  events: 10s

exec:
  shell: /bin/sh
```

## Development

```bash
make run      # Build and run
make test     # Run tests
make clean    # Remove build artifacts
```
