package cli_config

import (
	"os"
	"os/user"
	"path"
)

// KlothoConfigPath returns a path to a file in ~/.klotho/<filename>
func KlothoConfigPath(file string) (string, error) {
	osUser, err := user.Current()
	if err != nil {
		return "", err
	}
	klothoPath := path.Join(osUser.HomeDir, ".klotho")

	configPath := path.Join(klothoPath, file)
	return configPath, nil
}

func CreateKlothoConfigPath() error {
	osUser, err := user.Current()
	if err != nil {
		return err
	}
	klothoPath := path.Join(osUser.HomeDir, ".klotho")

	// create the directory if it doesn't exist
	_, err = os.Stat(klothoPath)
	if os.IsNotExist(err) {
		err = os.MkdirAll(klothoPath, os.ModePerm)
	}
	if err != nil {
		return err
	}
	return nil
}
