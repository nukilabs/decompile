package decompile

import "github.com/nukilabs/decompile/graph"

// stack is a LIFO stack of nodes.
type stack[N comparable] struct {
	nodes []*graph.Node[N]
}

// newStack creates a new stack.
func newStack[N comparable]() *stack[N] {
	return &stack[N]{
		nodes: make([]*graph.Node[N], 0),
	}
}

// push appends the node to the end of the stack.
func (q *stack[N]) push(node *graph.Node[N]) {
	q.nodes = append(q.nodes, node)
}

// pop removes and returns the last node in the stack.
func (q *stack[N]) pop() *graph.Node[N] {
	last := len(q.nodes) - 1
	node := q.nodes[last]
	q.nodes = q.nodes[:last]
	return node
}

// empty returns true if the stack is empty.
func (q *stack[N]) empty() bool {
	return len(q.nodes) == 0
}
