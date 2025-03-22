package decompile

import (
	"errors"
	"fmt"
	"math"
	"slices"

	"github.com/nukilabs/decompile/dominator"
	"github.com/nukilabs/decompile/graph"
)

// Structure structures the control flow graph into primitives.
func Structure[N comparable](g *graph.Graph[N]) ([]Primitive[N], error) {
	prims := make([]Primitive[N], 0)
	errs := make([]error, 0)
	// Initialize the control flow graph.
	g.InitOrder()
	// Compute the dominator tree.
	dom := dominator.New(g)
	// Structure loops in the control flow graph.
	loops, err := StructureLoops(g, dom)
	if err != nil {
		errs = append(errs, err)
	}
	prims = append(prims, loops...)
	// Structure 2-way conditionals in the control flow graph.
	conditionals := StructureTwoWayConditionals(g, dom)
	prims = append(prims, conditionals...)
	return prims, errors.Join(errs...)
}

// StructureLoops structures loops in the given control flow graph.
func StructureLoops[N comparable](g *graph.Graph[N], dom *dominator.Tree[N]) ([]Primitive[N], error) {
	graphs, intervals := DerivedSequence(g)
	prims := make([]Primitive[N], 0)
	errs := make([]error, 0)
	for i := range graphs {
		for _, interval := range intervals[i] {
			head, latch, ok := findLatch(graphs[0], interval, intervals)
			if ok && !latch.IsLoopNode {
				latch.IsLoopLatch = true
				nodes := markNodesInLoop(g, head, latch, dom)
				kind, err := findLoopKind(g, head, latch, nodes)
				if err != nil {
					errs = append(errs, err)
					continue
				}
				follow, err := findLoopFollow(g, kind, head, latch, nodes, dom)
				if err != nil {
					errs = append(errs, err)
					continue
				}

				// Create loop primitive.
				prim := Primitive[N]{
					Kind:  kind,
					Entry: head.Value,
					Extra: map[string]N{
						"latch": latch.Value,
					},
				}

				if follow != nil {
					prim.Extra["follow"] = follow.Value
					prim.Exit = follow.Value
				}

				// Remove the follow node from the loop body.
				for i, node := range nodes {
					if node == follow {
						nodes = slices.Delete(nodes, i, i+1)
					}
				}

				// Add nodes to loop body.
				for _, node := range nodes {
					prim.Body = append(prim.Body, node.Value)
				}

				prims = append(prims, prim)
			}
		}
	}
	return prims, errors.Join(errs...)
}

// findLatch locates the loop latch node in the interval, based on the interval
// header node. The boolean return value indicates success.
func findLatch[N comparable](g *graph.Graph[N], interval *Interval[N], intervals [][]*Interval[N]) (*graph.Node[N], *graph.Node[N], bool) {
	var latch *graph.Node[N]
	// iis is used to look up the nodes belonging to an interval, e.g. I_1. Note,
	var iis []*Interval[N]
	for _, i := range intervals {
		iis = append(iis, i...)
	}
	// Each header of an interval in G^i is checked for having a back-edge from a
	// latching node that belong to the same interval.
	for _, pred := range interval.Predecessors(interval.head) {
		if latch == nil || pred.Order > latch.Order {
			latch = pred
		}
	}
	if latch != nil {
		// Locate node in original control flow graph corresponding to the latch
		// node in the derived sequence of graphs.
		if l, ok := g.GetNode(latch.Value); ok {
			return interval.head, l, true
		}
		h := findOrigHead(interval.head, iis)
		cands := descReversePostOrder(g.Predecessors(h))
		for i, cand := range cands {
			if cand.Order < h.Order {
				cands = cands[:i]
				break
			}
		}
		l := findOrigLatch(latch, cands, iis)
		return h, l, true
	}
	return nil, nil, false
}

// findOrigHead returns the loop header node in the original control flow graph
// corresponding to the header node of an interval in the derived sequence of
// graphs.
func findOrigHead[N comparable](head *graph.Node[N], intervals []*Interval[N]) *graph.Node[N] {
	// Find the outer-most interval which has the loop header as interval header.
	i, ok := getInterval(head.ID(), intervals)
	if !ok {
		return head
	}
	return findOrigHead(i.head, intervals)
}

// findOrigLatch returns the latch node in the original control flow graph
// corresponding to the latch node of an interval in the derived sequence of
// graphs.
func findOrigLatch[N comparable](latch *graph.Node[N], cands []*graph.Node[N], intervals []*Interval[N]) *graph.Node[N] {
	i, ok := getInterval(latch.ID(), intervals)
	if !ok {
		return latch
	}
	l, ok := findNodeInInterval(cands, i, intervals)
	if !ok {
		panic("unable to find latch node in original control flow graph")
	}
	return l
}

