package tui

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Taishi66/okd-tui/internal/config"
	"github.com/Taishi66/okd-tui/internal/domain"
)

// ClientFactory creates a new KubeGateway (used for reconnection from error screen).
type ClientFactory func() (domain.KubeGateway, error)

// --- Views ---

type View int

const (
	ViewProjects    View = iota
	ViewPods
	ViewDeployments
	ViewEvents
	ViewLogs
	ViewYAML
	ViewError // startup error screen
)

func (v View) String() string {
	switch v {
	case ViewProjects:
		return "PROJECTS"
	case ViewPods:
		return "PODS"
	case ViewDeployments:
		return "DEPLOYS"
	case ViewEvents:
		return "EVENTS"
	case ViewLogs:
		return "LOGS"
	case ViewYAML:
		return "YAML"
	default:
		return ""
	}
}

// --- Messages ---

type namespacesLoadedMsg struct{ items []domain.NamespaceInfo }
type podsLoadedMsg struct{ items []domain.PodInfo }
type deploymentsLoadedMsg struct{ items []domain.DeploymentInfo }
type eventsLoadedMsg struct{ items []domain.EventInfo }
type logsLoadedMsg struct{ content string }
type yamlLoadedMsg struct{ content string }
type actionDoneMsg struct{ message string }
type apiErrMsg struct{ err error }
type execDoneMsg struct{ err error }
type watchEventMsg struct{ event domain.WatchEvent }
type watchStoppedMsg struct{ resource string }

// --- Model ---

type Model struct {
	client        domain.KubeGateway
	clientFactory ClientFactory

	// Views
	view     View
	prevView View

	// Data
	namespaces  []domain.NamespaceInfo
	pods        []domain.PodInfo
	deployments []domain.DeploymentInfo
	events      []domain.EventInfo
	logState    logState
	yamlState   yamlViewState

	// UI state
	cursor    int
	width     int
	height    int
	loading   bool
	toast     toast
	confirm   confirmState
	startupErr error // non-nil if launched with NewModelWithError

	// Filter
	filter    textinput.Model
	filtering bool

	// Scale input
	scaleInput    textinput.Model
	scalingDep    string
	scaleActive   bool

	// Container selector (multi-container pods)
	containerSelector bool
	containerChoices  []string
	containerCursor   int
	containerPodName        string
	containerSelectorAction string // "logs" or "exec"

	// Connection state
	disconnected bool

	// Watch state
	watchCancel context.CancelFunc
	watching    bool
	watchCh     <-chan domain.WatchEvent

	// Sort
	sortState map[View]SortState

	// Config
	cfg *config.AppConfig
}

func NewModel(client domain.KubeGateway, factory ClientFactory, cfg *config.AppConfig) Model {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	fi := textinput.New()
	fi.Placeholder = "filtre..."
	fi.CharLimit = 64
	fi.Width = 30

	si := textinput.New()
	si.Placeholder = "nombre de replicas"
	si.CharLimit = 4
	si.Width = 20

	return Model{
		client:        client,
		clientFactory: factory,
		view:          ViewPods,
		filter:        fi,
		scaleInput:    si,
		confirm:       newConfirmState(),
		sortState:     make(map[View]SortState),
		cfg:           cfg,
	}
}

func NewModelWithError(err error, factory ClientFactory) Model {
	return Model{
		view:          ViewError,
		startupErr:    err,
		clientFactory: factory,
		confirm:       newConfirmState(),
	}
}

func (m Model) Init() tea.Cmd {
	if m.view == ViewError {
		return nil
	}
	return m.loadCurrentView()
}

