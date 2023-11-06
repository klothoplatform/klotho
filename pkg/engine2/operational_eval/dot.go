package operational_eval

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"

	construct "github.com/klothoplatform/klotho/pkg/construct2"
)

const rankSize = 20

func keyAttributes(eval *Evaluator, key Key) map[string]string {
	attribs := make(map[string]string)
	var style []string
	if !key.Ref.Resource.IsZero() {
		attribs["label"] = fmt.Sprintf(`%s\n%s`, key.Ref.Resource, key.Ref.Property)
		attribs["shape"] = "box"
	} else if key.GraphState != "" {
		attribs["label"] = string(key.GraphState)
		attribs["shape"] = "box"
		style = append(style, "dashed")
	} else if key.PathSatisfication != nil {
		attribs["label"] = fmt.Sprintf(`%s\n→ %s`, key.Edge.Source, key.Edge.Target)
		var extra []string
		if key.PathSatisfication.Classification != "" {
			extra = append(extra, fmt.Sprintf("<%s>", key.PathSatisfication.Classification))
		}
		if key.PathSatisfication.AsTarget {
			extra = append(extra, "(target)")
		}
		if len(extra) > 0 {
			attribs["label"] += `\n` + strings.Join(extra, " ")
		}

		attribs["shape"] = "parallelogram"
		style = append(style, "dashed")
	} else if key.Edge != (construct.SimpleEdge{}) {
		attribs["label"] = fmt.Sprintf(`%s\n→ %s`, key.Edge.Source, key.Edge.Target)
		attribs["shape"] = "parallelogram"
	} else {
		attribs["label"] = fmt.Sprintf(`%s\n(UNKOWN)`, key)
		attribs["color"] = "#fc8803"
	}
	if eval.errored.Contains(key) {
		style = append(style, "filled")
		attribs["fillcolor"] = "#e87b7b"
	}
	attribs["style"] = strings.Join(style, ",")
	return attribs
}

func attributesToString(attribs map[string]string) string {
	var keys []string
	for k := range attribs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var list []string
	for _, k := range keys {
		list = append(list, fmt.Sprintf(`%s="%s"`, k, attribs[k]))
	}
	return strings.Join(list, ", ")
}

type evalRank struct {
	Unevaluated bool
	Rank        int
	SubRanks    [][]Key
}

func toRanks(eval *Evaluator) ([]evalRank, error) {
	ranks := make([]evalRank, len(eval.evaluatedOrder), len(eval.evaluatedOrder)+1)

	pred, err := eval.graph.PredecessorMap()
	if err != nil {
		return nil, err
	}
	adj, err := eval.graph.AdjacencyMap()
	if err != nil {
		return nil, err
	}
	for i, keys := range eval.evaluatedOrder {
		ranks[i] = evalRank{Rank: i}
		rank := &ranks[i]

		if len(keys) > rankSize {
			// split large ranks into smaller ones
			var noDeps []Key
			var onlyDownstream []Key
			var hasUpstream []Key
			for key := range keys {
				switch {
				case len(pred[key]) == 0 && len(adj[key]) == 0:
					noDeps = append(noDeps, key)

				case len(pred[key]) == 0:
					onlyDownstream = append(onlyDownstream, key)

				default:
					hasUpstream = append(hasUpstream, key)
				}
			}
			if len(onlyDownstream) > 0 {
				for i := 0; i < len(onlyDownstream); i += rankSize {
					rank.SubRanks = append(rank.SubRanks, onlyDownstream[i:min(i+rankSize, len(onlyDownstream))])
				}
			}
			for i := 0; i < len(noDeps); i += rankSize {
				rank.SubRanks = append(rank.SubRanks, noDeps[i:min(i+rankSize, len(noDeps))])
			}
			if len(hasUpstream) > 0 {
				for i := 0; i < len(hasUpstream); i += rankSize {
					rank.SubRanks = append(rank.SubRanks, hasUpstream[i:min(i+rankSize, len(hasUpstream))])
				}
			}
		} else {
			rank.SubRanks = [][]Key{keys.ToSlice()}
		}
	}
	var unevaluated []Key
	for key := range pred {
		evaluated := false
		for _, keys := range eval.evaluatedOrder {
			if keys.Contains(key) {
				evaluated = true
				break
			}
		}
		if !evaluated {
			unevaluated = append(unevaluated, key)
		}
	}
	if len(unevaluated) > 0 {
		rank := evalRank{
			Unevaluated: true,
			Rank:        len(ranks),
		}
		for i := 0; i < len(unevaluated); i += rankSize {
			rank.SubRanks = append(rank.SubRanks, unevaluated[i:min(i+rankSize, len(unevaluated))])
		}

		ranks = append(ranks, rank)
	}
	return ranks, nil
}