// findNodeInInterval locates the a latch node in the original control flow
// graph corresponding to one of the latch node candidates in the derived
// sequence of graphs.
func findNodeInInterval[N comparable](cands []*graph.Node[N], interval *Interval[N], intervals []*Interval[N]) (*graph.Node[N], bool) {
	for _, cand := range cands {
		for _, node := range interval.Nodes() {
			j, ok := getInterval(cand.ID(), intervals)
			if !ok {
				if node.Value == cand.Value {
					return node, true
				}
			} else if l, ok := findNodeInInterval(cands, j, intervals); ok {
				return l, true
			}
		}
	}
	return nil, false
}

// getInterval returns the interval of the given node (with ID e.g. "I(42)").
// The boolean return value indicates success.
func getInterval[N comparable](id graph.ID[N], intervals []*Interval[N]) (*Interval[N], bool) {
	if id.Kind != graph.IntervalNode {
		return nil, false
	}
	return intervals[id.Idx], true
}

// loop returns the nodes of the loop (latch, I.head), marking the loop header
func markNodesInLoop[N comparable](g *graph.Graph[N], head, latch *graph.Node[N], dom *dominator.Tree[N]) []*graph.Node[N] {
	nodes := []*graph.Node[N]{head}
	head.IsLoopNode = true
	head.IsLoopHead = true
	for _, node := range ascReversePostOrder(g.Nodes()) {
		// The loop is formed of all nodes that are between x and y in terms of
		// node numbering.
		if head.Order < node.Order && node.Order <= latch.Order {
			// The nodes belong to the same interval, since the interval header
			// (i.e. x) dominates all nodes of the interval, and in a loop, the
			// loop header node dominates all nodes of the loop. If a node belongs
			// to a different interval, it is not dominated by the loop header
			// node, thus it cannot belong to the same loop.
			if dom.Dominates(head, node) {
				nodes = append(nodes, node)
				node.IsLoopNode = true
			}
		}
		if node.Order > latch.Order {
			break
		}
	}
	return nodes
}

// findLoopKind determines the structural type of a loop based on the control flow properties
// of its header and latch nodes, returning one of PreTestedLoop, PostTestedLoop, or EndlessLoop.
func findLoopKind[N comparable](g *graph.Graph[N], head, latch *graph.Node[N], nodes []*graph.Node[N]) (PrimitiveKind, error) {
	// Special case: self-loop where the header is also the latch
	// This forms a post-tested loop structure (do-while loop)
	if head.ID() == latch.ID() {
		return PostTestedLoop, nil
	}

	headSuccs := g.Successors(head)
	latchSuccs := g.Successors(latch)

	switch len(latchSuccs) {
	// Case: Latch node has 2 outgoing edges (conditional latch)
	case 2:
		switch len(headSuccs) {
		// Case: Header node has 2 outgoing edges (conditional header)
		case 2:
			// If both successors of the header are within the loop,
			// then the loop condition is evaluated at the end (post-tested/do-while loop)
			if contains(nodes, headSuccs[0]) && contains(nodes, headSuccs[1]) {
				return PostTestedLoop, nil
			} else {
				// Otherwise, the loop condition is evaluated at the beginning (pre-tested/while loop)
				return PreTestedLoop, nil
			}
		// Case: Header node has 1 outgoing edge (unconditional header)
		case 1:
			// With unconditional header but conditional latch, this is a post-tested loop
			return PostTestedLoop, nil
		default:
			return None, fmt.Errorf("unsupported %d-way header node", len(headSuccs))
		}
	// Case: Latch node has 1 outgoing edge (unconditional latch)
	case 1:
		switch len(headSuccs) {
		// Case: Header node has 2 outgoing edges (conditional header)
		case 2:
			// With conditional header but unconditional latch, this is a pre-tested loop
			return PreTestedLoop, nil
		// Case: Header node has 1 outgoing edge (unconditional header)
		case 1:
			// With both unconditional header and latch, this forms an endless loop
			return EndlessLoop, nil
		default:
			return None, fmt.Errorf("unsupported %d-way header node", len(headSuccs))
		}
	default:
		return None, fmt.Errorf("unsupported %d-way latching node", len(latchSuccs))
	}
}