// --- Update ---

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)

	case namespacesLoadedMsg:
		m.namespaces = msg.items
		m.loading = false
		m.cursor = 0
		m.disconnected = false
		return m, nil

	case podsLoadedMsg:
		m.pods = msg.items
		m.loading = false
		m.cursor = 0
		m.disconnected = false
		cmd := m.startWatch()
		return m, cmd

	case deploymentsLoadedMsg:
		m.deployments = msg.items
		m.loading = false
		m.cursor = 0
		m.disconnected = false
		cmd := m.startWatch()
		return m, cmd

	case eventsLoadedMsg:
		m.events = msg.items
		m.loading = false
		m.cursor = 0
		m.disconnected = false
		cmd := m.startWatch()
		return m, cmd

	case watchEventMsg:
		switch msg.event.Resource {
		case "pod":
			m.mergePodEvent(msg.event)
		case "deployment":
			m.mergeDeploymentEvent(msg.event)
		case "event":
			m.mergeEventEvent(msg.event)
		}
		if m.watchCh != nil {
			return m, listenWatch(m.watchCh, msg.event.Resource)
		}
		return m, nil

	case watchStoppedMsg:
		m.watching = false
		cmd := m.startWatch()
		return m, cmd

	case logsLoadedMsg:
		m.logState.setContent(msg.content)
		m.loading = false
		return m, nil

	case yamlLoadedMsg:
		m.yamlState.setContent(msg.content)
		m.loading = false
		return m, nil

	case execDoneMsg:
		if msg.err != nil {
			m.toast = newToast(fmt.Sprintf("Exec: %v", msg.err), toastError)
		} else {
			m.toast = newToast("Shell terminé", toastSuccess)
		}
		return m, tea.Batch(scheduleToastClear(), m.loadCurrentView())

	case actionDoneMsg:
		m.toast = newToast(msg.message, toastSuccess)
		m.loading = false
		return m, tea.Batch(scheduleToastClear(), m.loadCurrentView())

	case apiErrMsg:
		return m.handleAPIError(msg.err)

	case toastExpiredMsg:
		m.toast = toast{}
		return m, nil
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Startup error screen: only q/r
	if m.view == ViewError {
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			if m.clientFactory == nil {
				return m, nil
			}
			newClient, err := m.clientFactory()
			if err != nil {
				m.startupErr = err
				return m, nil
			}
			m.client = newClient
			m.startupErr = nil
			m.view = ViewPods
			return m, m.loadCurrentView()
		}
		return m, nil
	}

	// Confirm dialog captures all input
	if m.confirm.isActive() {
		cmd, handled := m.confirm.update(msg)
		if handled {
			return m, cmd
		}
		return m, nil
	}

	// Container selector captures all input
	if m.containerSelector {
		return m.handleContainerSelector(msg)
	}

	// Scale input captures all input
	if m.scaleActive {
		return m.handleScaleInput(msg)
	}

	// Filter mode
	if m.filtering {
		return m.handleFilterInput(msg)
	}

	// Global keys
	switch {
	case key.Matches(msg, keys.Quit):
		if m.view == ViewLogs {
			m.view = m.prevView
			m.logState = logState{}
			return m, nil
		}
		if m.view == ViewYAML {
			m.view = m.prevView
			m.yamlState = yamlViewState{}
			return m, nil
		}
		m.stopWatch()
		return m, tea.Quit

	case key.Matches(msg, keys.Escape):
		if m.view == ViewLogs {
			m.view = m.prevView
			m.logState = logState{}
			return m, nil
		}
		if m.view == ViewYAML {
			m.view = m.prevView
			m.yamlState = yamlViewState{}
			return m, nil
		}
		m.toast = toast{}
		return m, nil

	// Tab switching
	case key.Matches(msg, keys.Tab1):
		return m.switchView(ViewProjects)
	case key.Matches(msg, keys.Tab2):
		return m.switchView(ViewPods)
	case key.Matches(msg, keys.Tab3):
		return m.switchView(ViewDeployments)
	case key.Matches(msg, keys.Tab4):
		return m.switchView(ViewEvents)
	case key.Matches(msg, keys.TabNext):
		next := (m.view + 1) % 4 // cycle through Projects/Pods/Deployments/Events
		return m.switchView(View(next))

	// Filter
	case key.Matches(msg, keys.Filter):
		if m.view != ViewLogs {
			m.filtering = true
			m.filter.SetValue("")
			m.filter.Focus()
			return m, textinput.Blink
		}

	// Refresh
	case key.Matches(msg, keys.Refresh):
		if m.disconnected && m.client != nil {
			_ = m.client.Reconnect()
			m.disconnected = false
		}
		m.loading = true
		return m, m.loadCurrentView()

	// Navigation
	case key.Matches(msg, keys.Down):
		if m.view == ViewLogs {
			m.logState.scrollDown(1, m.contentHeight())
		} else if m.view == ViewYAML {
			m.yamlState.scrollDown(1, m.contentHeight())
		} else {
			maxIdx := m.listLen() - 1
			if maxIdx < 0 {
				maxIdx = 0
			}
			m.cursor = min(m.cursor+1, maxIdx)
		}
	case key.Matches(msg, keys.Up):
		if m.view == ViewLogs {
			m.logState.scrollUp(1)
		} else if m.view == ViewYAML {
			m.yamlState.scrollUp(1)
		} else {
			m.cursor = max(m.cursor-1, 0)
		}
	case key.Matches(msg, keys.Top):
		if m.view == ViewLogs {
			m.logState.offset = 0
		} else if m.view == ViewYAML {
			m.yamlState.offset = 0
		} else {
			m.cursor = 0
		}
	case key.Matches(msg, keys.Bottom):
		if m.view == ViewLogs {
			m.logState.jumpToBottom(m.contentHeight())
		} else if m.view == ViewYAML {
			m.yamlState.jumpToBottom(m.contentHeight())
		} else {
			m.cursor = max(m.listLen()-1, 0)
		}
	case key.Matches(msg, keys.PageDown):
		if m.view == ViewLogs {
			m.logState.scrollDown(20, m.contentHeight())
		} else if m.view == ViewYAML {
			m.yamlState.scrollDown(20, m.contentHeight())
		} else {
			m.cursor = min(m.cursor+20, max(m.listLen()-1, 0))
		}
	case key.Matches(msg, keys.PageUp):
		if m.view == ViewLogs {
			m.logState.scrollUp(20)
		} else if m.view == ViewYAML {
			m.yamlState.scrollUp(20)
		} else {
			m.cursor = max(m.cursor-20, 0)
		}

	// Enter
	case key.Matches(msg, keys.Enter):
		return m.handleEnter()

	// Actions
	case key.Matches(msg, keys.Delete):
		if m.view == ViewPods {
			return m.handleDeletePod()
		}
	case key.Matches(msg, keys.ScaleUp):
		if m.view == ViewDeployments {
			return m.handleScaleDelta(1)
		}
	case key.Matches(msg, keys.ScaleDn):
		if m.view == ViewDeployments {
			return m.handleScaleDelta(-1)
		}
	case key.Matches(msg, keys.ScaleSet):
		if m.view == ViewDeployments {
			return m.activateScaleInput()
		}
		if m.view == ViewPods {
			return m.handleExecPod()
		}
	case key.Matches(msg, keys.Previous):
		if m.view == ViewLogs {
			return m.togglePreviousLogs()
		}
	case key.Matches(msg, keys.Wrap):
		if m.view == ViewLogs {
			m.logState.wrap = !m.logState.wrap
			return m, nil
		}
	case key.Matches(msg, keys.YAML):
		if m.view == ViewPods || m.view == ViewDeployments {
			return m.handleYAML()
		}
	case key.Matches(msg, keys.Sort):
		if m.view == ViewPods || m.view == ViewDeployments || m.view == ViewEvents {
			return m.cycleSort()
		}
	case key.Matches(msg, keys.Copy):
		if m.view == ViewPods {
			return m.copyPodName()
		}
	}

	return m, nil
}

