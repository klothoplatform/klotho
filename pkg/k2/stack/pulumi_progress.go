package stack

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/klothoplatform/klotho/pkg/logging"
	"github.com/klothoplatform/klotho/pkg/tui"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/events"
	"github.com/pulumi/pulumi/sdk/v3/go/common/apitype"
	"go.uber.org/zap"
)

type PulumiProgress struct {
	Progress tui.Progress

	complete int
	total    int
}

func (p *PulumiProgress) Write(b []byte) (n int, err error) {
	scan := bufio.NewScanner(bytes.NewReader(b))
	for scan.Scan() {
		line := scan.Text()
		line = strings.TrimSpace(line)
		switch {
		case strings.Contains(line, "creating"), strings.Contains(line, "deleting"):
			if strings.Contains(line, "failed") {
				p.complete++
			} else {
				p.total++
			}

		case strings.Contains(line, "created"), strings.Contains(line, "deleted"):
			p.complete++
		}
	}
	p.Progress.Update("Deploying stack", p.complete, p.total)
	return len(b), scan.Err()
}

func Events(ctx context.Context, action string) chan<- events.EngineEvent {
	ech := make(chan events.EngineEvent)
	go func() {
		log := logging.GetLogger(ctx).Named("pulumi.events").Sugar()
		progress := tui.GetProgress(ctx)
		status := fmt.Sprintf("%s stack", action)

		// resourceStatus tracks each resource's status. The key is the resource's URN and the value is the status.
		// The value is an enum that represents the resource's status:
		// 0. Pending / resource pre event, this just marks which resources we're aware of
		// 1. Refresh complete
		// 2. In progress
		// 3. Done
		resourceStatus := make(map[string]int)

		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		for {
			select {
			case <-ctx.Done():
				return
			case e, ok := <-ech:
				if !ok {
					return
				}
				buf.Reset()
				if err := enc.Encode(e); err != nil {
					log.Error("Failed to encode pulumi event", zap.Error(err))
					continue
				}
				logLine := strings.TrimSpace(buf.String())
				log.Debugf("Pulumi event: %s", logLine)

				switch {
				case e.PreludeEvent != nil:
					progress.UpdateIndeterminate(status)

				case e.ResourcePreEvent != nil:
					e := e.ResourcePreEvent
					if e.Metadata.Op == apitype.OpRefresh {
						resourceStatus[e.Metadata.URN] = 0
					} else {
						resourceStatus[e.Metadata.URN] = 2
					}

				case e.ResOutputsEvent != nil:
					e := e.ResOutputsEvent
					if e.Metadata.Op == apitype.OpRefresh {
						resourceStatus[e.Metadata.URN] = 1
					} else {
						resourceStatus[e.Metadata.URN] = 3
					}
				}

				current, total := 0, 0
				for _, stateCode := range resourceStatus {
					total += 3
					current += stateCode
				}
				if total > 0 {
					progress.Update(status, current, total)
				}
			}
		}
	}()
	return ech
}
