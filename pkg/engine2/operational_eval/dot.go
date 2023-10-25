package operational_eval

import (
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
)

func keyAttributes(eval *Evaluator, key Key) map[string]string {
	attribs := make(map[string]string)
	if !key.Ref.Resource.IsZero() {
		attribs["label"] = fmt.Sprintf(`%s\n%s`, key.Ref.Resource, key.Ref.Property)
		attribs["shape"] = "box"
	} else if key.GraphState != "" {
		attribs["label"] = key.GraphState
		attribs["shape"] = "box"
		attribs["style"] = "dashed"
	} else {
		attribs["label"] = fmt.Sprintf(`%s\n→ %s`, key.Edge.Source, key.Edge.Target)
		attribs["shape"] = "parallelogram"
	}
	evaluated := false
	for _, eval := range eval.evaluatedOrder {
		if eval.Contains(key) {
			evaluated = true
			break
		}
	}
	if !evaluated {
		attribs["style"] = "filled"
		attribs["fillcolor"] = "#e87b7b"
	}
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
	Rank     int
	SubRanks [][]Key
}

func toRanks(eval *Evaluator) ([]evalRank, error) {
	ranks := make([]evalRank, len(eval.evaluatedOrder))

	pred, err := eval.graph.PredecessorMap()
	if err != nil {
		return nil, err
	}
	for i, keys := range eval.evaluatedOrder {
		ranks[i] = evalRank{Rank: i}
		rank := &ranks[i]

		if len(keys) > 20 {
			// split large ranks into smaller ones
			var noDeps []Key
			var hasDeps []Key
			for key := range keys {
				if len(pred[key]) == 0 {
					noDeps = append(noDeps, key)
				} else {
					hasDeps = append(hasDeps, key)
				}
			}
			for i := 0; i < len(noDeps); i += 20 {
				rank.SubRanks = append(rank.SubRanks, noDeps[i:min(i+20, len(noDeps))])
			}
			rank.SubRanks = append(rank.SubRanks, hasDeps)
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
		ranks = append(ranks, evalRank{
			Rank:     len(ranks),
			SubRanks: [][]Key{unevaluated},
		})
	}
	return ranks, nil
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
	newrank = true
`)

	ranks, err := toRanks(eval)
	if err != nil {
		return err
	}

	for _, evalRank := range ranks {
		rank := evalRank.Rank
		printf("  subgraph cluster_%d {\n", rank)
		printf("    label = \"Evaluation Order %d\"\n", rank)
		printf("    labelloc=b\n")
		for i, subrank := range evalRank.SubRanks {
			printf("    { rank=same\n")
			for _, key := range subrank {
				attribs := keyAttributes(eval, key)
				attribs["group"] = fmt.Sprintf("group%d-%d", rank, i)
				printf("    %q [%s]\n", key, attributesToString(attribs))
			}
			printf("    }\n")
			if i == 0 {
				if rank > 0 {
					prevRank := ranks[rank-1]
					lastSubrank := prevRank.SubRanks[len(prevRank.SubRanks)-1]
					printf("    %q -> %q [style=invis, weight=10]\n", subrank[0], lastSubrank[0])
				}
			} else {
				printf("    %q -> %q [style=invis, weight=10]\n", subrank[0], evalRank.SubRanks[i-1][0])
			}
		}
		printf("  }\n")
	}

	adj, err := eval.graph.AdjacencyMap()
	if err != nil {
		return err
	}
	for src, a := range adj {
		for tgt := range a {
			printf("  %q -> %q\n", src, tgt)
		}
	}
	printf("}\n")

	return errs
}