// --- Key Handlers ---

func (m Model) handleFilterInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter", "esc":
		m.filtering = false
		m.filter.Blur()
		if msg.String() == "esc" {
			m.filter.SetValue("")
		}
		m.cursor = 0
		return m, nil
	default:
		var cmd tea.Cmd
		m.filter, cmd = m.filter.Update(msg)
		m.cursor = 0
		return m, cmd
	}
}

func (m Model) handleScaleInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.scaleActive = false
		m.scaleInput.Blur()
		m.scaleInput.SetValue("")
		return m, nil
	case "enter":
		val := strings.TrimSpace(m.scaleInput.Value())
		replicas, err := strconv.Atoi(val)
		if err != nil || replicas < 0 {
			m.toast = newToast("Nombre invalide", toastError)
			m.scaleActive = false
			m.scaleInput.Blur()
			return m, scheduleToastClear()
		}
		m.scaleActive = false
		m.scaleInput.Blur()
		depName := m.scalingDep
		r := int32(replicas)

		if r > 10 {
			isProd := config.IsProdNamespace(m.client.GetNamespace(), m.cfg.ProdPatterns)
			m.confirm.activate(
				fmt.Sprintf("Scale %s à %d replicas", depName, r),
				depName, m.client.GetNamespace(), isProd,
				func() tea.Msg {
					err := m.client.ScaleDeployment(context.Background(), depName, r)
					if err != nil {
						return apiErrMsg{err}
					}
					return actionDoneMsg{fmt.Sprintf("Scaled %s à %d", depName, r)}
				},
			)
			return m, nil
		}

		m.loading = true
		return m, func() tea.Msg {
			err := m.client.ScaleDeployment(context.Background(), depName, r)
			if err != nil {
				return apiErrMsg{err}
			}
			return actionDoneMsg{fmt.Sprintf("Scaled %s à %d", depName, r)}
		}
	default:
		var cmd tea.Cmd
		m.scaleInput, cmd = m.scaleInput.Update(msg)
		return m, cmd
	}
}

