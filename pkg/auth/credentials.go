package auth

import (
	"encoding/json"
	"github.com/pkg/errors"
	"os"

	"github.com/klothoplatform/klotho/pkg/cli_config"
)

type Credentials struct {
	IdToken      string
	RefreshToken string
}

func WriteIDToken(token string) error {
	configPath, err := cli_config.KlothoConfigPath("credentials.json")
	if err != nil {
		return err
	}
	err = cli_config.CreateKlothoConfigPath()
	if err != nil {
		return err
	}
	err = os.WriteFile(configPath, []byte(token), 0644)
	if err != nil {
		return err
	}
	return nil
}

func GetIDToken() (*Credentials, error) {
	idToken := os.Getenv("KLOTHO_ID_TOKEN")
	if idToken != "" {
		return &Credentials{
			IdToken: idToken,
		}, nil
	}

	configPath, err := cli_config.KlothoConfigPath("credentials.json")
	result := Credentials{}

	if err != nil {
		return &result, err
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			err = ErrNoCredentialsFile
		}
		return nil, err
	}
	if len(content) > 0 {
		err = json.Unmarshal(content, &result)
	}

	return &result, err
}
