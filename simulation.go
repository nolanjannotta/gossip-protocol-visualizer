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
	spread         int
}

type RelayMsg struct {
	status bool
	coords [][2]int
	// coord  [2]int
}

type SimulationStatus struct {
	done      bool
	iteration int
	time      time.Duration
}
type nodeIds []int

func (n nodeIds) sortByDistance(nodeId int, nodes []Node) {
	slices.SortFunc(n,
		func(a, b int) int {
			x, y := nodes[nodeId].x, nodes[nodeId].y

			x1, y1 := nodes[a].x, nodes[a].y
			x2, y2 := nodes[b].x, nodes[b].y

			distance1 := math.Sqrt(math.Pow(float64(x1-x), 2) + math.Pow(float64(y1-y), 2))
			distance2 := math.Sqrt(math.Pow(float64(x2-x), 2) + math.Pow(float64(y2-y), 2))

			return cmp.Compare(distance1, distance2)

		})
}

func (n nodeIds) removeAt(index int) nodeIds {
	return append(n[:index], n[index+1:]...)
}

func (s *Simulation) run(p *tea.Program) {

	if len(s.completedNodes) == 0 {
		return
	}

	var coords [][2]int
	var nodeIds nodeIds

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
		iterations++
		// fmt.Println(lastIteration)
		for _, nodeId := range lastIteration {
			if done {
				break
			}

			nodeIds.sortByDistance(nodeId, s.nodes)

			for range s.spread {
				if len(s.completedNodes) >= len(s.nodes) {
					elapsed := time.Since(start)
					p.Send(SimulationStatus{done: true, iteration: iterations, time: elapsed})
					done = true
					break
				}

				for i, id := range nodeIds {
					if !slices.Contains(s.completedNodes, id) {
						coord := [2]int{s.nodes[id].x, s.nodes[id].y}
						coords = append(coords, coord)
						currentIteration = append(currentIteration, id)
						s.completedNodes = append(s.completedNodes, id)
						nodeIds = nodeIds.removeAt(i)
						break
					}
				}

			}
			p.Send(RelayMsg{status: true, coords: coords})

		}

		lastIteration = currentIteration
		currentIteration = nil

		// p.Send(RelayMsg{status: true, coords: coords}) // calling this here makes it faster but look less cool
		coords = nil

	}

}
