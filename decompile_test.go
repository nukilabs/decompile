package decompile

import (
	"fmt"
	"testing"

	"github.com/nukilabs/decompile/dominator"
	"github.com/nukilabs/decompile/graph"
)

func TestComputeIntervals(t *testing.T) {
	// Create a simple graph with root 1.
	g := graph.New[int]()

	// Set the root node.
	a := g.Node(1)
	g.SetRoot(a)

	// Add additional nodes.
	b := g.Node(2)
	c := g.Node(3)
	d := g.Node(4)
	e := g.Node(5)
	f := g.Node(6)

	// Add edges to form the control flow graph:
	// 1 -> 2, 2 -> 3, 3 -> 4, 4 -> 2, 2 -> 5, 5 -> 6, 6 -> 1.
	g.SetEdge(a, b)
	g.SetEdge(b, c)
	g.SetEdge(c, d)
	g.SetEdge(d, b)
	g.SetEdge(b, e)
	g.SetEdge(e, f)
	g.SetEdge(f, a)

	// Compute the intervals.
	intervals := Intervals(g)
	if len(intervals) != 2 {
		t.Fatalf("expected 2 intervals, got %d", len(intervals))
	}

	// Check the first interval.
	t.Log(intervals[0])
	items1 := []*graph.Node[int]{a}
	for _, node := range items1 {
		if !intervals[0].Contains(node) {
			t.Fatalf("interval 1 does not contain node %v", node)
		}
	}

	// Check the second interval.
	t.Log(intervals[1])
	items2 := []*graph.Node[int]{b, c, d, e, f}
	for _, node := range items2 {
		if !intervals[1].Contains(node) {
			t.Fatalf("interval 2 does not contain node %v", node)
		}
	}
}

func TestDerivedSequence(t *testing.T) {
	// Create a simple graph with root 1.
	g := graph.New[int]()

	// Set the root node.
	a := g.Node(1)
	g.SetRoot(a)

	// Add additional nodes.
	b := g.Node(2)
	c := g.Node(3)
	d := g.Node(4)
	e := g.Node(5)
	f := g.Node(6)

	// Add edges to form the control flow graph:
	// 1 -> 2, 2 -> 3, 3 -> 4, 4 -> 2, 2 -> 5, 5 -> 6, 6 -> 1.
	g.SetEdge(a, b)
	g.SetEdge(b, c)
	g.SetEdge(c, d)
	g.SetEdge(d, b)
	g.SetEdge(b, e)
	g.SetEdge(e, f)
	g.SetEdge(f, a)

	// Compute the derived sequence.
	graphs, intervals := DerivedSequence(g)

	// Check the number of graphs.
	if len(graphs) != len(intervals) {
		t.Fatalf("expected same number of graphs and corresponding intervals, got %d and %d", len(graphs), len(intervals))
	}

	for _, graph := range graphs {
		println(graph.String())
	}
}

func TestStructureLoops(t *testing.T) {
	// Create a simple graph with root 1.
	g := graph.New[int]()

	// Set the root node.
	n1 := g.Node(1)
	g.SetRoot(n1)

	// Add additional nodes.
	n2 := g.Node(2)
	n3 := g.Node(3)
	n4 := g.Node(4)
	n5 := g.Node(5)
	n6 := g.Node(6)
	n7 := g.Node(7)
	n8 := g.Node(8)
	n9 := g.Node(9)
	n10 := g.Node(10)
	n11 := g.Node(11)
	n12 := g.Node(12)
	n13 := g.Node(13)
	n14 := g.Node(14)
	n15 := g.Node(15)

	// Add edges to form the control flow graph:
	g.SetEdge(n1, n2)
	g.SetEdge(n1, n5)
	g.SetEdge(n2, n3)
	g.SetEdge(n2, n4)
	g.SetEdge(n3, n5)
	g.SetEdge(n4, n5)
	g.SetEdge(n5, n6)
	g.SetEdge(n6, n7)
	g.SetEdge(n7, n8)
	g.SetEdge(n7, n9)
	g.SetEdge(n8, n9)
	g.SetEdge(n8, n10)
	g.SetEdge(n9, n10)
	g.SetEdge(n10, n11)
	g.SetEdge(n6, n12)
	g.SetEdge(n12, n13)
	g.SetEdge(n13, n14)
	g.SetEdge(n14, n13)
	g.SetEdge(n14, n15)
	g.SetEdge(n15, n6)

	// Compute the derived sequence.
	graphs, intervals := DerivedSequence(g)

	for _, graph := range graphs {
		fmt.Println(graph)
	}

	for _, iis := range intervals {
		for _, interval := range iis {
			fmt.Println(interval)
		}
	}

	// Compute the dominator tree.
	dom := dominator.New(g)

	// Init DFS numbering.
	g.InitOrder()

	// Compute the structure loops.
	loops, _ := StructureLoops(g, dom)
	conds := StructureTwoWayConditionals(g, dom)

	// Check the structure loop.
	for _, loop := range loops {
		fmt.Println(loop)
	}
	for _, cond := range conds {
		fmt.Println(cond)
	}
}
