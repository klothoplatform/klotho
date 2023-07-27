package topology

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gojek/heimdall/v7/httpclient"
	"github.com/klothoplatform/klotho/pkg/config"
	"github.com/klothoplatform/klotho/pkg/core"
	execunit "github.com/klothoplatform/klotho/pkg/exec_unit"
	"go.uber.org/zap"
)

const baseIconUrl = "https://raw.githubusercontent.com/mingrammer/diagrams/master/resources/"

const defaultVisURL = "http://viz-v0.klo.dev/render"

type Plugin struct {
	Config *config.Application
}

func (p Plugin) Name() string { return "Topology" }

func (p Plugin) Transform(result *core.CompilationResult, deps *core.Dependencies) error {

	var topology core.TopologyData

	switch p.Config.Provider {
	case core.ProviderAWS, core.ProviderGCP, core.ProviderAzure:
		// Valid provider
	default:
		return fmt.Errorf("invalid provider: %s", p.Config.Provider)
	}

	for _, r := range result.Resources() {
		switch r.Key().Kind {
		case core.InfraAsCodeKind, core.InputFilesKind, execunit.FileDependenciesResourceKind:
			continue
		}
		icons, edges := p.generateIconsAndEdges(r, deps.Downstream(r.Key()))
		topology.IconData = append(topology.IconData, icons...)
		topology.EdgeData = append(topology.EdgeData, edges...)
	}

	vizURL := os.Getenv("VIZ_URL")
	if vizURL == "" {
		vizURL = defaultVisURL
	}
	var image []byte
	if strings.ToLower(vizURL) != "disable" {
		diagramPlan := p.generateImageString(topology)
		var err error
		image, err = createViz(vizURL, diagramPlan)
		if err != nil {
			return err
		}
	}

	resource := core.NewTopology(p.Config.AppName, topology, image)
	result.Add(resource)

	return nil
}

func (p Plugin) generateIconsAndEdges(resource core.CloudResource, dependencies []core.ResourceKey) ([]core.TopologyIconData, []core.TopologyEdgeData) {
	icons := make([]core.TopologyIconData, 0)
	edges := make([]core.TopologyEdgeData, 0)
	zap.S().Named("topology").Debugf("%s dependencies = %v", resource.Key(), dependencies)

	icons = append(icons, core.TopologyIconData{
		ID:    generateIconID(resource.Key()),
		Title: resource.Key().Name,
		Image: p.getImagePath(resource),
		Kind:  resource.Key().Kind,
		Type:  p.Config.GetResourceType(resource),
	})

	for _, dependency := range dependencies {
		edge := core.TopologyEdgeData{
			Source: generateIconID(resource.Key()),
			Target: generateIconID(dependency),
		}
		edges = append(edges, edge)
	}

	return icons, edges
}

func generateIconID(resource core.ResourceKey) string {
	return fmt.Sprintf("%s_%s", resource.Name, resource.Kind)
}
func (p Plugin) getImagePath(resource core.CloudResource) string {
	imgPath, _ := core.DiagramEntityToImgPath.Get(resource.Key().Kind, p.Config.GetResourceType(resource), p.Config.Provider)
	return baseIconUrl + imgPath
}

func (p Plugin) generateImageString(t core.TopologyData) string {
	var result strings.Builder
	for _, i := range t.IconData {
		code, _ := core.DiagramEntityToCode.Get(i.Kind, i.Type, p.Config.Provider)
		if code == "" {
			continue
		}

		code = fmt.Sprintf(code, i.Title)
		fmt.Fprintf(&result, "nodeList[\"%s\"] = %s\n", i.ID, code)
	}

	for _, e := range t.EdgeData {
		fmt.Fprintf(&result, "nodeList[\"%s\"] >> nodeList[\"%s\"]\n", e.Source, e.Target)
	}

	return result.String()
}

var httpClient = httpclient.NewClient(httpclient.WithHTTPTimeout(20 * time.Second))

func createViz(vizURL string, diagramPlan string) ([]byte, error) {
	req, err := http.NewRequest("POST", vizURL, bytes.NewBufferString(diagramPlan))
	if err != nil {
		return nil, err
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode > 299 {
		return nil, fmt.Errorf("unexpected status code: %s", res.Status)
	}

	decoder := base64.NewDecoder(base64.StdEncoding, res.Body)

	var b bytes.Buffer
	_, err = b.ReadFrom(decoder)
	if err != nil {
		return nil, err
	}

	return b.Bytes(), nil

}
