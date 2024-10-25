package main

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/bubbletea"
	"github.com/charmbracelet/wish/logging"
	"github.com/muesli/termenv"
)

const (
	start = iota
	nodeAmountInput
	spreadInput
	chooseStartingNode
	simulationRunning
)

type styles struct {
	border, nodesStyle, controls, inputStyle, directionStyle lipgloss.Style
}

type model struct {
	*program
	renderer                   *lipgloss.Renderer
	term                       ssh.Pty
	width, height, programStep int
	inputs                     []textinput.Model
	directions                 []string
	extraMessage, screenOutput string
	simulation                 Simulation
	styles                     styles
	hasError                   bool
}

type program struct {
	program *tea.Program
}

const (
	host = "0.0.0.0"
	port = "2224"
)

func main() {
	program := program{}

	home, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	s, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(host, port)),
		wish.WithHostKeyPath(fmt.Sprint(home, "/.ssh/gossip")),
		wish.WithMiddleware(
			bubbletea.MiddlewareWithProgramHandler(program.programHandler, termenv.ANSI256),
			// bubbletea.Middleware(teaHandler),
			activeterm.Middleware(), // Bubble Tea apps usually require a PTY.
			logging.Middleware(),
		),
	)
	if err != nil {
		log.Error("Could not start server", "error", err)
	}

	// var err error
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	log.Info("Starting SSH server", "host", host, "port", port)
	go func() {
		if err = s.ListenAndServe(); err != nil {
			log.Error("Could not start server", "error", err)
			done <- nil
		}
	}()

	<-done
	log.Info("Stopping SSH server")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer func() { cancel() }()
	if err := s.Shutdown(ctx); err != nil {
		log.Error("Could not stop server", "error", err)
	}

}

func (p *program) programHandler(s ssh.Session) *tea.Program {
	pty, _, _ := s.Pty()
	renderer := bubbletea.MakeRenderer(s)

	m := model{}
	m.term = pty
	m.renderer = renderer
	m.program = p
	m.initializeModel()

	p.program = tea.NewProgram(m, []tea.ProgramOption{tea.WithAltScreen(), tea.WithOutput(s), tea.WithInput(s)}...)

	return p.program

}

func (m *model) initializeModel() {

	m.inputs = make([]textinput.Model, 2) // 1. amount of nodes, 2. sendsPerNode
	m.directions = []string{
		"> press enter to start new simulation.\n> press ctrl+c to quit.",
		"> choose the number of nodes.\n> the press enter",
		"> choose the spread amount.\n> press enter to load simulation. press ctrl+z for previous input.",
		"> simulation loaded.\n> Click on a starting node, then press enter to start simulation.",
		"> simulation is running..."}
	m.programStep = 0

	m.styles.border = lipgloss.NewStyle()
	m.styles.inputStyle = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Align(lipgloss.Left).Width(25).Height(1).MarginLeft(1)

	m.inputs[0] = textinput.New()
	m.inputs[1] = textinput.New()

	m.inputs[0].Placeholder = "Number of nodes"
	m.inputs[1].Placeholder = "spread"
}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := message.(type) {
	case SimulationStatusMsg:
		if msg.done {

			m.extraMessage = fmt.Sprintf("> simulation finished in %d iterations and took %s. \n> press ctrl+x to reset.", msg.iteration, msg.time)
			m.programStep++

		}

	case RelayMsg:
		for _, coord := range msg.coords {
			m.simulation.pixelMap[coord] = "⬤"

		}

		m.drawPixels()
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit

		case "enter", "ctrl+z":
			if m.programStep >= start && m.programStep <= len(m.inputs) {
				if msg.String() == "ctrl+z" {
					m.programStep--
				}
			}
			if msg.String() == "enter" && m.programStep < simulationRunning {
				m.programStep++

			}
			cmds = append(cmds, m.updateProgramStep())

		case "ctrl+x":
			cmds = append(cmds, m.reset())
		}
	case tea.MouseMsg:

		if m.programStep == chooseStartingNode && msg.String() == "left press" && len(m.simulation.completedNodes) == 0 {

			nodeX, nodeY := msg.X-2, msg.Y-3 // substracting offset

			key := [2]int{nodeX, nodeY}

			char := m.simulation.pixelMap[key]

			if char != "◯" {
				return m, nil
			}

			m.simulation.pixelMap[key] = "⬤"
			m.simulation.completedNodes = append(m.simulation.completedNodes, m.simulation.nodeMap[key])

			// m.simulation.startingNode = m.simulation.nodeMap[key]
			m.drawPixels()

		}

	case tea.WindowSizeMsg:

		m.handleResize(msg)

	}
	// this handles the curser blinking
	cmds = append(cmds, m.updateInputs(message))
	return m, tea.Batch(cmds...)
}

