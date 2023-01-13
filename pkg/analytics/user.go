package analytics

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/klothoplatform/klotho/pkg/cli_config"
	"net/http"
	"net/mail"
	"os"

	"github.com/fatih/color"
	"github.com/google/uuid"
	"go.uber.org/zap"
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

func CreateUser(email string) error {

	// Check if the analytics file exists. If it does, try retrieving the user.
	// If it doesn't or we error because the data is invalid, it's fine.
	// We will create the new user and override the invalid or non-existent file
	result, err := getTrackingFileContents(analyticsFile)
	var existUser *User
	if err == nil {
		existUser = RetrieveUser(result)
	}

	user := User{}
	if email == "local" {
		// login local will wipe an existing set email, but we want to preserve any set uuid
		if existUser != nil {
			user.Id = existUser.Id
		} else {
			user.Id = uuid.New().String()
		}
		printLocalLoginMessage()
	} else {
		addr, err := mail.ParseAddress(email)
		if err != nil {
			return err
		}

		if existUser == nil {
			user.Email = addr.Address
			if err := user.SendUserEmailValidation(); err != nil {
				return err
			}
			printEmailLoginMessage(user.Email)
		} else {
			// preserve the uuid if it was set before
			user.Id = existUser.Id
			user.Email = addr.Address
			validated := false
			// Determine if the address provided is new or the same and if we need to do any validation
			if existUser.Email == addr.Address {
				validated = existUser.Validated
			} else {
				if v, err := user.CheckUserEmailValidation(); err != nil {
					zap.L().Warn("Failed to validate email with server")
				} else {
					user.Validated = v.Validated
				}
				validated = user.Validated
			}

			if validated {
				printEmailLoginMessage(user.Email)
			} else {
				if err := user.SendUserEmailValidation(); err != nil {
					return err
				}
				printEmailLoginMessage(user.Email)
			}
		}
	}

	configPath, err := cli_config.KlothoConfigPath(analyticsFile)
	if err != nil {
		return err
	}
	return user.writeConfig(configPath)
}

func RetrieveUser(result AnalyticsFile) *User {
	user := User{}

	if result.Email != "" {
		user.Email = result.Email
		if v, err := user.CheckUserEmailValidation(); err != nil {
			zap.L().Warn("Failed to validate email with server")
		} else {
			user.Validated = v.Validated
		}
	}
	if result.Id != "" {
		user.Id = result.Id
	}
	if (User{} == user) {
		return nil
	}
	return &user
}

func (u *User) CheckUserEmailValidation() (*Validated, error) {
	postBody, err := json.Marshal(u)
	if err != nil {
		return nil, err
	}

	data := bytes.NewBuffer(postBody)
	resp, err := http.Post(fmt.Sprintf("%v/user/check-validation", kloServerUrl), "application/json", data)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("failed to check user validation")
	}

	defer resp.Body.Close()

	validated := Validated{}

	err = json.NewDecoder(resp.Body).Decode(&validated)

	if err != nil {
		return nil, err
	}

	return &validated, nil
}

func (u *User) SendUserEmailValidation() error {
	postBody, _ := json.Marshal(u)
	data := bytes.NewBuffer(postBody)
	resp, err := http.Post(fmt.Sprintf("%v/user/send-validation", kloServerUrl), "application/json", data)

	if err != nil {
		return err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.New("failed to send user validation email")
	}

	return nil
}

func (user *User) writeConfig(configPath string) error {
	content, err := json.Marshal(user)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, content, 0660)
}

func (u *User) RegisterUser() error {

	postBody, err := json.Marshal(u)
	if err != nil {
		return err
	}

	data := bytes.NewBuffer(postBody)
	resp, err := http.Post(fmt.Sprintf("%v/analytics/user", kloServerUrl), "application/json", data)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("non 200 status code: %v", resp.StatusCode)
	}

	return nil
}

func printLocalLoginMessage() {
	color.New(color.FgHiGreen).Println("Success: Logged in as local user")
	color.New(color.FgYellow).Println(
		"If you would like to \n",
		"  \u2022 Receive support with klotho issues\n",
		"  \u2022 Help shape the future of the product\n",
		"  \u2022 Access features like the developer console",
	)
	color.New(color.FgHiBlue).Println(
		"run:\n",
		"  $ klotho --login <email>",
	)
}

func printEmailLoginMessage(email string) {
	color.New(color.FgHiGreen).Printf("Success: Logged in as %s\n\n", email)
}
