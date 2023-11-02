package kubernetes

// import (
// 	"errors"
// 	"fmt"
// 	"io/fs"

// 	"github.com/klothoplatform/klotho/pkg/config"
// 	construct "github.com/klothoplatform/klotho/pkg/construct2"
// 	"github.com/klothoplatform/klotho/pkg/engine2/solution_context"
// 	kio "github.com/klothoplatform/klotho/pkg/io"
// 	knowledgebase "github.com/klothoplatform/klotho/pkg/knowledge_base2"
// 	"github.com/klothoplatform/klotho/pkg/lang/javascript"
// )

// type Plugin struct {
// 	Config *config.Application
// 	KB     *knowledgebase.KnowledgeBase
// }

// func (p Plugin) Name() string {
// 	return "kubernetes"
// }

// func (p Plugin) Translate(ctx solution_context.SolutionContext) ([]kio.File, error) {
// 	return []kio.File{indexTs, packageJson, pulumiYaml, pulumiStack, tsConfig}, errs
// }
