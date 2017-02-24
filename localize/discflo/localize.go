package discflo

import (
	"fmt"
	"math/rand"
	"sort"
)

import (
	"github.com/timtadh/data-structures/errors"
)

import (
	"github.com/timtadh/dynagrok/localize/lattice"
	"github.com/timtadh/dynagrok/localize/lattice/subgraph"
	"github.com/timtadh/dynagrok/localize/test"
	"github.com/timtadh/dynagrok/localize/stat"
)

// todo
// - make it possible to compute a statistical measure on a subgraph
// - use a subgraph measure to guide a discriminative search
// - make the measure statisfy downward closure?
//         (a < b) --> (m(a) >= m(b))
// - read the leap search paper again


type SearchNode struct {
	Node  *lattice.Node
	Score float64
}

func (s *SearchNode) String() string {
	return fmt.Sprintf("%v %v", s.Score, s.Node)
}

func Localize(tests []*test.Testcase, score Score, lat *lattice.Lattice) error {
	WALKS := 500
	nodes := make([]*SearchNode, 0, WALKS)
	seen := make(map[string]bool, WALKS)
	for i := 0; i < WALKS; i++ {
		n := Walk(score, lat)
		if n.Node.SubGraph == nil || len(n.Node.SubGraph.E) < 2 {
			continue
		}
		if false {
			errors.Logf("DEBUG", "found %d %v", i, n)
		}
		label := string(n.Node.SubGraph.Label())
		if !seen[label] {
			nodes = append(nodes, n)
			seen[label] = true
		}
	}
	if len(nodes) == 0 {
		fmt.Println("no graphs")
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Score > nodes[j].Score
	})
	colors := make(map[int][]*SearchNode)
	for i := 0; i < 10 && i < len(nodes); i++ {
		for j := range nodes[i].Node.SubGraph.V {
			colors[nodes[i].Node.SubGraph.V[j].Color] = append(colors[nodes[i].Node.SubGraph.V[j].Color], nodes[i])
		}
		fmt.Println(nodes[i])
		fmt.Printf("------------ ranks %d ----------------\n", i)
		fmt.Println(RankNodes(score, lat, nodes[i].Node.SubGraph))
		fmt.Println("--------------------------------------")
		for count := 0; count < len(tests) ; count++ {
			j := rand.Intn(len(tests))
			t := tests[j]
			min, err := t.Minimize(lat, nodes[i].Node.SubGraph)
			if err != nil {
				return err
			}
			if min == nil {
				continue
			}
			fmt.Printf("------------ min test %d %d ----------\n", i, j)
			fmt.Println(min)
			fmt.Println("--------------------------------------")
			break
		}
	}
	fmt.Println(RankColors(score, lat, colors))
	return nil
}

func RankNodes(score Score, lat *lattice.Lattice, sg *subgraph.SubGraph) stat.Result {
	result := make(stat.Result, 0, len(sg.V))
	for i := range sg.V {
		color := sg.V[i].Color
		vsg := subgraph.Build(1, 0).FromVertex(color).Build()
		embIdxs := lat.Fail.ColorIndex[color]
		embs := make([]*subgraph.Embedding, 0, len(embIdxs))
		for _, embIdx := range embIdxs {
			embs = append(embs, subgraph.StartEmbedding(subgraph.VertexEmbedding{SgIdx: 0, EmbIdx: embIdx}))
		}
		n := lattice.NewNode(lat, vsg, embs)
		s := score(lat, n)
		result = append(result, stat.Location{
			lat.Positions[color],
			lat.FnNames[color],
			lat.BBIds[color],
			s,
		})
	}
	result.Sort()
	return result
}

func RankColors(score Score, lat *lattice.Lattice, colors map[int][]*SearchNode) stat.Result {
	result := make(stat.Result, 0, len(colors))
	for color, searchNodes := range colors {
		vsg := subgraph.Build(1, 0).FromVertex(color).Build()
		embIdxs := lat.Fail.ColorIndex[color]
		embs := make([]*subgraph.Embedding, 0, len(embIdxs))
		for _, embIdx := range embIdxs {
			embs = append(embs, subgraph.StartEmbedding(subgraph.VertexEmbedding{SgIdx: 0, EmbIdx: embIdx}))
		}
		colorNode := lattice.NewNode(lat, vsg, embs)
		colorScore := score(lat, colorNode)
		var s float64
		for _, sn := range searchNodes {
			s += sn.Score
		}
		s = (colorScore * s) / float64(len(searchNodes))
		result = append(result, stat.Location{
			lat.Positions[color],
			lat.FnNames[color],
			lat.BBIds[color],
			s,
		})
	}
	result.Sort()
	return result
}

func Walk(score Score, lat *lattice.Lattice) (*SearchNode) {
	cur := &SearchNode{
		Node: lat.Root(),
		Score: -100000000,
	}
	i := 0
	prev := cur
	for cur != nil {
		if false {
			errors.Logf("DEBUG", "cur %v", cur)
		}
		kids, err := cur.Node.Children()
		if err != nil {
			panic(err)
		}
		prev = cur
		cur = weighted(filterKids(score, cur.Score, lat, kids))
		if i == 1 {
		}
		i++
	}
	return prev
}

func filterKids(score Score, parentScore float64, lat *lattice.Lattice, kids []*lattice.Node) []*SearchNode {
	entries := make([]*SearchNode, 0, len(kids))
	for _, kid := range kids {
		if kid.FIS() < 2 {
			continue
		}
		kidScore := score(lat, kid)
		if kidScore > parentScore {
			entries = append(entries, &SearchNode{kid, kidScore})
		}
	}
	return entries
}

func uniform(slice []*SearchNode) (*SearchNode) {
	if len(slice) > 0 {
		return slice[rand.Intn(len(slice))]
	}
	return nil
}

func weighted(slice []*SearchNode) (*SearchNode) {
	if len(slice) <= 0 {
		return nil
	}
	if len(slice) == 1 {
		return slice[0]
	}
	prs := transitionPrs(slice)
	if prs == nil {
		return nil
	}
	i := weightedSample(prs)
	return slice[i]
}

func transitionPrs(slice []*SearchNode) []float64 {
	weights := make([]float64, 0, len(slice))
	var total float64 = 0
	for _, v := range slice {
		weights = append(weights, v.Score)
		total += v.Score
	}
	if total == 0 {
		return nil
	}
	prs := make([]float64, 0, len(slice))
	for _, wght := range weights {
		prs = append(prs, wght/total)
	}
	return prs
}

func weightedSample(prs []float64) int {
	var total float64
	for _, pr := range prs {
		total += pr
	}
	i := 0
	x := total * (1 - rand.Float64())
	for x > prs[i] {
		x -= prs[i]
		i += 1
	}
	return i
}
