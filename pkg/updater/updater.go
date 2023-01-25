package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"
	"time"

	"github.com/coreos/go-semver/semver"
	"github.com/gojek/heimdall/v7/httpclient"
	"github.com/inconshreveable/go-update"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var (
	OS   string = runtime.GOOS
	Arch string = runtime.GOARCH
)

var (
	DefaultServer string = "http://srv.klo.dev"
)

type Updater struct {
	ServerURL string
	// Stream is the update stream to check
	Stream string
	// CurrentStream is the stream this binary came from
	CurrentStream string
}

func selfUpdate(data io.ReadCloser) error {
	//TODO add signature verification if we want
	return update.Apply(data, update.Options{})
}

// CheckUpdate compares the version of the klotho binary
// against the latest github release, returns true
// if the latest release is newer
func (u *Updater) CheckUpdate(currentVersion string) (bool, error) {
	timeout := 10 * time.Second
	cli := httpclient.NewClient(httpclient.WithHTTPTimeout(timeout))
	endpoint := fmt.Sprintf("%s/update/check-latest-version?stream=%s", u.ServerURL, u.Stream)
	res, err := cli.Get(endpoint, nil)
	if err != nil {
		return false, fmt.Errorf("failed to query for latest version: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		return false, fmt.Errorf("failed to query for latest version, bad response from server: %d", res.StatusCode)

	}
	defer res.Body.Close()

	result := make(map[string]string)
	dec := json.NewDecoder(res.Body)
	if err := dec.Decode(&result); err != nil {
		return false, fmt.Errorf("failed to decode body: %v", err)
	}

	ver, ok := result["latest_version"]
	if !ok {
		return false, errors.New("no version found in result")
	}

	latestVersion, err := semver.NewVersion(ver)
	if err != nil {
		return false, fmt.Errorf("strange version received: %s", latestVersion)
	}

	currVersion, err := semver.NewVersion(strings.TrimPrefix(currentVersion, "v"))
	if err != nil {
		return false, fmt.Errorf("invalid version %s: %v", currentVersion, err)
	}

	return currVersion.LessThan(*latestVersion), nil
}

// Update performs an update if a newer version is
// available
func (u *Updater) Update(currentVersion string) error {
	doUpdate, err := u.CheckUpdate(currentVersion)
	if err != nil {
		zap.S().Errorf(`error checking for updates on stream "%s": %v`, u.Stream, err)
		return err
	}

	if !doUpdate {
		zap.S().Infof(`already up to date on stream "%s".`, u.Stream)
		return nil
	}

	body, err := getLatest(u.ServerURL, u.Stream)
	if err != nil {
		return errors.Wrapf(err, "failed to get latest")
	}
	if body != nil {
		defer body.Close()
	}

	if err := selfUpdate(body); err != nil {
		return errors.Wrapf(err, "failed to update klotho")
	}
	zap.S().Infof(`updated to the latest version on stream "%s"`, u.Stream)
	return nil
}

// getLatest Grabs latest release from klotho server
func getLatest(baseUrl string, stream string) (io.ReadCloser, error) {
	timeout := 10 * time.Second
	cli := httpclient.NewClient(httpclient.WithHTTPTimeout(timeout))

	endpoint := fmt.Sprintf("%s/update/latest/%s/%s?stream=%s", baseUrl, OS, Arch, stream)
	res, err := cli.Get(endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query for latest version: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to query for latest version, bad response from server: %d", res.StatusCode)

	}
	return res.Body, nil

}