func (m *model) updateInputs(message tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))

	switch msg := message.(type) {
	case tea.KeyMsg:
		_, err := strconv.Atoi(msg.String())
		if err != nil && msg.String() != "backspace" {
			return tea.Batch(cmds...)
		}

	}

	nodes, _ := strconv.Atoi(m.inputs[0].Value())
	spread, _ := strconv.Atoi(m.inputs[1].Value())

	// if nodes < 2 && m.programStep == nodeAmountInput {
	// 	m.extraMessage = "> please add 1 or more nodes. \n> ctrl+z to go back."
	// }

	m.simulation.nodeCount = nodes
	m.simulation.spread = spread

	for i := range m.inputs {

		m.inputs[i], cmds[i] = m.inputs[i].Update(message)

	}

	return tea.Batch(cmds...)
}

func (m *model) updateProgramStep() tea.Cmd {
	var cmd tea.Cmd
	for i := 0; i < len(m.inputs); i++ {
		if i == m.programStep-1 { // program steps start at 1
			cmd = m.inputs[i].Focus()
			continue
		}
		m.inputs[i].Blur()

	}
	if m.programStep == chooseStartingNode {
		m.inputs[1].Blur()
		m.loadBlankScreen()
		m.loadNodes()
		m.drawPixels()
		return cmd
	}
	if m.programStep == simulationRunning {
		go m.simulation.run(m.program.program)

		return cmd
	}

	return cmd
}

func (m *model) reset() tea.Cmd {
	var cmd tea.Cmd

	m.programStep = 1
	m.screenOutput = ""
	m.extraMessage = ""
	m.hasError = false
	m.simulation = Simulation{}
	m.inputs[0].Reset()
	m.inputs[1].Reset()

	for i := 0; i < len(m.inputs); i++ {
		if i == m.programStep-1 { // program steps start at 1
			cmd = m.inputs[i].Focus()
			continue
		}
		m.inputs[i].Blur()

	}

	return cmd

}

func (m *model) drawPixels() {
	if m.hasError {
		return
	}
	m.screenOutput = ""

	for y := 0; y < m.simulation.height; y++ {
		for x := 0; x < m.simulation.width; x++ {
			m.screenOutput += m.simulation.pixelMap[[2]int{x, y}]
		}
		if y < m.simulation.height-1 {
			m.screenOutput += "\n"
		}

	}

}

func (m *model) loadBlankScreen() {
	if m.hasError {
		return
	}

	m.simulation.pixelMap = make(map[[2]int]string)

	m.simulation.height = m.styles.nodesStyle.GetHeight()
	m.simulation.width = m.styles.nodesStyle.GetWidth()

	max := m.simulation.height * m.simulation.width
	if m.simulation.nodeCount > max {
		m.extraMessage = fmt.Sprintf("> too many nodes. Please enter %d or less\n> press ctrl+x", max)
		m.hasError = true
		return
	}
	for y := 0; y < m.simulation.height; y++ {
		for x := 0; x < m.simulation.width; x++ {
			m.simulation.pixelMap[[2]int{x, y}] = " "
		}
	}

}

func (m *model) loadNodes() {
	if m.hasError {
		return
	}
	m.simulation.nodes = make([]Node, m.simulation.nodeCount)
	m.simulation.nodeMap = make(map[[2]int]int)
	if m.simulation.isLoaded {
		return
	}

	for i := range m.simulation.nodes {
		x := rand.Int() % (m.simulation.width)
		y := rand.Int() % (m.simulation.height)

		pixel := [2]int{x, y}
		for m.simulation.pixelMap[pixel] == "◯" {
			x = rand.Int() % (m.simulation.width)
			y = rand.Int() % (m.simulation.height)
			pixel = [2]int{x, y}

		}
		node := Node{
			x: x,
			y: y,
		}
		m.simulation.pixelMap[pixel] = "◯"
		m.simulation.nodes[i] = node
		m.simulation.nodeMap[pixel] = i

	}
	m.simulation.isLoaded = true
}

func (m *model) handleResize(msg tea.WindowSizeMsg) {
	m.width = msg.Width
	m.height = msg.Height

	controlsHeight := 4
	nodeHeight := m.height - controlsHeight - 7

	border := lipgloss.NewStyle().Border(lipgloss.NormalBorder())

	m.styles.border = border.
		Width(m.width - 2).
		Height(m.height - 2).
		Align(lipgloss.Center).
		SetString("gossip visualizer")

	m.styles.nodesStyle = border.
		Width(m.width - 4).
		Height(nodeHeight)

	m.styles.controls = border.
		Width(m.width - 4).
		Height(controlsHeight)

	m.styles.inputStyle = border.
		Align(lipgloss.Left).
		Width(m.width / 8).
		Height(1)

	m.styles.directionStyle = lipgloss.NewStyle().
		Width(m.width / 2).
		MarginLeft(4)
}

func (m model) View() string {

	var message string
	if m.extraMessage == "" {
		message = m.directions[m.programStep]
	} else {
		message = m.extraMessage
	}

	ctrl := lipgloss.JoinHorizontal(lipgloss.Center,
		m.styles.inputStyle.Render(m.inputs[0].View()),
		m.styles.inputStyle.Render(m.inputs[1].View()),
		m.styles.directionStyle.Render(message),
	)

	return m.styles.border.Render(
		m.styles.nodesStyle.Render(m.screenOutput),
		m.styles.controls.Render(ctrl),
	)

}

// func (m *model) willError() {
// 	if m.hasError {
// 		return
// 	}
// }
