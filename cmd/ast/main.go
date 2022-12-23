package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/lang"
	"github.com/klothoplatform/klotho/pkg/lang/javascript"
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

var root = &cobra.Command{
	Use:  "klo-ast",
	RunE: run,
}

var cfg struct {
	verbose bool
	file    string
	outFile string
}

func main() {
	flags := root.Flags()

	flags.BoolVarP(&cfg.verbose, "verbose", "v", false, "Verbose flag")
	flags.StringVarP(&cfg.file, "file", "f", "", "File to query")

	err := root.Execute()
	if err != nil {
		panic(err)
	}
}

func run(cmd *cobra.Command, args []string) error {
	var zapCfg zap.Config
	if cfg.verbose {
		zapCfg = zap.NewDevelopmentConfig()
	} else {
		zapCfg = zap.NewProductionConfig()
	}
	zapCfg.Encoding = "console"

	z, err := zapCfg.Build()
	if err != nil {
		return err
	}
	defer z.Sync() // nolint:errcheck
	defer zap.ReplaceGlobals(z)()

	if cfg.file == "" {
		return errors.New("no file specified")
	}

	file, err := os.Open(cfg.file)
	if err != nil {
		return err
	}
	defer file.Close()

	jsFile, err := javascript.NewFile(cfg.file, file)
	if err != nil {
		return err
	}

	if len(args) > 0 {
		q := []byte(args[0])

		query, err := sitter.NewQuery(q, jsFile.Language.Sitter)
		if err != nil {
			return err
		}

		queryCursor := sitter.NewQueryCursor()
		queryCursor.Exec(query, jsFile.Tree().RootNode())

		for matchNum := 0; ; matchNum++ {
			match, found := queryCursor.NextMatch()
			if !found {
				break
			}
			for i, capture := range match.Captures {
				fmt.Printf("[%d.%d] ", matchNum, i)
				err := lang.WriteAST(capture.Node, jsFile.Program(), os.Stdout)
				if err != nil {
					return err
				}
			}
		}
	} else {
		err := lang.WriteAST(jsFile.Tree().RootNode(), jsFile.Program(), os.Stdout)
		if err != nil {
			return err
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		annots, err := javascript.Language.CapabilityFinder.FindAllCapabilities(jsFile)
		if err != nil {
			return err
		}
		var caps []core.Annotation
		for _, v := range annots {
			caps = append(caps, *v)
		}
		sort.Slice(caps, func(i, j int) bool {
			startI := 0
			if caps[i].Node != nil {
				startI = int(caps[i].Node.StartByte())
			}
			startJ := 0
			if caps[j].Node != nil {
				startJ = int(caps[j].Node.StartByte())
			}
			return startI < startJ
		})
		return enc.Encode(caps)
	}

	return nil
}
