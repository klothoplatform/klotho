package visualizer

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/klothoplatform/klotho/pkg/cli_config"
	"github.com/klothoplatform/klotho/pkg/core"
	"go.uber.org/zap"
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
var validateTypes = cli_config.EnvVar("KLOTHO_VIZ_VALIDATE_TYPES").GetBool()

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
func (p Plugin) Translate(dag *core.ResourceGraph) ([]core.File, error) {
	api := visApi{client: p.Client}

	if validateTypes {
		types := TypesChecker{DAG: dag}

		if resp, err := api.request(http.MethodGet, `validate-types`, `application/text`, ``, types); err != nil {
			if badStatus, isBadStatus := err.(httpStatusBad); isBadStatus {
				unknowns := strings.ReplaceAll(string(resp), "\n", "\n•  ")
				zap.S().Warnf("Failed to validate all types in visualizer (%d). %s", badStatus, unknowns)
			} else {
				zap.S().With(zap.Error(err)).Warnf("Failed to validate types in visualizer: %v", err)
			}
		}
	}

	spec := &File{
		AppName:  p.AppName,
		Provider: p.Provider,
		DAG:      dag,
	}

	resp, err := api.request(http.MethodPost, `generate-infra-diagram`, "application/yaml", "image/png", spec)
	if err != nil {
		return nil, err
	}

	diagram := &core.RawFile{
		FPath:   "diagram.png",
		Content: resp,
	}

	return []core.File{
		spec,
		diagram,
	}, nil
}

// Translate implements compiler.IaCPlugin - although it's not strictly an IaC plugin, it uses the same API
func (p Plugin) Generate(dag *core.ResourceGraph, filenamePrefix string) ([]core.File, error) {
	api := visApi{client: p.Client}

	if validateTypes {
		types := TypesChecker{DAG: dag}

		if resp, err := api.request(http.MethodGet, `validate-types`, `application/text`, ``, types); err != nil {
			if badStatus, isBadStatus := err.(httpStatusBad); isBadStatus {
				unknowns := strings.ReplaceAll(string(resp), "\n", "\n•  ")
				zap.S().Warnf("Failed to validate all types in visualizer (%d). %s", badStatus, unknowns)
			} else {
				zap.S().With(zap.Error(err)).Warnf("Failed to validate types in visualizer: %v", err)
			}
		}
	}

	spec := &File{
		PathPrefix: fmt.Sprintf("%s-", filenamePrefix),
		AppName:    p.AppName,
		Provider:   p.Provider,
		DAG:        dag,
	}

	resp, err := api.request(http.MethodPost, `generate-infra-diagram`, "application/yaml", "image/png", spec)
	if err != nil {
		return nil, err
	}

	diagram := &core.RawFile{
		FPath:   fmt.Sprintf("%s-diagram.png", filenamePrefix),
		Content: resp,
	}

	return []core.File{
		spec,
		diagram,
	}, nil
}

func (h httpStatusBad) Error() string {
	return fmt.Sprintf("visualizer returned status code %d", h)
}
