package decompile

import (
	"slices"
	"strings"

	"github.com/nukilabs/decompile/graph"
)

// An Interval I(h) with header node h is a maximal single-entry subgraph of
// a control flow graph in which h is the only entry node and all cycles contain h.
type Interval[N comparable] struct {
	graph *graph.Graph[N]
	head  *graph.Node[N]
	nodes map[graph.ID[N]]*graph.Node[N]
}

// New creates a new interval with a given head node.
func NewInterval[N comparable](head *graph.Node[N], g *graph.Graph[N]) *Interval[N] {
	return &Interval[N]{
		graph: g,
		head:  head,
		nodes: map[graph.ID[N]]*graph.Node[N]{
			head.ID(): head,
		},
	}
}

// Add adds a node to the interval.
func (i *Interval[N]) add(node *graph.Node[N]) {
	i.nodes[node.ID()] = node
}

// Contains returns true if the interval contains a given node.
func (i *Interval[N]) Contains(node *graph.Node[N]) bool {
	_, ok := i.nodes[node.ID()]
	return ok
}

// Nodes returns the nodes in the interval.
func (i *Interval[N]) Nodes() []*graph.Node[N] {
	nodes := make([]*graph.Node[N], 0, len(i.nodes))
	for _, node := range i.nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

// Predecessors returns the predecessors of a node in the interval.
func (i *Interval[N]) Predecessors(node *graph.Node[N]) []*graph.Node[N] {
	preds := make([]*graph.Node[N], 0)
	for _, pred := range i.graph.Predecessors(node) {
		if i.Contains(pred) {
			preds = append(preds, pred)
		}
	}
	return preds
}

// String returns a string representation of the interval.
func (i *Interval[N]) String() string {
	var b strings.Builder
	b.WriteString("I(")
	b.WriteString(i.head.String())
	b.WriteString(") {")
	idx := 0
	for _, node := range i.nodes {
		if idx > 0 {
			b.WriteString(",")
		}
		b.WriteString(node.String())
		idx++
	}
	b.WriteString("}")
	return b.String()
}

// Intervals computes the intervals of a control flow.
func Intervals[N comparable](g *graph.Graph[N]) []*Interval[N] {
	intervals := make([]*Interval[N], 0)

	// 1. Establish a set for header nodes and initialize it with n⁰, the
	//    unique entry node for the
	headers := newQueue[N]()
	headers.push(g.Root())

	// 2. While the set of header nodes is not empty, do the following:
	for !headers.empty() {
		// 2.1. Put h in I(h) as the first element of I(h).
		head := headers.pop()
		interval := NewInterval(head, g)

		// 2.2. Add to I(h) any node all of whose immediate predecessors are
		//      already in I(h).
		for {
			node, ok := findNodeWithImmediatePredecessorsInInterval(g, interval)
			if !ok {
				break
			}
			interval.add(node)
		}

		// 3. Add to H all nodes in G which are not already in H and which are not
		//    in I(h) but which have immediate predecessors in I(h). Therefore a
		//    node is added to H the first time any (but not all) of its immediate
		//    predecessors become members of an interval.
		for {
			node, ok := findUnprocessedNodeWithImmediatePredecessors(g, interval, headers)
			if !ok {
				break
			}
			headers.push(node)
		}

		// 4. Add I(h) to a set Is of intervals being developed.
		intervals = append(intervals, interval)

		// 5. Repeat from step 2.
	}

	return intervals
}

// findNodeWithImmediatePredecessorsInInterval returns a node not in the interval
// with all immediate predecessors in the interval.
func findNodeWithImmediatePredecessorsInInterval[N comparable](g *graph.Graph[N], interval *Interval[N]) (*graph.Node[N], bool) {
outer:
	for _, node := range g.Nodes() {
		// Skip the root node.
		if g.Root().ID() == node.ID() {
			continue
		}
		// Skip nodes already in the interval.
		if interval.Contains(node) {
			continue
		}

		for _, pred := range g.Predecessors(node) {
			// Skip node as it has a predecessor not in the interval.
			if !interval.Contains(pred) {
				continue outer
			}
		}

		// All predecessors are in the interval.
		return node, true
	}

	return nil, false
}

// findUnprocessedNodeWithImmediatePredecessors locates a node not in the interval
// nor in the headers that has at least one immediate predecessor in the interval.
func findUnprocessedNodeWithImmediatePredecessors[N comparable](g *graph.Graph[N], interval *Interval[N], headers *queue[N]) (*graph.Node[N], bool) {
	for _, node := range g.Nodes() {
		// Skip nodes already in the interval.
		if interval.Contains(node) {
			continue
		}
		// Skip nodes already in the headers.
		if headers.contains(node) {
			continue
		}

		if slices.ContainsFunc(g.Predecessors(node), interval.Contains) {
			return node, true
		}
	}

	return nil, false
}
