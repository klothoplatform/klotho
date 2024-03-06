package code

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"

	"go.lsp.dev/jsonrpc2"
	lsp "go.lsp.dev/protocol"
	"go.uber.org/zap"
)

type (
	LSP struct {
		ServerProcess *os.Process
		Server        lsp.Server
		Ctx           context.Context
	}

	lspIO struct {
		io.Reader
		io.Writer
		Closer func() error
	}
)

func NewLSP(ctx context.Context, lspCmd string, cmdStderr io.Writer) (*LSP, error) {
	log := zap.L().Named(fmt.Sprintf("lsp/%s", lspCmd))

	cmd := exec.CommandContext(ctx, lspCmd, "-vv")
	if cmd.Err != nil {
		return nil, fmt.Errorf("failed to create command: %w", cmd.Err)
	}
	send, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	cmd.Stderr = cmdStderr

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	log.Sugar().Infof("started")

	lio := &lspIO{out, send, send.Close}
	stream := jsonrpc2.NewStream(lio)
	clCtx, _, server := lsp.NewClient(ctx, &client{Log: log.Named("client")}, stream, log)

	reqCtx, reqCtxCancel := context.WithTimeout(clCtx, 30*time.Second)
	defer reqCtxCancel()

	res, err := server.Initialize(reqCtx, &lsp.InitializeParams{
		ProcessID: int32(os.Getpid()),
		ClientInfo: &lsp.ClientInfo{
			Name: "klotho",
		},
		Capabilities: lsp.ClientCapabilities{
			Workspace: &lsp.WorkspaceClientCapabilities{WorkspaceFolders: true},
			Window: &lsp.WindowClientCapabilities{
				ShowDocument: &lsp.ShowDocumentClientCapabilities{Support: false},
			},
			General: &lsp.GeneralClientCapabilities{
				RegularExpressions: &lsp.RegularExpressionsClientCapabilities{Engine: "re2"},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize server: %w", err)
	}
	log.Sugar().Infof("initialized: %s", JsonString{res})

	return &LSP{
		ServerProcess: cmd.Process,
		Server:        server,
		Ctx:           clCtx,
	}, nil
}

func (lio lspIO) Close() error {
	return lio.Closer()
}

type JsonString struct {
	V any
}

func (j JsonString) String() string {
	s, err := json.Marshal(j.V)
	if err != nil {
		return fmt.Sprintf("%+v (error while marshalling: %s)", j.V, err)
	}
	return string(s)
}

type client struct {
	Log *zap.Logger
}

func (c *client) Progress(ctx context.Context, params *lsp.ProgressParams) (err error) {
	return nil
}
func (c *client) WorkDoneProgressCreate(ctx context.Context, params *lsp.WorkDoneProgressCreateParams) (err error) {
	return nil
}
func (c *client) LogMessage(ctx context.Context, params *lsp.LogMessageParams) (err error) {
	return c.ShowMessage(ctx, &lsp.ShowMessageParams{Message: params.Message, Type: params.Type})
}
func (c *client) PublishDiagnostics(ctx context.Context, params *lsp.PublishDiagnosticsParams) (err error) {
	return nil
}
func (c *client) ShowMessage(ctx context.Context, params *lsp.ShowMessageParams) (err error) {
	switch params.Type {
	case lsp.MessageTypeError:
		c.Log.Error(params.Message)
	case lsp.MessageTypeWarning:
		c.Log.Warn(params.Message)
	case lsp.MessageTypeInfo:
		c.Log.Info(params.Message)
	case lsp.MessageTypeLog:
		c.Log.Debug(params.Message)
	}
	return nil
}
func (c *client) ShowMessageRequest(ctx context.Context, params *lsp.ShowMessageRequestParams) (result *lsp.MessageActionItem, err error) {
	return nil, c.ShowMessage(ctx, &lsp.ShowMessageParams{Message: params.Message, Type: params.Type})
}
func (c *client) Telemetry(ctx context.Context, params interface{}) (err error) {
	return nil
}
func (c *client) RegisterCapability(ctx context.Context, params *lsp.RegistrationParams) (err error) {
	return nil
}
func (c *client) UnregisterCapability(ctx context.Context, params *lsp.UnregistrationParams) (err error) {
	return nil
}
func (c *client) ApplyEdit(ctx context.Context, params *lsp.ApplyWorkspaceEditParams) (result bool, err error) {
	return false, nil
}
func (c *client) Configuration(ctx context.Context, params *lsp.ConfigurationParams) (result []interface{}, err error) {
	return nil, nil
}
func (c *client) WorkspaceFolders(ctx context.Context) (result []lsp.WorkspaceFolder, err error) {
	return nil, nil
}