func (m Model) handleEnter() (tea.Model, tea.Cmd) {
	switch m.view {
	case ViewProjects:
		items := m.filteredNamespaces()
		if m.cursor < len(items) {
			m.stopWatch()
			m.client.SetNamespace(items[m.cursor].Name)
			m.filter.SetValue("")
			m.view = ViewPods
			m.loading = true
			return m, m.loadCurrentView()
		}
	case ViewPods:
		items := m.filteredPods()
		if m.cursor < len(items) {
			pod := items[m.cursor]
			if len(pod.Containers) > 1 {
				// Multi-container: show selector
				m.containerPodName = pod.Name
				m.containerChoices = make([]string, len(pod.Containers))
				for i, c := range pod.Containers {
					m.containerChoices[i] = c.Name
				}
				m.containerCursor = 0
				m.containerSelector = true
				m.containerSelectorAction = "logs"
				return m, nil
			}
			return m.openLogsForContainer(pod.Name, "")
		}
	}
	return m, nil
}

func (m Model) handleDeletePod() (tea.Model, tea.Cmd) {
	items := m.filteredPods()
	if m.cursor >= len(items) {
		return m, nil
	}
	podName := items[m.cursor].Name
	isProd := config.IsProdNamespace(m.client.GetNamespace(), m.cfg.ProdPatterns)

	m.confirm.activate("Supprimer pod", podName, m.client.GetNamespace(), isProd, func() tea.Msg {
		err := m.client.DeletePod(context.Background(), podName)
		if err != nil {
			return apiErrMsg{err}
		}
		return actionDoneMsg{fmt.Sprintf("Pod '%s' supprimé", podName)}
	})
	return m, nil
}

func (m Model) handleScaleDelta(delta int32) (tea.Model, tea.Cmd) {
	items := m.filteredDeployments()
	if m.cursor >= len(items) {
		return m, nil
	}
	dep := items[m.cursor]
	newReplicas := dep.Replicas + delta
	if newReplicas < 0 {
		newReplicas = 0
	}
	depName := dep.Name
	m.loading = true
	return m, func() tea.Msg {
		err := m.client.ScaleDeployment(context.Background(), depName, newReplicas)
		if err != nil {
			return apiErrMsg{err}
		}
		return actionDoneMsg{fmt.Sprintf("Scaled %s à %d", depName, newReplicas)}
	}
}

func (m Model) activateScaleInput() (tea.Model, tea.Cmd) {
	items := m.filteredDeployments()
	if m.cursor >= len(items) {
		return m, nil
	}
	m.scalingDep = items[m.cursor].Name
	m.scaleActive = true
	m.scaleInput.SetValue("")
	m.scaleInput.Focus()
	return m, textinput.Blink
}

func (m Model) openLogsForContainer(podName, containerName string) (Model, tea.Cmd) {
	m.prevView = m.view
	m.view = ViewLogs
	m.loading = true
	m.logState = logState{podName: podName, containerName: containerName, wrap: m.logState.wrap}
	return m, func() tea.Msg {
		content, err := m.client.GetPodLogs(context.Background(), podName, containerName, 200, false)
		if err != nil {
			return apiErrMsg{err}
		}
		return logsLoadedMsg{content}
	}
}

func (m Model) handleContainerSelector(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, keys.Escape):
		m.containerSelector = false
		return m, nil
	case key.Matches(msg, keys.Down):
		m.containerCursor = min(m.containerCursor+1, len(m.containerChoices)-1)
		return m, nil
	case key.Matches(msg, keys.Up):
		m.containerCursor = max(m.containerCursor-1, 0)
		return m, nil
	case key.Matches(msg, keys.Enter):
		containerName := m.containerChoices[m.containerCursor]
		m.containerSelector = false
		if m.containerSelectorAction == "exec" {
			return m.startExec(m.containerPodName, containerName)
		}
		return m.openLogsForContainer(m.containerPodName, containerName)
	}
	return m, nil
}

func (m Model) togglePreviousLogs() (tea.Model, tea.Cmd) {
	newPrevious := !m.logState.previous
	podName := m.logState.podName
	containerName := m.logState.containerName
	m.loading = true
	m.logState.previous = newPrevious
	return m, func() tea.Msg {
		content, err := m.client.GetPodLogs(context.Background(), podName, containerName, 200, newPrevious)
		if err != nil {
			return apiErrMsg{err}
		}
		return logsLoadedMsg{content}
	}
}