// findLoopFollow returns the follow node of the loop (latch, head).
func findLoopFollow[N comparable](g *graph.Graph[N], kind PrimitiveKind, head, latch *graph.Node[N], nodes []*graph.Node[N], dom *dominator.Tree[N]) (*graph.Node[N], error) {
	headSuccs := g.Successors(head)
	latchSuccs := g.Successors(latch)

	switch kind {
	case PreTestedLoop:
		// For a pre-tested loop, we need to identify which successor of the head node
		// is the loop follow (exit) node, and which one leads to the loop body.
		targetNode := latch
		// Walk up the dominator tree from the latch until we find a node that is
		// a direct successor of the head node. This helps identify the branch
		// that leads to the loop body.
		for targetNode.ID() != headSuccs[0].ID() && targetNode.ID() != headSuccs[1].ID() {
			targetNode = dom.DominatorOf(targetNode)
		}

		switch {
		// Case 1: The first successor is inside the loop, meaning the second successor
		// must be the follow node (exit path). We verify this by ensuring:
		// - The first successor is part of the loop nodes
		// - The second successor is not the latch node itself
		// - The dominant path from latch doesn't lead to the second successor
		case contains(nodes, headSuccs[0]) && headSuccs[1] != latch && targetNode.ID() != headSuccs[1].ID():
			return headSuccs[1], nil // The second successor is the loop follow node

		// Case 2: The second successor is inside the loop, meaning the first successor
		// must be the follow node (exit path)
		case contains(nodes, headSuccs[1]) && headSuccs[0] != latch:
			return headSuccs[0], nil // The first successor is the loop follow node

		default:
			// If we can't determine the follow node with the above rules,
			// the loop structure might be abnormal or complex
			return nil, errors.New("unable to locate follow node of pre-tested loop")
		}

	case PostTestedLoop:
		switch {
		// If the first successor of the latch node is inside the loop,
		// the second successor must be the exit point (follow node)
		case contains(nodes, latchSuccs[0]):
			return latchSuccs[1], nil

		// If the second successor of the latch node is inside the loop,
		// the first successor must be the exit point (follow node)
		case contains(nodes, latchSuccs[1]):
			return latchSuccs[0], nil

		default:
			return nil, errors.New("unable to locate follow node of post-tested loop")
		}

	case EndlessLoop:
		// For endless loops, we need to find an exit point by examining conditional branches
		// Initial value is maximum integer to ensure any valid node has lower order
		followRevPostNum := math.MaxInt64
		var follow *graph.Node[N]

		// Examine all 2-way conditional nodes within the loop to find potential exit points
		for _, n := range nodes {
			nSuccs := g.Successors(n)
			if len(nSuccs) != 2 {
				// Skip nodes that aren't 2-way conditionals
				continue
			}

			switch {
			// If first successor is outside the loop and has lower reverse post order number
			// than our current candidate, it becomes the new follow node candidate
			case !contains(nodes, nSuccs[0]) && nSuccs[0].Order < followRevPostNum:
				followRevPostNum = nSuccs[0].Order
				follow = nSuccs[0]

			// If second successor is outside the loop and has lower reverse post order number
			// than our current candidate, it becomes the new follow node candidate
			case !contains(nodes, nSuccs[1]) && nSuccs[1].Order < followRevPostNum:
				followRevPostNum = nSuccs[1].Order
				follow = nSuccs[1]
			}
		}

		// If we found a valid follow node (exit point)
		if followRevPostNum != math.MaxInt64 {
			return follow, nil
		}

		// No exit point found - this is a truly endless loop
		return nil, nil
	default:
		return nil, errors.New("unsupported loop kind")
	}
}

// StructureTwoWayConditionals structures 2-way conditionals in the given control
// flow graph.
func StructureTwoWayConditionals[N comparable](g *graph.Graph[N], dom *dominator.Tree[N]) []Primitive[N] {
	prims := make([]Primitive[N], 0)
	unresolved := newStack[N]()
	for _, node := range descReversePostOrder(g.Nodes()) {
		if len(g.Successors(node)) == 2 && !node.IsLoopHead && !node.IsLoopLatch {
			var follow *graph.Node[N]
			for _, n := range dom.DominatedBy(node) {
				if len(g.Predecessors(n)) < 2 {
					continue
				}
				if follow == nil || follow.Order < n.Order {
					follow = n
				}
			}
			if follow != nil {
				prim := Primitive[N]{
					Kind:  TwoWayConditional,
					Entry: node.Value,
					Exit:  follow.Value,
					Extra: map[string]N{
						"cond":   node.Value,
						"follow": follow.Value,
					},
				}
				for i := 0; !unresolved.empty(); i++ {
					n := unresolved.pop()
					prim.Body = append(prim.Body, n.Value)
				}
				prims = append(prims, prim)
			} else {
				unresolved.push(node)
			}
		}
	}
	return prims
}
