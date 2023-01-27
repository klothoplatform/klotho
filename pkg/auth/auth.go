package auth

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/klothoplatform/klotho/pkg/cli_config"
	"github.com/pkg/browser"
	"go.uber.org/zap"
)

type LoginResponse struct {
	Url   string
	State string
}

func Login() error {
	state, err := CallLoginEndpoint()
	if err != nil {
		return err
	}
	err = retry(20, time.Duration(5)*time.Second, CallGetTokenEndpoint, state)
	return err
}

func CallLoginEndpoint() (string, error) {
	res, err := http.Get("http://localhost:3000/login")
	if err != nil {
		return "", err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	result := LoginResponse{}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return "", err
	}
	err = browser.OpenURL(result.Url)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	return result.State, nil
}

func CallGetTokenEndpoint(state string) error {
	values := map[string]string{"state": state}
	jsonData, err := json.Marshal(values)
	if err != nil {
		log.Fatal(err)
	}
	res, err := http.Post("http://localhost:3000/logintoken", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	if res.StatusCode != 200 {
		return fmt.Errorf("recieved invalid status code %d", res.StatusCode)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	err = WriteIDToken(string(body))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	return nil
}

func CallLogoutEndpoint() error {
	res, _ := http.Get("http://localhost:3000/logout")
	body, _ := io.ReadAll(res.Body)
	_ = browser.OpenURL(string(body))
	defer res.Body.Close()

	configPath, err := cli_config.KlothoConfigPath("credentials.json")
	if err != nil {
		return err
	}
	err = os.Remove(configPath)
	if err != nil {
		return err
	}
	return nil
}

func CallRefreshToken(token string) error {
	values := map[string]string{"refresh_token": token}
	jsonData, err := json.Marshal(values)
	if err != nil {
		return err
	}
	res, err := http.Post("http://localhost:3000/refresh", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	err = WriteIDToken(string(body))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	return nil
}

type MyCustomClaims struct {
	ProEnabled    bool
	ProTier       int
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	jwt.StandardClaims
}

func Authorize(tokenRefreshed bool) error {
	creds, err := GetIDToken()
	if err != nil {
		return errors.New("failed to get credentials for user, please login")
	}

	token, err := jwt.ParseWithClaims(creds.IdToken, &MyCustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return nil, nil
	})
	if err != nil {
		zap.S().Debug(err)
	}
	if claims, ok := token.Claims.(*MyCustomClaims); ok {
		if !claims.EmailVerified {
			if tokenRefreshed {
				return fmt.Errorf("user %s, has not verified their email", claims.Email)
			}
			err := CallRefreshToken(creds.RefreshToken)
			if err != nil {
				return err
			}
			err = Authorize(true)
			if err != nil {
				return err
			}
		} else if !claims.ProEnabled {
			return fmt.Errorf("user %s is not authorized to use KlothoPro", claims.Email)
		} else if claims.IssuedAt < time.Now().Unix() {
			if tokenRefreshed {
				return fmt.Errorf("user %s, does not have a valid token", claims.Email)
			}
			err := CallRefreshToken(creds.RefreshToken)
			if err != nil {
				return err
			}
			err = Authorize(true)
			if err != nil {
				return err
			}
		}
	} else {
		return errors.New("failed to authorize user")
	}
	return nil
}

func GetUserEmail() (string, error) {
	creds, err := GetIDToken()
	if err != nil {
		return "", errors.New("failed to get credentials for user, please login")
	}
	token, err := jwt.ParseWithClaims(creds.IdToken, &MyCustomClaims{}, func(token *jwt.Token) (interface{}, error) {
		return nil, nil
	})
	if err != nil {
		zap.S().Debug(err)
	}
	if claims, ok := token.Claims.(*MyCustomClaims); ok {
		return claims.Email, nil
	} else {
		return "", errors.New("failed to authorize user")
	}
}

func retry(attempts int, sleep time.Duration, f func(state string) error, state string) (err error) {
	for i := 0; ; i++ {
		err = f(state)
		if err == nil {
			return
		}

		if i >= (attempts - 1) {
			break
		}

		time.Sleep(sleep)
		sleep *= 2
	}
	return fmt.Errorf("after %d attempts, last error: %s", attempts, err)
}