func (m Model) handleExecPod() (tea.Model, tea.Cmd) {
	items := m.filteredPods()
	if m.cursor >= len(items) {
		return m, nil
	}
	pod := items[m.cursor]

	// Check readonly namespace
	if config.IsReadonlyNamespace(m.client.GetNamespace(), m.cfg.ReadonlyNamespaces) {
		m.toast = newToast("Namespace en lecture seule — exec interdit", toastError)
		return m, scheduleToastClear()
	}

	if len(pod.Containers) > 1 {
		m.containerPodName = pod.Name
		m.containerChoices = make([]string, len(pod.Containers))
		for i, c := range pod.Containers {
			m.containerChoices[i] = c.Name
		}
		m.containerCursor = 0
		m.containerSelector = true
		m.containerSelectorAction = "exec"
		return m, nil
	}
	return m.startExec(pod.Name, "")
}

func (m Model) startExec(podName, containerName string) (Model, tea.Cmd) {
	shell := m.cfg.Exec.Shell
	ns := m.client.GetNamespace()
	cmd, err := m.client.BuildExecCmd(ns, podName, containerName, shell)
	if err != nil {
		m.toast = newToast(fmt.Sprintf("Exec: %v", err), toastError)
		return m, scheduleToastClear()
	}
	return m, tea.ExecProcess(cmd, func(err error) tea.Msg {
		return execDoneMsg{err: err}
	})
}

func (m Model) handleYAML() (tea.Model, tea.Cmd) {
	var resourceName, resourceType string
	switch m.view {
	case ViewPods:
		items := m.filteredPods()
		if m.cursor >= len(items) {
			return m, nil
		}
		resourceName = items[m.cursor].Name
		resourceType = "pod"
	case ViewDeployments:
		items := m.filteredDeployments()
		if m.cursor >= len(items) {
			return m, nil
		}
		resourceName = items[m.cursor].Name
		resourceType = "deployment"
	default:
		return m, nil
	}

	m.prevView = m.view
	m.view = ViewYAML
	m.loading = true
	m.yamlState = yamlViewState{resourceName: resourceName, resourceType: resourceType}

	name := resourceName
	rType := resourceType
	return m, func() tea.Msg {
		var content string
		var err error
		if rType == "pod" {
			content, err = m.client.GetPodYAML(context.Background(), name)
		} else {
			content, err = m.client.GetDeploymentYAML(context.Background(), name)
		}
		if err != nil {
			return apiErrMsg{err}
		}
		return yamlLoadedMsg{content}
	}
}

func (m Model) cycleSort() (tea.Model, tea.Cmd) {
	state := m.sortState[m.view]
	switch m.view {
	case ViewPods:
		state.Column = NextPodSort(state.Column)
	case ViewDeployments:
		state.Column = NextDeploymentSort(state.Column)
	case ViewEvents:
		state.Column = NextEventSort(state.Column)
	}
	state.Ascending = true
	if m.sortState == nil {
		m.sortState = make(map[View]SortState)
	}
	m.sortState[m.view] = state
	m.cursor = 0
	return m, nil
}

func (m Model) copyPodName() (tea.Model, tea.Cmd) {
	items := m.filteredPods()
	if m.cursor >= len(items) {
		return m, nil
	}
	// Copy to clipboard via OSC52 escape sequence (works in most modern terminals)
	podName := items[m.cursor].Name
	m.toast = newToast(fmt.Sprintf("Copié: %s", podName), toastSuccess)
	return m, tea.Batch(
		scheduleToastClear(),
		tea.Printf("\033]52;c;%s\a", encodeBase64(podName)),
	)
}

func (m Model) switchView(v View) (tea.Model, tea.Cmd) {
	if m.view == ViewLogs {
		// from logs, go back first
		m.logState = logState{}
	}
	m.stopWatch()
	m.view = v
	m.cursor = 0
	m.filter.SetValue("")
	m.loading = true
	return m, m.loadCurrentView()
}

// --- Error handling ---