func graphToClusterDOT(eval *Evaluator, out io.Writer) error {
	var errs error
	printf := func(s string, args ...any) {
		_, err := fmt.Fprintf(out, s, args...)
		errs = errors.Join(errs, err)
	}

	printf(`strict digraph {
  rankdir = "BT"
	ranksep = 4
	newrank = true
	concentrate = true
`)

	ranks, err := toRanks(eval)
	if err != nil {
		return err
	}

	adj, err := eval.graph.AdjacencyMap()
	if err != nil {
		return err
	}

	for _, evalRank := range ranks {
		rank := evalRank.Rank
		printf("  subgraph cluster_%d {\n", rank)
		if evalRank.Unevaluated {
			printf(`    label = "Unevaluated"
			style=filled
			color="#e3cf9d"`)
		} else {
			printf("    label = \"Evaluation Order %d\"\n", rank)
		}
		printf("    labelloc=b\n")
		for i, subrank := range evalRank.SubRanks {
			printf("    {")
			if evalRank.Unevaluated {
				printf("\n")
			} else {
				printf("rank=same\n")
			}
			for _, key := range subrank {
				attribs := keyAttributes(eval, key)
				attribs["group"] = fmt.Sprintf("group%d.%d", rank, i)
				printf("    %q [%s]\n", key, attributesToString(attribs))

				for tgt := range adj[key] {
					printf("  %q -> %q\n", key, tgt)
				}
			}
			printf("    }\n")
			if i == 0 {
				if rank > 0 {
					prevRank := ranks[rank-1]
					lastSubrank := prevRank.SubRanks[len(prevRank.SubRanks)-1]
					printf("    %q -> %q [style=invis, weight=10]\n", subrank[0], lastSubrank[0])
					printf("    %q -> %q [style=invis, weight=10]\n", subrank[len(subrank)-1], lastSubrank[len(lastSubrank)-1])
				}
			} else {
				lastSubrank := evalRank.SubRanks[i-1]
				printf("    %q -> %q [style=invis, weight=10]\n", subrank[0], lastSubrank[0])
				printf("    %q -> %q [style=invis, weight=10]\n", subrank[len(subrank)-1], lastSubrank[len(lastSubrank)-1])
			}
		}
		printf("  }\n")
	}

	printf("}\n")

	return errs
}

func graphToDOT(eval *Evaluator, out io.Writer) error {
	var errs error
	printf := func(s string, args ...any) {
		_, err := fmt.Fprintf(out, s, args...)
		errs = errors.Join(errs, err)
	}

	printf(`strict digraph {
  rankdir = "BT"
	ranksep = 1
`)
	adj, err := eval.graph.AdjacencyMap()
	if err != nil {
		return err
	}

	evalOrder := make(map[Key]int)
	for i, keys := range eval.evaluatedOrder {
		for key := range keys {
			evalOrder[key] = i
		}
	}

	for src, a := range adj {
		attribs := keyAttributes(eval, src)
		order, hasOrder := evalOrder[src]
		if hasOrder {
			attribs["label"] = fmt.Sprintf("[%d] %s", order, attribs["label"])
		}
		printf("  %q [%s]\n", src, attributesToString(attribs))
		for tgt := range a {
			printf("  %q -> %q\n", src, tgt)
		}
	}
	printf("}\n")
	return errs
}
