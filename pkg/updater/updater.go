package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"strings"

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

const (
	DefaultServer string = "http://srv.klo.dev"
)

type Updater struct {
	ServerURL string
	// Stream is the update stream to check
	Stream string
	// CurrentStream is the stream this binary came from
	CurrentStream string

	Client *httpclient.Client
}

func selfUpdate(data io.ReadCloser) error {
	//TODO add signature verification if we want
	return update.Apply(data, update.Options{})
}

// CheckUpdate compares the version of the klotho binary
// against the latest github release, returns true
// if the latest release is newer
func (u *Updater) CheckUpdate(currentVersion string) (bool, error) {
	endpoint := fmt.Sprintf("%s/update/check-latest-version?stream=%s", u.ServerURL, u.Stream)
	res, err := u.Client.Get(endpoint, nil)
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

	// Given a stream "xxx:yyyy", the qualifier is the "xxx" and the tag is the "yyyy".
	//
	// (1) If the qualifiers are different, always update (this is to handle open <--> pro)
	// Otherwise, check the cli's version against latest. This is a bit trickier:
	//
	// (2a) If the tags are the same, then either it's a specific version or it's a monotonic tag like "latest".
	//   • If it's a monotonic tag, we only want to perform upgrades. A downgrade would be a situation like if we gave
	//     someone a pre-release, in which case we don't want to downgrade them.
	//   • If it's a specific version, we can assume that the version will never change.
	//   • So in either case, we want to only perform upgrades.
	// (2b) If the tags are different, then someone is either pinning to a specific version, or going from a pinned
	//     version to a monotonic version. In either case, we should allow downgrades. (Going from pinned to monotonic
	//     *may* be an incorrect downgrade, with a similar pre-release reason. But if someone has a pre-release, they
	//     shouldn't be worrying about any upgrade stuff, including not changing their update stream from pinned to
	//     monotonic.)

	// case (1): different qualifiers always update
	if strings.Split(u.CurrentStream, ":")[0] != strings.Split(u.Stream, ":")[0] {
		return true, nil
	}

	// the qualifiers are the same, so the tags are the same iff the full stream strings are the same
	if u.CurrentStream == u.Stream {
		return currVersion.LessThan(*latestVersion), nil // case (2a): only upgrades
	} else {
		return !currVersion.Equal(*latestVersion), nil // case (2b): upgrades or downgrades
	}
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

	body, err := u.getLatest()
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
func (u *Updater) getLatest() (io.ReadCloser, error) {
	endpoint := fmt.Sprintf("%s/update/latest/%s/%s?stream=%s", u.ServerURL, OS, Arch, u.Stream)
	res, err := u.Client.Get(endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to query for latest version: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to query for latest version, bad response from server: %d", res.StatusCode)

	}
	return res.Body, nil

}