func (m Model) handleAPIError(err error) (tea.Model, tea.Cmd) {
	var apiErr *domain.APIError
	if !errors.As(err, &apiErr) {
		m.toast = newToast(err.Error(), toastError)
		m.loading = false
		return m, scheduleToastClear()
	}

	switch apiErr.Type {
	case domain.ErrTokenExpired:
		m.disconnected = true
		m.toast = newToast(apiErr.Message, toastError)
		m.loading = false
		return m, nil // no auto-clear, keep visible

	case domain.ErrUnreachable:
		m.disconnected = true
		m.toast = newToast("Connexion perdue - données en cache. 'r' pour reconnecter", toastError)
		m.loading = false
		return m, nil

	case domain.ErrForbidden:
		m.toast = newToast(fmt.Sprintf("Accès refusé au namespace '%s'", m.client.GetNamespace()), toastError)
		m.loading = false
		return m, scheduleToastClear()

	case domain.ErrNotFound:
		m.toast = newToast(apiErr.Message, toastError)
		m.loading = false
		return m, tea.Batch(scheduleToastClear(), m.loadCurrentView())

	case domain.ErrConflict:
		m.toast = newToast("Conflit : la ressource a été modifiée. Réessayez.", toastError)
		m.loading = false
		return m, scheduleToastClear()

	case domain.ErrRateLimited:
		m.toast = newToast("Trop de requêtes. Pause 2s...", toastError)
		m.loading = false
		return m, scheduleToastClear()

	default:
		m.toast = newToast(apiErr.Message, toastError)
		m.loading = false
		return m, scheduleToastClear()
	}
}

// --- Data loading ---

func (m Model) loadCurrentView() tea.Cmd {
	switch m.view {
	case ViewProjects:
		return func() tea.Msg {
			items, err := m.client.ListNamespaces(context.Background())
			if err != nil {
				return apiErrMsg{err}
			}
			return namespacesLoadedMsg{items}
		}
	case ViewPods:
		return func() tea.Msg {
			items, err := m.client.ListPods(context.Background())
			if err != nil {
				return apiErrMsg{err}
			}
			return podsLoadedMsg{items}
		}
	case ViewDeployments:
		return func() tea.Msg {
			items, err := m.client.ListDeployments(context.Background())
			if err != nil {
				return apiErrMsg{err}
			}
			return deploymentsLoadedMsg{items}
		}
	case ViewEvents:
		return func() tea.Msg {
			items, err := m.client.ListEvents(context.Background())
			if err != nil {
				return apiErrMsg{err}
			}
			return eventsLoadedMsg{items}
		}
	}
	return nil
}

// --- Watch lifecycle ---

func (m *Model) stopWatch() {
	if m.watchCancel != nil {
		m.watchCancel()
		m.watchCancel = nil
	}
	m.watching = false
	m.watchCh = nil
}

