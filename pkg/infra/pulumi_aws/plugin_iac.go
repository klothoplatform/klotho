package pulumi_aws

import (
	"io"
	"os"
	"strings"

	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/logging"
	"go.uber.org/zap"
)

type Plugin struct {
	Config *config.Application
}

func (p Plugin) Name() string { return "Pulumi:AWS" }

func (p Plugin) Translate(cloudGraph *core.ResourceGraph) ([]core.File, error) {
	return nil, nil
}

type StackFile struct {
	core.RawFile
	AppName string
}

func (s *StackFile) WriteTo(w io.Writer) (int64, error) {
	if f, ok := w.(*os.File); ok {
		outDir := strings.TrimSuffix(f.Name(), s.FPath)
		zap.L().
			With(logging.FileField(s)).Sugar().
			Infof("Make sure to run `pulumi config set aws:region YOUR_REGION --cwd '%s' -s '%s'` to configure the target AWS region.", outDir, s.AppName)
	}
	return s.RawFile.WriteTo(w)
}

func (s *StackFile) Overwrite(f *os.File) bool {
	if f != nil {
		zap.L().Debug("Detected existing Pulumi stack yaml, skipping")
		return false
	}
	return true
}
