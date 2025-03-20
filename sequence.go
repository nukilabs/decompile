package decompile

import "github.com/nukilabs/decompile/graph"

func DerivedSequence[N comparable](g *graph.Graph[N]) ([]*graph.Graph[N], [][]*Interval[N]) {
	graphs := make([]*graph.Graph[N], 0)
	graphs = append(graphs, g)
	intervals := make([][]*Interval[N], 0)
	intervals = append(intervals, Intervals(g))

	root := g.Root()

	count := 0
	for i := 0; ; i++ {
		prevGraph := graphs[i]
		newGraph := graph.New[N]()

		// Make each interval of G^{i-1} a node in G^i.
		nodes := make([]*graph.Node[N], 0)
		for _, interval := range intervals[i] {
			node := newGraph.Interval(count)
			nodes = append(nodes, node)
			if root.ID() == interval.head.ID() {
				newGraph.SetRoot(node)
				root = node
			}
			count++
		}

		// The collapsed node n of an interval I(h) has the immediate predecessors
		// of h not part of the interval I(h).
		for j, interval := range intervals[i] {
			node := nodes[j]
			for _, pred := range prevGraph.Predecessors(interval.head) {
				if interval.Contains(pred) {
					continue
				}

				for k, predInterval := range intervals[i] {
					if predInterval.Contains(pred) {
						newGraph.SetEdge(nodes[k], node)
					}
				}
			}
		}

		// The collapsed node n of an interval I(h) has the immediate successors
		// of the exit nodes of I(h) not part of the interval I(h).
		for j, interval := range intervals[i] {
			node := nodes[j]
			for _, succ := range prevGraph.Successors(interval.head) {
				if interval.Contains(succ) {
					continue
				}

				for k, succInterval := range intervals[i] {
					if succInterval.Contains(succ) {
						newGraph.SetEdge(node, nodes[k])
					}
				}
			}
		}

		if newGraph.Len() == prevGraph.Len() {
			break
		}

		graphs = append(graphs, newGraph)
		intervals = append(intervals, Intervals(newGraph))
	}

	return graphs, intervals
}
