package decompile

import "github.com/nukilabs/decompile/graph"

// queue is a FIFO queue of nodes which keeps track of all nodes that has been
// in the queue.
type queue[N comparable] struct {
	all   map[graph.ID[N]]struct{}
	nodes []*graph.Node[N]
}

// newQueue creates a new queue.
func newQueue[N comparable]() *queue[N] {
	return &queue[N]{
		all:   make(map[graph.ID[N]]struct{}),
		nodes: make([]*graph.Node[N], 0),
	}
}

// push adds a node to the queue if it was not already present.
func (q *queue[N]) push(node *graph.Node[N]) {
	if _, ok := q.all[node.ID()]; !ok {
		q.nodes = append(q.nodes, node)
		q.all[node.ID()] = struct{}{}
	}
}

// pop removes and returns the first node in the queue.
func (q *queue[N]) pop() *graph.Node[N] {
	node := q.nodes[0]
	q.nodes = q.nodes[1:]
	return node
}

// empty returns true if the queue is empty.
func (q *queue[N]) empty() bool {
	return len(q.nodes) == 0
}

// has reports whether the given node is present in the queue or has been
// present before.
func (q *queue[N]) contains(node *graph.Node[N]) bool {
	_, ok := q.all[node.ID()]
	return ok
}
