package decompile

import (
	"slices"

	"github.com/nukilabs/decompile/graph"
)

// descReversePostOrder returns a slice of nodes in descending reverse postorder.
func descReversePostOrder[N comparable](nodes []*graph.Node[N]) []*graph.Node[N] {
	slices.SortFunc(nodes, func(a, b *graph.Node[N]) int {
		return b.Order - a.Order
	})
	return nodes
}

// ascReversePostOrder returns a slice of nodes in ascending reverse postorder.
func ascReversePostOrder[N comparable](nodes []*graph.Node[N]) []*graph.Node[N] {
	slices.SortFunc(nodes, func(a, b *graph.Node[N]) int {
		return a.Order - b.Order
	})
	return nodes
}

// contains returns true if the given node is in the list of nodes.
func contains[N comparable](nodes []*graph.Node[N], node *graph.Node[N]) bool {
	return slices.ContainsFunc(nodes, func(n *graph.Node[N]) bool {
		return n.ID() == node.ID()
	})
}