func (m *Model) startWatch() tea.Cmd {
	m.stopWatch()

	if m.client == nil {
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	m.watchCancel = cancel

	var ch <-chan domain.WatchEvent
	var err error
	var resource string

	switch m.view {
	case ViewPods:
		ch, err = m.client.WatchPods(ctx)
		resource = "pod"
	case ViewDeployments:
		ch, err = m.client.WatchDeployments(ctx)
		resource = "deployment"
	case ViewEvents:
		ch, err = m.client.WatchEvents(ctx)
		resource = "event"
	default:
		cancel()
		return nil
	}

	if err != nil || ch == nil {
		cancel()
		return nil
	}

	m.watching = true
	m.watchCh = ch
	return listenWatch(ch, resource)
}

func listenWatch(ch <-chan domain.WatchEvent, resource string) tea.Cmd {
	return func() tea.Msg {
		evt, ok := <-ch
		if !ok {
			return watchStoppedMsg{resource: resource}
		}
		return watchEventMsg{event: evt}
	}
}

// --- Watch merge ---

func (m *Model) mergePodEvent(evt domain.WatchEvent) {
	if evt.Pod == nil {
		return
	}
	switch evt.Type {
	case domain.EventAdded:
		m.pods = append(m.pods, *evt.Pod)
	case domain.EventModified:
		for i, p := range m.pods {
			if p.Name == evt.Pod.Name {
				m.pods[i] = *evt.Pod
				break
			}
		}
	case domain.EventDeleted:
		for i, p := range m.pods {
			if p.Name == evt.Pod.Name {
				m.pods = append(m.pods[:i], m.pods[i+1:]...)
				if m.cursor > 0 && m.cursor >= len(m.pods) {
					m.cursor--
				}
				break
			}
		}
	}
}

func (m *Model) mergeDeploymentEvent(evt domain.WatchEvent) {
	if evt.Deployment == nil {
		return
	}
	switch evt.Type {
	case domain.EventAdded:
		m.deployments = append(m.deployments, *evt.Deployment)
	case domain.EventModified:
		for i, d := range m.deployments {
			if d.Name == evt.Deployment.Name {
				m.deployments[i] = *evt.Deployment
				break
			}
		}
	case domain.EventDeleted:
		for i, d := range m.deployments {
			if d.Name == evt.Deployment.Name {
				m.deployments = append(m.deployments[:i], m.deployments[i+1:]...)
				if m.cursor > 0 && m.cursor >= len(m.deployments) {
					m.cursor--
				}
				break
			}
		}
	}
}

func (m *Model) mergeEventEvent(evt domain.WatchEvent) {
	if evt.Event == nil {
		return
	}
	switch evt.Type {
	case domain.EventAdded:
		m.events = append(m.events, *evt.Event)
	case domain.EventModified:
		for i, e := range m.events {
			if e.Reason == evt.Event.Reason && e.Object == evt.Event.Object {
				m.events[i] = *evt.Event
				break
			}
		}
	case domain.EventDeleted:
		for i, e := range m.events {
			if e.Reason == evt.Event.Reason && e.Object == evt.Event.Object {
				m.events = append(m.events[:i], m.events[i+1:]...)
				if m.cursor > 0 && m.cursor >= len(m.events) {
					m.cursor--
				}
				break
			}
		}
	}
}

// --- Filtering ---

func (m Model) filterText() string {
	return strings.ToLower(m.filter.Value())
}

func (m Model) filteredNamespaces() []domain.NamespaceInfo {
	f := m.filterText()
	if f == "" {
		return m.namespaces
	}
	var result []domain.NamespaceInfo
	for _, ns := range m.namespaces {
		if strings.Contains(strings.ToLower(ns.Name), f) {
			result = append(result, ns)
		}
	}
	return result
}

func (m Model) filteredPods() []domain.PodInfo {
	f := m.filterText()
	var result []domain.PodInfo
	if f == "" {
		result = m.pods
	} else {
		for _, p := range m.pods {
			if strings.Contains(strings.ToLower(p.Name), f) ||
				strings.Contains(strings.ToLower(p.Status), f) {
				result = append(result, p)
			}
		}
	}
	return SortPods(result, m.sortState[ViewPods])
}

func (m Model) filteredDeployments() []domain.DeploymentInfo {
	f := m.filterText()
	var result []domain.DeploymentInfo
	if f == "" {
		result = m.deployments
	} else {
		for _, d := range m.deployments {
			if strings.Contains(strings.ToLower(d.Name), f) {
				result = append(result, d)
			}
		}
	}
	return SortDeployments(result, m.sortState[ViewDeployments])
}

func (m Model) filteredEvents() []domain.EventInfo {
	f := m.filterText()
	var result []domain.EventInfo
	if f == "" {
		result = m.events
	} else {
		for _, e := range m.events {
			if strings.Contains(strings.ToLower(e.Reason), f) ||
				strings.Contains(strings.ToLower(e.Message), f) {
				result = append(result, e)
			}
		}
	}
	return SortEvents(result, m.sortState[ViewEvents])
}

func (m Model) listLen() int {
	switch m.view {
	case ViewProjects:
		return len(m.filteredNamespaces())
	case ViewPods:
		return len(m.filteredPods())
	case ViewDeployments:
		return len(m.filteredDeployments())
	case ViewEvents:
		return len(m.filteredEvents())
	default:
		return 0
	}
}

func (m Model) contentHeight() int {
	// header(1) + tabs(1) + blank(1) + col_header(1) + status_bar(1) = 5 lines overhead
	ch := m.height - 6
	if ch < 1 {
		return 1
	}
	return ch
}

// --- View ---

func (m Model) View() string {
	if m.width == 0 {
		return "Chargement..."
	}

	// Startup error screen
	if m.view == ViewError {
		return m.renderErrorScreen()
	}

	var b strings.Builder

	// Context bar
	b.WriteString(m.renderContextBar())
	b.WriteString("\n")

	// Tabs
	b.WriteString(m.renderTabs())
	b.WriteString("\n")

	// Disconnected banner
	if m.disconnected {
		banner := bannerWarnStyle.Width(m.width).Render("Connexion perdue - données en cache. Appuyez sur 'r' pour reconnecter")
		b.WriteString(banner)
		b.WriteString("\n")
	}

	// Confirm dialog / Container selector
	if m.confirm.isActive() {
		b.WriteString(m.confirm.view(m.width))
	} else if m.containerSelector {
		b.WriteString(renderContainerSelector(m.containerPodName, m.containerChoices, m.containerCursor))
	} else if m.scaleActive {
		b.WriteString(fmt.Sprintf("\n  Scale %s - Replicas: %s\n", m.scalingDep, m.scaleInput.View()))
	} else if m.loading {
		b.WriteString("\n  Chargement...\n")
	} else {
		// Content
		b.WriteString(m.renderContent())
	}

	// Filter bar
	if m.filtering {
		b.WriteString(fmt.Sprintf("  /%s", m.filter.View()))
		b.WriteString("\n")
	}

	// Fill remaining space
	lines := strings.Count(b.String(), "\n")
	for i := lines; i < m.height-2; i++ {
		b.WriteString("\n")
	}

	// Toast
	if m.toast.isActive() {
		b.WriteString(m.toast.render())
		b.WriteString("\n")
	}

	// Status bar
	b.WriteString(m.renderStatusBar())

	return b.String()
}

func (m Model) renderContextBar() string {
	title := titleStyle.Render("OKD TUI")
	if m.client == nil {
		return title
	}
	ctx := contextStyle.Render(m.client.GetContext())
	ns := namespaceStyle.Render(m.client.GetNamespace())
	return fmt.Sprintf(" %s  ctx:%s  ns:%s", title, ctx, ns)
}

func (m Model) renderTabs() string {
	tabs := []struct {
		view  View
		key   string
		label string
	}{
		{ViewProjects, "1", "Projects"},
		{ViewPods, "2", "Pods"},
		{ViewDeployments, "3", "Deploys"},
		{ViewEvents, "4", "Events"},
	}

	var parts []string
	for _, t := range tabs {
		label := fmt.Sprintf("[%s] %s", t.key, t.label)
		if m.view == t.view || (m.view == ViewLogs && m.prevView == t.view) {
			parts = append(parts, tabActiveStyle.Render(label))
		} else {
			parts = append(parts, tabInactiveStyle.Render(label))
		}
	}
	return "  " + strings.Join(parts, "  ")
}

func (m Model) renderContent() string {
	ch := m.contentHeight()
	switch m.view {
	case ViewProjects:
		return renderProjectList(m.filteredNamespaces(), m.cursor, m.width, ch, m.client.GetNamespace())
	case ViewPods:
		return renderPodList(m.filteredPods(), m.cursor, m.width, ch)
	case ViewDeployments:
		return renderDeploymentList(m.filteredDeployments(), m.cursor, m.width, ch)
	case ViewEvents:
		return renderEventList(m.filteredEvents(), m.cursor, m.width, ch)
	case ViewLogs:
		return renderLogs(&m.logState, m.width, ch)
	case ViewYAML:
		return renderYAMLView(&m.yamlState, m.width, ch)
	default:
		return ""
	}
}

func (m Model) renderStatusBar() string {
	var helpText string
	switch m.view {
	case ViewProjects:
		helpText = projectHelpKeys()
	case ViewPods:
		helpText = podHelpKeys()
	case ViewDeployments:
		helpText = deploymentHelpKeys()
	case ViewEvents:
		helpText = eventHelpKeys()
	case ViewLogs:
		helpText = logHelpKeys(m.logState.previous, m.logState.wrap)
	case ViewYAML:
		helpText = yamlHelpKeys()
	}

	nsInfo := ""
	if m.client != nil {
		nsInfo = m.client.GetNamespace()
	}

	liveIndicator := ""
	if m.watching {
		liveIndicator = liveStyle.Render(" ● LIVE")
	}
	var itemInfo string
	switch m.view {
	case ViewLogs:
		itemInfo = fmt.Sprintf("%d lignes", len(m.logState.lines))
	case ViewYAML:
		itemInfo = fmt.Sprintf("%d lignes", len(m.yamlState.lines))
	default:
		itemInfo = fmt.Sprintf("%d items", m.listLen())
	}
	left := fmt.Sprintf(" %s | %s | %s%s", m.view.String(), nsInfo, itemInfo, liveIndicator)
	bar := statusBarStyle.Width(m.width).Render(left + "  " + helpText)
	return bar
}

func (m Model) renderErrorScreen() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(errorScreenStyle.Render("OKD TUI - Erreur de connexion"))
	b.WriteString("\n\n")
	b.WriteString(fmt.Sprintf("  %s\n", m.startupErr.Error()))
	b.WriteString("\n")
	b.WriteString("  [r] Réessayer  [q] Quitter\n")

	lines := strings.Count(b.String(), "\n")
	for i := lines; i < m.height; i++ {
		b.WriteString("\n")
	}
	return b.String()
}

// --- Helpers ---

func truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen == 1 {
		return string(runes[:1])
	}
	return string(runes[:maxLen-1]) + "…"
}

func encodeBase64(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}
