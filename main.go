package main

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	start = iota
	nodeAmountInput
	successRateInput
	sendsPerNodeInput
	chooseStartingNode
	simulationRunning
)

type model struct {
	border         lipgloss.Style
	width          int
	height         int
	nodesStyle     lipgloss.Style
	controls       lipgloss.Style
	paramsDisplay  string
	inputs         []textinput.Model
	programStep    int
	inputStyle     lipgloss.Style
	directions     []string
	directionStyle lipgloss.Style
	screenOutput   string
	simulation     Simulation

	hello string
}

var program = tea.Program{}

func main() {

	m := model{
		border: lipgloss.NewStyle(),
		inputs: make([]textinput.Model, 3), // 1. amount of nodes, 2. successRate, 3. sendsPerNode
		directions: []string{
			"> press enter to start new simulation",
			"> choose the number of nodes",
			"> choose the success rate of messages",
			"> choose the amount of messages a node sends. then press enter to load simulation.",
			"> simulation loaded.\nClick on a starting node",
			"> simulation is running"},
		programStep: 0,
		// hello:       "hello",
	}

	m.inputStyle = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Align(lipgloss.Left).Width(25).Height(1).MarginLeft(1)

	m.inputs[0] = textinput.New()
	m.inputs[1] = textinput.New()
	m.inputs[2] = textinput.New()

	m.inputs[0].Placeholder = "Number of nodes"
	m.inputs[1].Placeholder = "success rate %"
	m.inputs[2].Placeholder = "sends per node"

	// m.simulation.startingNode = -1

	p := tea.NewProgram(m, tea.WithAltScreen())

	program = *p
	// m.p = p

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}

}

func (m model) Init() tea.Cmd {
	return nil
}

func (m model) Update(message tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := message.(type) {

	case RelayMsg:
		// fmt.Println("msg received in program")
		// key := [2]int{msg.x, msg.y}

		for _, coord := range msg.coords {
			m.simulation.pixelMap[coord] = green(m.simulation.pixelMap[coord])
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
		// the node screen start at (2,2), so we can subtract 2 from each coordinate
		// to get the relative position in the screen... i think...

		if m.programStep == chooseStartingNode && msg.String() == "left press" && len(m.simulation.completedNodes) == 0 {

			// fmt.Println(msg.X, msg.Y)
			nodeX, nodeY := msg.X-2, msg.Y-3

			key := [2]int{nodeX, nodeY}

			char := m.simulation.pixelMap[key]

			if char != "⬤" {
				return m, nil
			}

			m.simulation.pixelMap[key] = green(char)
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
	success, _ := strconv.Atoi(m.inputs[1].Value())
	spread, _ := strconv.Atoi(m.inputs[2].Value())

	if success > 100 {
		m.inputs[1].SetValue("100")
		success = 100
	}
	m.simulation.numNodes = nodes
	m.simulation.successRate = success
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
		m.inputs[2].Blur()
		m.loadBlankScreen()
		m.loadNodes()
		m.drawPixels()
		m.paramsDisplay = fmt.Sprint(
			"    nodes:", len(m.simulation.nodes),
			"    success rate: ", m.simulation.successRate, "%",
			"    sends per node: ", m.simulation.spread)
		return cmd
	}
	if m.programStep == simulationRunning {

		go m.simulation.run(&program)

		// node := <-c

		// m.simulation.pixelMap[[2]int{node.x, node.y}] = green(m.simulation.pixelMap[[2]int{node.x, node.y}])
		// m.drawPixels()
		return cmd
	}

	return cmd
}

func (m *model) reset() tea.Cmd {
	var cmd tea.Cmd

	m.programStep = 1
	m.screenOutput = ""
	m.simulation = Simulation{}
	m.inputs[0].Reset()
	m.inputs[1].Reset()
	m.inputs[2].Reset()

	for i := 0; i < len(m.inputs); i++ {
		if i == m.programStep-1 { // program steps start at 1
			cmd = m.inputs[i].Focus()
			continue
		}
		m.inputs[i].Blur()

	}

	return cmd

}

// func (m *model) changePixel(x, y int) {
// 	// index := y*m.simulation.width + x
// 	m.screenOutput = strings.Replace(m.screenOutput, "⬤", green("⬤"), 2)
// 	// m.screenOutput[index] = green(m.screenOutput[index])
// }

func (m *model) drawPixels() {
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
	m.simulation.pixelMap = make(map[[2]int]string)

	m.simulation.height = m.nodesStyle.GetHeight()
	m.simulation.width = m.nodesStyle.GetWidth()

	for x := 0; x < m.simulation.width; x++ {
		for y := 0; y < m.simulation.height; y++ {
			m.simulation.pixelMap[[2]int{x, y}] = " "
		}
	}

}

func (m *model) loadNodes() {
	m.simulation.nodes = make([]Node, m.simulation.numNodes)
	m.simulation.nodeMap = make(map[[2]int]int)
	if m.simulation.isLoaded {
		return
	}

	for i := range m.simulation.nodes {
		x := rand.Int() % (m.simulation.width - 1)
		y := rand.Int() % (m.simulation.height)
		node := Node{
			state: "state",
			// totalNodes:  len(m.simulation.nodes),
			x:           x,
			y:           y,
			id:          i,
			successRate: m.simulation.successRate,
			spread:      m.simulation.spread,
			// otherNodes:  make([]*Node, len(m.simulation.nodes)),
			// otherNodes:
		}
		m.simulation.pixelMap[[2]int{x, y}] = "⬤"
		m.simulation.nodes[i] = node
		m.simulation.nodeMap[[2]int{x, y}] = i

	}

	// theres got to be a better way to do this

	// pointerToNodes := &m.simulation.nodes
	// for i := range m.simulation.nodes {
	// 	// nodes := make([]Node, len(m.simulation.nodes))
	// 	// copy(nodes, m.simulation.nodes)
	// 	m.simulation.nodes[i].otherNodes = pointerToNodes

	// }
	// fmt.Println(m.simulation.nodes[2])

	m.simulation.isLoaded = true
}

func (m *model) handleResize(msg tea.WindowSizeMsg) {
	m.width = msg.Width
	m.height = msg.Height

	controlsHeight := 4
	nodeHeight := m.height - controlsHeight - 7

	border := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder())

	m.border = border.
		Width(m.width - 2).
		Height(m.height - 2).
		Align(lipgloss.Center).
		SetString("gossip visualizer")

	m.nodesStyle = border.
		Width(m.width - 4).
		Height(nodeHeight)

	m.controls = border.
		Width(m.width - 4).
		Height(controlsHeight)

	m.inputStyle = border.
		Align(lipgloss.Left).
		Width(m.width / 8).
		Height(1)
		// MarginLeft(1)

	m.directionStyle = lipgloss.NewStyle().
		Width(m.width / 2).
		MarginLeft(4)
	// Background(lipgloss.Color("34"))

	// m.subcontainer = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).Width(m.width - 5).Height(m.height - 4).Align(lipgloss.Center).Background(lipgloss.Color("123"))

}

func red(letter string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")).Render(letter)
}
func green(letter string) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#008000")).Render(letter)
}

func (m model) View() string {

	ctrl := lipgloss.JoinHorizontal(lipgloss.Center,
		m.inputStyle.Render(m.inputs[0].View()),
		m.inputStyle.Render(m.inputs[1].View()),
		m.inputStyle.Render(m.inputs[2].View()),
		m.directionStyle.Render(m.directions[m.programStep]),
	)

	return m.border.Render(
		m.nodesStyle.Render(m.screenOutput),
		m.controls.Render(ctrl),
	)
	// return ""

}
