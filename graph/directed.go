package graph

import (
	"strings"
)

// Graph represents a directed graph.
type Graph[N comparable] struct {
	root     *Node[N]
	nodes    map[ID[N]]*Node[N]
	incoming map[*Node[N]]map[*Node[N]]struct{}
	outgoing map[*Node[N]]map[*Node[N]]struct{}
}

// New creates a new directed graph with a given root node.
func New[N comparable]() *Graph[N] {
	return &Graph[N]{
		nodes:    map[ID[N]]*Node[N]{},
		incoming: map[*Node[N]]map[*Node[N]]struct{}{},
		outgoing: map[*Node[N]]map[*Node[N]]struct{}{},
	}
}

// String returns a string representation of the graph.
func (g *Graph[N]) String() string {
	var sb strings.Builder
	for _, node := range g.nodes {
		sb.WriteString(node.String())
		sb.WriteString(" -> ")
		for _, succ := range g.Successors(node) {
			sb.WriteString(succ.String())
			sb.WriteString(" ")
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// SetRoot sets the root node of the graph.
func (g *Graph[N]) SetRoot(node *Node[N]) {
	g.root = node
}

// Root returns the root node of the graph.
func (g *Graph[N]) Root() *Node[N] {
	return g.root
}

// GetNode returns the node with the given value.
func (g *Graph[N]) GetNode(value N) (*Node[N], bool) {
	id := ID[N]{Kind: DefaultNode, Value: value}
	node, ok := g.nodes[id]
	return node, ok
}

// Node adds a new node with the given value to the graph.
// If a node with the same value already exists, it returns the existing node.
func (g *Graph[N]) Node(value N) *Node[N] {
	id := ID[N]{Kind: DefaultNode, Value: value}
	if node, ok := g.nodes[id]; ok {
		return node
	}
	node := &Node[N]{
		Kind:  DefaultNode,
		Value: value,
	}
	g.nodes[node.ID()] = node
	g.incoming[node] = make(map[*Node[N]]struct{})
	g.outgoing[node] = make(map[*Node[N]]struct{})
	return node
}

// Interval adds a new interval node to the graph.
// If an interval node with the same index already exists, it returns the existing node.
func (g *Graph[N]) Interval(idx int) *Node[N] {
	id := ID[N]{Kind: IntervalNode, Idx: idx}
	if node, ok := g.nodes[id]; ok {
		return node
	}
	node := &Node[N]{
		Kind: IntervalNode,
		Idx:  idx,
	}
	g.nodes[node.ID()] = node
	g.incoming[node] = make(map[*Node[N]]struct{})
	g.outgoing[node] = make(map[*Node[N]]struct{})
	return node
}

// SetEdge creates an edge from the "from" node to the "to" node.
func (g *Graph[N]) SetEdge(from, to *Node[N]) {
	if _, ok := g.outgoing[from]; !ok {
		g.outgoing[from] = make(map[*Node[N]]struct{})
	}
	g.outgoing[from][to] = struct{}{}

	if _, ok := g.incoming[to]; !ok {
		g.incoming[to] = make(map[*Node[N]]struct{})
	}
	g.incoming[to][from] = struct{}{}
}

// Nodes returns a slice of all nodes in the graph.
func (g *Graph[N]) Nodes() []*Node[N] {
	var nodes []*Node[N]
	for _, node := range g.nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

// Len returns the number of nodes in the graph.
func (g *Graph[N]) Len() int {
	return len(g.nodes)
}

// Successors returns a slice of nodes that are directly reachable from the given node.
func (g *Graph[N]) Successors(n *Node[N]) []*Node[N] {
	var succ []*Node[N]
	for neighbor := range g.outgoing[n] {
		succ = append(succ, neighbor)
	}
	return succ
}

// Predecessors returns a slice of nodes that have a direct edge to the given node.
func (g *Graph[N]) Predecessors(n *Node[N]) []*Node[N] {
	var preds []*Node[N]
	for neighbor := range g.incoming[n] {
		preds = append(preds, neighbor)
	}
	return preds
}

// DFS performs a depth-first search on the graph.
//   - The 'pre' callback is invoked before exploring a node's children,
//   - The 'post' callback is invoked after all its children have been processed.
func (g *Graph[N]) DFS(pre, post func(n *Node[N])) {
	visited := make(map[ID[N]]bool)

	var visit func(n *Node[N])
	visit = func(n *Node[N]) {
		visited[n.ID()] = true
		if pre != nil {
			pre(n)
		}
		// Use the Successors function to get all nodes directly reachable from n.
		for _, succ := range g.Successors(n) {
			if !visited[succ.ID()] {
				visit(succ)
			}
		}
		if post != nil {
			post(n)
		}
	}

	// Start DFS from the root node.
	visit(g.root)
}

// InitOrder initializes the reverse postorder numbering of the graph nodes.
func (g *Graph[N]) InitOrder() {
	num := g.Len()
	g.DFS(nil, func(n *Node[N]) {
		n.Order = num
		num--
	})
}
