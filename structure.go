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
				follow, err := findLoopFollow(g, kind, head, latch, nodes)
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

// findLoopKind returns the kind of the loop (latch, head).
func findLoopKind[N comparable](g *graph.Graph[N], head, latch *graph.Node[N], nodes []*graph.Node[N]) (PrimitiveKind, error) {
	// Add extra case not present in Cifuentes' for when head == latch.
	if head.ID() == latch.ID() {
		return PostTestedLoop, nil
	}
	headSuccs := g.Successors(head)
	latchSuccs := g.Successors(latch)
	switch len(latchSuccs) {
	// if (nodeType(y) == 2-way)
	case 2:
		switch len(headSuccs) {
		// if (nodeType(x) == 2-way)
		case 2:
			// if (outEdge(x, 1) \in nodesInLoop \land (outEdge(x, 2) \in nodesInLoop)
			if contains(nodes, headSuccs[0]) && contains(nodes, headSuccs[1]) {
				// loopType(x) = Post_Tested.
				return PostTestedLoop, nil
			} else {
				// loopType(x) = Pre_Tested.
				return PreTestedLoop, nil
			}
		// 1-way header node.
		case 1:
			// loopType(x) = Post_Tested.
			return PostTestedLoop, nil
		default:
			return None, fmt.Errorf("unsupported %d-way header node", len(headSuccs))
		}
	// 1-way latching node.
	case 1:
		switch len(headSuccs) {
		// if nodeType(x) == 2-way
		case 2:
			// loopType(x) = Pre_Tested.
			return PreTestedLoop, nil
		// 1-way header node.
		case 1:
			// loopType(x) = Endless.
			return EndlessLoop, nil
		default:
			return None, fmt.Errorf("unsupported %d-way header node", len(headSuccs))
		}
	default:
		return None, fmt.Errorf("unsupported %d-way latching node", len(latchSuccs))
	}
}

// findLoopFollow returns the follow node of the loop (latch, head).
func findLoopFollow[N comparable](g *graph.Graph[N], kind PrimitiveKind, head, latch *graph.Node[N], nodes []*graph.Node[N]) (*graph.Node[N], error) {
	headSuccs := g.Successors(head)
	latchSuccs := g.Successors(latch)
	switch kind {
	// if (loopType(x) == Pre_Tested)
	case PreTestedLoop:
		switch {
		// if (outEdges(x, 1) \in nodesInLoop)
		case contains(nodes, headSuccs[0]):
			// loopFollow(x) = outEdges(x, 2)
			return headSuccs[1], nil
		case contains(nodes, headSuccs[1]):
			// loopFollow(x) = outEdges(x, 1)
			return headSuccs[0], nil
		default:
			return nil, errors.New("unable to locate follow node of pre-tested loop")
		}
	// else if (loopType(x) == Post_Tested)
	case PostTestedLoop:
		switch {
		// if (outEdges(y, 1) \in nodesInLoop)
		case contains(nodes, latchSuccs[0]):
			// loopFollow(x) = outEdges(y, 2)
			return latchSuccs[1], nil
		case contains(nodes, latchSuccs[1]):
			// loopFollow(x) = outEdges(y, 1)
			return latchSuccs[0], nil
		default:
			return nil, errors.New("unable to locate follow node of post-tested loop")
		}
	// endless loop.
	case EndlessLoop:
		// fol = Max // a large constant.
		followRevPostNum := math.MaxInt64
		var follow *graph.Node[N]
		// for (all 2-way nodes n \in nodesInLoop)
		for _, n := range nodes {
			nSuccs := g.Successors(n)
			if len(nSuccs) != 2 {
				// Skip node as not 2-way conditional.
				continue
			}
			switch {
			// if ((outEdges(n, 1) \not \in nodesInLoop) \land (outEdges(x, 1) < fol))
			case !contains(nodes, nSuccs[0]) && nSuccs[0].Order < followRevPostNum:
				followRevPostNum = nSuccs[0].Order
				follow = nSuccs[0]
			// if ((outEdges(x, 2) \not \in nodesInLoop) \land (outEdges(x, 2) < fol))			}
			case !contains(nodes, nSuccs[1]) && nSuccs[1].Order < followRevPostNum:
				followRevPostNum = nSuccs[1].Order
				follow = nSuccs[1]
			}
		}
		// if (fol != Max)
		if followRevPostNum != math.MaxInt64 {
			// loopFollow(x) = fol
			return follow, nil
		}
		// No follow node located.
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
			// if (\exists n, n = max{i | immedDom(i) = m \land #inEdges(i) >= 2})
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
