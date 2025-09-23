package arboreal

import "sort"

type Hashable interface {
	Hash() string
}

type Stack[T any] struct {
	Items []T `json:"items"`
}

func (s *Stack[T]) IsEmpty() bool {
	return len(s.Items) == 0
}

func (s *Stack[T]) Push(b T) {
	s.Items = append(s.Items, b)
}

func (s *Stack[T]) Pop() T {
	last := s.Items[len(s.Items)-1]
	s.Items = s.Items[:len(s.Items)-1]
	return last
}

type Graph[T Hashable] struct {
	Transitions [][]int `json:"transitions"`
	Nodes       []T     `json:"nodes"`
	lookupMap   map[string]int
}

func (g *Graph[T]) grow() {
	for idx, row := range g.Transitions {
		g.Transitions[idx] = append(row, -1)
	}
	newRow := make([]int, len(g.Transitions)+1)
	for idx, _ := range newRow {
		newRow[idx] = -1
	}
	g.Transitions = append(g.Transitions, newRow)
}

func (g *Graph[T]) AddNode(n T) int {
	if g.lookupMap == nil {
		g.lookupMap = make(map[string]int)
	}

	index, found := g.lookupMap[n.Hash()]
	if !found {
		index = len(g.Transitions)
		g.lookupMap[n.Hash()] = index
		g.Nodes = append(g.Nodes, n)
		g.grow()
	}

	return index
}

func (g *Graph[T]) AddTransition(from, to T) {
	fromIndex := g.AddNode(from)
	toIndex := g.AddNode(to)

	var highest = -1
	for _, val := range g.Transitions[fromIndex] {
		if val > highest {
			highest = val
		}
	}

	g.Transitions[fromIndex][toIndex] = highest + 1
}

func (g *Graph[T]) Initial() T {
	return g.Nodes[0]
}

func (g *Graph[T]) Children(of T) []T {
	index, found := g.lookupMap[of.Hash()]
	if !found {
		return []T{}
	}

	var children []T
	var lookup = make(map[int]int)
	var childIndices []int

	for idx, v := range g.Transitions[index] {
		if v > -1 {
			childIndices = append(childIndices, v)
			lookup[v] = idx
		}
	}

	sort.Ints(childIndices)

	for _, idx := range childIndices {
		children = append(children, g.Nodes[lookup[idx]])
	}

	return children
}
