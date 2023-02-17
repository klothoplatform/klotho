package analytics

import (
	"encoding/json"
	"github.com/google/uuid"
	"github.com/klothoplatform/klotho/pkg/cli_config"
	"go.uber.org/zap"
	"os"
)

type User struct {
	Email string `json:"email,omitempty"`
	// uuid generated if user does not provide email
	Id string `json:"id,omitempty"`
	// omit validated field from being saved since we wouldn't trust the client side value anyways
	Validated bool `json:"-"`
}

type Validated struct {
	Validated bool
}

// located in ~/.klotho/
var analyticsFile = "analytics.json"

func GetOrCreateAnalyticsFile() AnalyticsFile {
	// Check if the analytics file exists. If it does, try retrieving the user.
	// If it doesn't or we error because the data is invalid, it's fine.
	// We will create the new user and override the invalid or non-existent file

	localLogin, err := getTrackingFileContents(analyticsFile)
	if err == nil {
		return localLogin
	}
	login := AnalyticsFile{Id: uuid.New().String()}

	// Try to write the file, but don't let any errors stop us
	err = writeTrackingFileContents(analyticsFile, AnalyticsFile{Id: login.Id})
	if err != nil {
		zap.L().Debug("Couldn't write local analytics state", zap.Error(err))
	}
	return login
}
func getTrackingFileContents(file string) (AnalyticsFile, error) {
	configPath, err := cli_config.KlothoConfigPath(file)
	result := AnalyticsFile{}

	if err != nil {
		return result, err
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		return result, err
	}
	err = json.Unmarshal(content, &result)

	return result, err
}

func writeTrackingFileContents(file string, contents AnalyticsFile) error {
	configPath, err := cli_config.KlothoConfigPath(file)
	if err != nil {
		return err
	}
	loginJson, err := json.Marshal(contents)
	if err != nil {
		return err
	}
	return os.WriteFile(configPath, loginJson, 0660)
}
