package main

import (
	"math"
	"math/rand"
	"slices"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type Node struct {
	state                         string
	id, x, y, successRate, spread int
}

type Simulation struct {
	numNodes       int
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

// func (n *Node) sendMsg(message string) bool {
// 	if n.state == message {
// 		return false
// 	}
// 	n.state = message
// 	return true

// }

func (s *Simulation) run(p *tea.Program) {
	if len(s.completedNodes) == 0 {
		return
	}

	var coords [][2]int

	// 1. cycle through all completed nodes (nodes that received a message)
	// 2. for each completed node, choose 3 (s.spread) more to send messages to
	// 3. if successful, add those nodes to completed nodes list
	// 4. update tea.Program to update the UI
	// 5. continue until all nodes are complete

	for {

		for _, nodeId := range s.completedNodes {

			// d=√((x_2-x_1)²+(y_2-y_1)²)
			distances := []float64{}
			for _, node := range s.nodes {

				x1, y1 := s.nodes[nodeId].x, s.nodes[nodeId].y
				x2, y2 := node.x, node.y
				if x1 == x2 && y1 == y2 {
					continue
				}
				distance := math.Sqrt(math.Pow(float64(x2-x1), 2) + math.Pow(float64(y2-y1), 2))
				distances = append(distances, distance)

			}
			slices.Sort(distances)

			for range s.spread {
				if len(s.completedNodes) >= len(s.nodes) {
					break
				}

				recipient := rand.Int() % len(s.nodes)

				for slices.Contains(s.completedNodes, recipient) {
					// if recipient is already in the completed list, find a different one
					recipient = rand.Int() % len(s.nodes)
				}
				coord := [2]int{s.nodes[recipient].x, s.nodes[recipient].y}
				coords = append(coords, coord)
				s.completedNodes = append(s.completedNodes, recipient)

			}

		}
		time.Sleep(time.Second)
		p.Send(RelayMsg{status: true, coords: coords})
		coords = nil

		// fmt.Println(len(s.completedNodes))

	}

}
