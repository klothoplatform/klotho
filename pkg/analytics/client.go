package analytics

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/klothoplatform/klotho/pkg/auth"

	"github.com/google/uuid"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type (
	Payload struct {
		UserId     string         `json:"id"`
		Event      string         `json:"event"`
		Source     []byte         `json:"source,omitempty"`
		Properties map[string]any `json:"properties"`
	}

	Client struct {
		serverUrlOverride   string
		userId              string
		universalProperties map[string]any
	}

	ErrorHandler interface {
		PrintErr(err error)
	}

	LogLevel string
)

const (
	Panic LogLevel = "panic"
	Error LogLevel = "error"
	Warn  LogLevel = "warn"
	Info  LogLevel = "info"
	Debug LogLevel = "debug"
)

const datadogLogLevel = "_logLevel"
const datadogStatus = "status"

func NewClient() *Client {
	local := GetOrCreateAnalyticsFile()

	client := &Client{}
	client.universalProperties = make(map[string]any)

	// These will get validated in AttachAuthorizations
	client.userId = local.Id
	client.universalProperties["validated"] = false

	client.universalProperties["localId"] = local.Id
	if runUuid, err := uuid.NewRandom(); err == nil {
		client.universalProperties["runId"] = runUuid.String()
	}

	return client
}

func (t *Client) AttachAuthorizations(loginInfo auth.LoginInfo) {
	if loginInfo.Email != "" {
		t.userId = loginInfo.Email
		t.universalProperties["validated"] = loginInfo.EmailVerified
	}
	t.universalProperties["loginMethod"] = loginInfo.Authorizer
}

func (t *Client) Info(event string) {
	t.send(t.createPayload(Info, event))
}

func (t *Client) Warn(event string) {
	t.send(t.createPayload(Warn, event))
}

func (t *Client) Error(event string) {
	t.send(t.createPayload(Error, event))
}

func (p *Payload) addError(err error) {
	p.Properties["error"] = fmt.Sprintf("%+v", err)
}

func (t *Client) AppendProperties(properties map[string]interface{}) {
	for k, v := range properties {
		t.AppendProperty(k, v)
	}
}

func (t *Client) AppendProperty(key string, value any) {
	t.universalProperties[key] = value
}

func (t *Client) UploadSource(source *core.InputFiles) {
	data, err := CompressFiles(source)
	if err != nil {
		zap.S().Warnf("Failed to upload debug bundle. %v", err)
		return
	}
	p := t.createPayload(Info, "klotho uploading")
	p.Source = data
	t.send(p)
}

func (t *Client) createPayload(level LogLevel, event string) Payload {
	p := Payload{
		UserId:     t.userId,
		Event:      event,
		Properties: make(map[string]any, len(t.universalProperties)+2),
	}
	for k, v := range t.universalProperties {
		p.Properties[k] = v
	}
	p.Properties[datadogLogLevel] = level
	if level == Panic {
		p.Properties[datadogStatus] = Error // datadog doesn't support panic for the reserved status field
	} else {
		p.Properties[datadogStatus] = level
	}
	return p
}

// Hash hashes a value, using this analytic sender's userId as a salt. It does not output anything or in any way modify the
// sender's state.
func (t *Client) Hash(value any) string {
	h := sha256.New()
	h.Write([]byte(t.userId)) // use this as a salt
	if json.NewEncoder(h).Encode(value) != nil {
		return "unknown"
	}
	return fmt.Sprintf("sha256:%x", h.Sum(nil))
}

func (t *Client) PanicHandler(err *error, errHandler ErrorHandler) {
	if r := recover(); r != nil {
		rerr, ok := r.(error)
		if !ok {
			rerr = errors.Errorf("panic recovered: %v", r)
		}
		if *err != nil {
			*err = multierr.Error{*err, rerr}
		} else {
			*err = rerr
		}
		if _, hasStack := (*err).(interface{ StackTrace() errors.StackTrace }); !hasStack {
			*err = errors.WithStack(*err)
		}
		p := t.createPayload(Error, "ERROR")
		p.addError(rerr)
		t.send(p)
		errHandler.PrintErr(*err)
	}
}
