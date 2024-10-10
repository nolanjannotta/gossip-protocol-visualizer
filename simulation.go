package main

import (
	"cmp"
	"math"
	"slices"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type Node struct {
	x, y int
}

type Simulation struct {
	nodeCount      int
	nodes          []Node            //index is the id
	completedNodes []int             //list of ids that are completed
	nodeMap        map[[2]int]int    // x,y mapped to node id
	pixelMap       map[[2]int]string // try this
	isLoaded       bool
	height         int
	width          int
	successRate    int
	spread         int // rename to "spread"
}

type RelayMsg struct {
	status bool
	coords [][2]int
}

type SimulationStatus struct {
	done      bool
	iteration int
	time      time.Duration
}

func (s *Simulation) run(p *tea.Program) {

	if len(s.completedNodes) == 0 {
		return
	}

	var coords [][2]int
	var nodeIds []int
	var currentIteration []int
	var lastIteration []int

	for i := range s.nodes {
		nodeIds = append(nodeIds, i)
	}

	lastIteration = s.completedNodes

	done := false
	var iterations int
	start := time.Now()
	for !done {

		for _, nodeId := range lastIteration {

			slices.SortFunc(nodeIds,
				func(a, b int) int {
					x, y := s.nodes[nodeId].x, s.nodes[nodeId].y

					x1, y1 := s.nodes[a].x, s.nodes[a].y
					x2, y2 := s.nodes[b].x, s.nodes[b].y

					distance1 := math.Sqrt(math.Pow(float64(x1-x), 2) + math.Pow(float64(y1-y), 2))
					distance2 := math.Sqrt(math.Pow(float64(x2-x), 2) + math.Pow(float64(y2-y), 2))

					return cmp.Compare(distance1, distance2)

				})
			var recipient int
			for range s.spread {
				if len(s.completedNodes) >= len(s.nodes) {
					done = true
					break
				}

				for i, id := range nodeIds {
					if !slices.Contains(s.completedNodes, id) {
						recipient = id
						nodeIds = append(nodeIds[:i], nodeIds[i+1:]...)
						break
					}
				}

				coord := [2]int{s.nodes[recipient].x, s.nodes[recipient].y}
				coords = append(coords, coord)
				currentIteration = append(currentIteration, recipient)

			}

		}

		s.completedNodes = append(s.completedNodes, currentIteration...)
		lastIteration = currentIteration
		currentIteration = []int{}
		iterations++
		p.Send(RelayMsg{status: true, coords: coords})
		coords = [][2]int{}

	}
	elapsed := time.Since(start)
	p.Send(SimulationStatus{done: true, iteration: iterations, time: elapsed})

}
