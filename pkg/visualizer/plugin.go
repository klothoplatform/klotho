package visualizer

import (
	"bytes"
	"fmt"
	"io"
	"net/http"

	"github.com/klothoplatform/klotho/pkg/cli_config"
	construct "github.com/klothoplatform/klotho/pkg/construct2"
	klotho_io "github.com/klothoplatform/klotho/pkg/io"
)

type Plugin struct {
	AppName  string
	Provider string
	Client   *http.Client
}

type (
	visApi struct {
		client *http.Client
		buf    bytes.Buffer
	}

	httpStatusBad int
)

// Name implements compiler.Plugin
func (p Plugin) Name() string {
	return "visualizer"
}

var visualizerBaseUrlEnv = cli_config.EnvVar("KLOTHO_VIZ_URL_BASE")
var visualizerBaseUrl = visualizerBaseUrlEnv.GetOr("https://viz.klo.dev")

func (a *visApi) request(method string, path string, contentType string, accept string, f io.WriterTo) ([]byte, error) {
	a.buf.Reset()
	_, err := f.WriteTo(&a.buf)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(method, visualizerBaseUrl+`/api/v1/`+path, &a.buf)
	if err != nil {
		return nil, err
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if accept != "" {
		req.Header.Set("Accept", accept)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	a.buf.Reset()
	_, err = a.buf.ReadFrom(resp.Body)
	if err != nil && resp.StatusCode < 200 || resp.StatusCode > 299 {
		err = httpStatusBad(resp.StatusCode)
	}
	return a.buf.Bytes(), err
}

// Translate implements compiler.IaCPlugin - although it's not strictly an IaC plugin, it uses the same API
func (p Plugin) Translate(dag construct.Graph) ([]klotho_io.File, error) {
	api := visApi{client: p.Client}

	var err error
	spec := &File{
		AppName:  p.AppName,
		Provider: p.Provider,
	}
	spec.Graph, err = ConstructToVis(dag)
	if err != nil {
		return nil, err
	}

	resp, err := api.request(http.MethodPost, `generate-infra-diagram`, "application/yaml", "image/png", spec)
	if err != nil {
		return nil, err
	}

	diagram := &klotho_io.RawFile{
		FPath:   "diagram.png",
		Content: resp,
	}

	return []klotho_io.File{
		spec,
		diagram,
	}, nil
}

// Translate implements compiler.IaCPlugin - although it's not strictly an IaC plugin, it uses the same API
func (p Plugin) Generate(dag construct.Graph, filenamePrefix string) ([]klotho_io.File, error) {
	api := visApi{client: p.Client}

	var err error
	spec := &File{
		FilenamePrefix: fmt.Sprintf("%s-", filenamePrefix),
		AppName:        p.AppName,
		Provider:       p.Provider,
	}
	spec.Graph, err = ConstructToVis(dag)
	if err != nil {
		return nil, err
	}

	resp, err := api.request(http.MethodPost, `generate-infra-diagram`, "application/yaml", "image/png", spec)
	if err != nil {
		return nil, err
	}

	diagram := &klotho_io.RawFile{
		FPath:   fmt.Sprintf("%s-diagram.png", filenamePrefix),
		Content: resp,
	}

	return []klotho_io.File{
		spec,
		diagram,
	}, nil
}

func (h httpStatusBad) Error() string {
	return fmt.Sprintf("visualizer returned status code %d", h)
}
