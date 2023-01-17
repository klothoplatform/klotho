package analytics

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/klothoplatform/klotho/pkg/core"
	"github.com/klothoplatform/klotho/pkg/multierr"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type (
	Client struct {
		UserId     string                 `json:"id"`
		Event      string                 `json:"event"`
		Source     []byte                 `json:"source,omitempty"`
		Properties map[string]interface{} `json:"properties,omitempty"`
	}
	ErrorHandler interface {
		PrintErr(err error)
	}
	LogLevel string
)

var (
	Panic LogLevel = "panic"
	Error LogLevel = "error"
	Warn  LogLevel = "warn"
	Info  LogLevel = "info"
	Debug LogLevel = "debug"
)

var datadogLogLevel = "_logLevel"
var datadogStatus = "status"

func NewClient(properties map[string]interface{}) (*Client, error) {
	result, err := getTrackingFileContents(analyticsFile)
	if err != nil {
		return nil, err
	}
	user := RetrieveUser(result)
	if user == nil {
		return nil, errors.New("required user info not set")
	}

	err = user.RegisterUser()
	if err != nil {
		return nil, err
	}

	client := &Client{
		Properties: properties,
	}
	if user.Email != "" {
		client.UserId = user.Email
		client.Properties["validated"] = user.Validated
		if user.Id != "" {
			client.Properties["localId"] = user.Id
		}
	} else {
		client.UserId = user.Id
	}
	if runUuid, err := uuid.NewRandom(); err == nil {
		client.Properties["runId"] = runUuid.String()
	}

	return client, nil
}

func (t *Client) Info(event string) {
	t.Properties[datadogLogLevel] = Info
	t.track(event)
}

func (t *Client) Debug(event string) {
	t.Properties[datadogLogLevel] = Debug
	t.track(event)
}

func (t *Client) Warn(event string) {
	t.Properties[datadogLogLevel] = Warn
	t.Properties[datadogStatus] = Warn
	t.track(event)
}

func (t *Client) Error(event string) {
	t.Properties[datadogLogLevel] = Error
	t.Properties[datadogStatus] = Error
	t.track(event)
}

func (t *Client) Panic(event string) {
	t.Properties[datadogLogLevel] = Panic
	// Using error since datadog does not support panic for the reserved status field
	t.Properties[datadogStatus] = Error
	t.track(event)
}

func (t *Client) AppendProperties(properties map[string]interface{}) {
	for k, v := range properties {
		t.Properties[k] = v
	}
}

func (t *Client) UploadSource(source *core.InputFiles) {
	data, err := CompressFiles(source)
	if err != nil {
		zap.S().Warnf("Failed to upload debug bundle. %v", err)
		return
	}
	srcClient := &Client{
		UserId:     t.UserId,
		Properties: t.Properties,
		Source:     data,
	}
	srcClient.Info("klotho uploading")
}

func (t *Client) track(event string) {
	t.Event = event
	err := SendTrackingToServer(t)

	if err != nil {
		zap.L().Debug(fmt.Sprintf("Failed to send metrics info. %v", err))
	}
}

// Hash hashes a value, using this analytic sender's UserId as a salt. It does not output anything or in any way modify the
// sender's state.
func (t *Client) Hash(value any) string {
	h := sha256.New()
	h.Write([]byte(t.UserId)) // use this as a salt
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
		t.Panic(rerr.Error())
		errHandler.PrintErr(*err)
		*err = nil
	}
}
