package cmd

// Precision and Recall are used by
//    H. Cheng, D. Lo, Y. Zhou, X. Wang, and X. Yan, “Identifying Bug Signatures
//    Using Discriminative Graph Mining,” in Proceedings of the Eighteenth
//    International Symposium on Software Testing and Analysis, 2009, pp.
//    141–152.
//
// Precision refers to the proportion of returned results that highlight the
// bug. Recall refers to the proportion of bugs that can be discovered by the
// returned bug signatures
//
// These metrics are across a whole set of bugs in either a single program or
// multiple programs. So not relevant to this evaluation which focuses on one
// version of one program with one or more bugs.

import (
	"context"
	"fmt"

	"github.com/timtadh/dynagrok/cmd"
	"github.com/timtadh/dynagrok/localize/fault"
	"github.com/timtadh/dynagrok/localize/mine"
	"github.com/timtadh/dynagrok/localize/mine/opts"
	"github.com/timtadh/getopt"
)

func NewEvalParser(c *cmd.Config, o *opts.Options) cmd.Runnable {
	return cmd.Cmd(
		"eval",
		`[options]`,
		`
Evaluate a fault localization method from ground truth

Option Flags
    -h,--help                         Show this message
    -f,--faults=<path>                Path to a fault file.
`,
		"f:",
		[]string{
			"faults=",
		},
		func(r cmd.Runnable, args []string, optargs []getopt.OptArg) ([]string, *cmd.Error) {
			faultsPath := ""
			for _, oa := range optargs {
				switch oa.Opt() {
				case "-f", "--faults":
					faultsPath = oa.Arg()
				}
			}
			if faultsPath == "" {
				return nil, cmd.Errorf(1, "You must supply the `-f` flag and give a path to the faults")
			}
			faults, err := fault.LoadFaults(faultsPath)
			if err != nil {
				return nil, cmd.Err(1, err)
			}
			for _, f := range faults {
				fmt.Println(f)
			}
			eval := func(name string, m *mine.Miner) {
				localized := m.Mine(context.TODO()).Group()
				for _, f := range faults {
					sum := 0
					for gid, group := range localized {
						var first *mine.SearchNode
						var bbid int
						var fnName, pos string
						count := 0
						for _, n := range group {
							for _, v := range n.Node.SubGraph.V {
								b, fn, _ := m.Lattice.Info.Get(v.Color)
								if fn == f.FnName && b == f.BasicBlockId {
									if first == nil {
										bbid, fnName, pos = m.Lattice.Info.Get(v.Color)
										first = n
									}
									count++
									break
								}
							}
						}
						if first != nil {
							fmt.Printf(
								"    %v {\n\tgroup: %v size: %d contained-in: %g,\n\trank: %v,\n\tscore: %v,\n\tfn: %v (%d),\n\tpos: %v,\n\tin: %v\n    }\n",
								name,
								gid, len(group), float64(count)/float64(len(group)),
								float64(sum)+float64(len(group))/2,
								first.Score,
								fnName,
								bbid,
								pos,
								first,
							)
							break
						} else {
							sum += len(group)
						}
					}
				}
			}
			if o.Score == nil {
				for name, score := range mine.Scores {
					m := mine.NewMiner(o.Miner, o.Lattice, score, o.Opts...)
					eval("mine-dsg + "+name, m)
				}
			} else {
				m := mine.NewMiner(o.Miner, o.Lattice, o.Score, o.Opts...)
				eval("mine-dsg + "+o.ScoreName, m)
			}
			return nil, nil
		})
}
