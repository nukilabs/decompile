package graph

import "fmt"

// Kind represents the kind of a node.
type Kind uint8

const (
	// DefaultNode is a default node.
	DefaultNode Kind = iota
	// IntervalNode is an interval node.
	IntervalNode
)

// ID is a unique identifier for a node.
type ID[N comparable] struct {
	// Kind of the node.
	Kind Kind
	// Index of the interval node.
	Idx int
	// Value of the default node.
	Value N
}

// Node represents a node in a
type Node[N comparable] struct {
	// Kind of the node.
	// Either a default node or an interval node.
	Kind Kind
	// Value of the default node.
	Value N
	// Index of the interval node.
	Idx int

	// Order of the node in the graph.
	// Zero if not initialized.
	Order int

	// Node used in loop.
	IsLoopNode bool
	// Node used as head node in loop.
	IsLoopHead bool
	// Node used as latch node in loop.
	IsLoopLatch bool
}

// ID returns the unique identifier of the node.
func (n *Node[N]) ID() ID[N] {
	return ID[N]{
		Kind:  n.Kind,
		Idx:   n.Idx,
		Value: n.Value,
	}
}

// String returns a string representation of the node.
func (n *Node[N]) String() string {
	switch n.Kind {
	case DefaultNode:
		return fmt.Sprintf("%v", n.Value)
	case IntervalNode:
		return fmt.Sprintf("I(%d)", n.Idx)
	}
	return ""
}
