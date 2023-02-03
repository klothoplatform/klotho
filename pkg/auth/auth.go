package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/klothoplatform/klotho/pkg/closenicely"
	"github.com/pkg/errors"
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

var authUrlBase = getAuthUrlBase()

type LoginResponse struct {
	Url   string
	State string
}

type Authorizer interface {
	Authorize() error
}

func DefaultIfNil(auth Authorizer) Authorizer {
	if auth == nil {
		return standardAuthorizer{}
	}
	return auth
}

type standardAuthorizer struct{}

func (s standardAuthorizer) Authorize() error {
	return Authorize()
}

func Login(onError func(error)) error {
	state, err := CallLoginEndpoint()
	if err != nil {
		return err
	}
	err = CallGetTokenEndpoint(state)
	if err != nil {
		onError(err)
	}
	return nil
}

func CallLoginEndpoint() (string, error) {
	res, err := http.Get(authUrlBase + "/login")
	if err != nil {
		return "", err
	}
	defer closenicely.OrDebug(res.Body)
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
	return result.State, nil
}

func CallGetTokenEndpoint(state string) error {
	values := map[string]string{"state": state}
	jsonData, err := json.Marshal(values)
	if err != nil {
		log.Fatal(err)
	}
	res, err := http.Post(authUrlBase+"/logintoken", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer closenicely.OrDebug(res.Body)
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
	return nil
}

func CallLogoutEndpoint() error {
	res, err := http.Get(authUrlBase + "/logout")
	if err != nil {
		return errors.Wrap(err, "couldn't invoke logout URL")
	}
	defer closenicely.OrDebug(res.Body)
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return errors.Wrap(err, "couldn't read logout redirect URL")
	}
	err = browser.OpenURL(string(body))
	if err != nil {
		zap.S().Debug("couldn't open logout URL: %s", string(body))
		zap.L().Warn("couldn't open logout URL. If this persists, run with --verbose to see it. Will still clear local credentials.")
	}

	configPath, err := cli_config.KlothoConfigPath("credentials.json")
	if err != nil {
		return err
	}
	if _, err := os.Stat(configPath); err == nil {
		err = os.Remove(configPath)
		if err != nil {
			return err
		}
	}
	return nil
}

func CallRefreshToken(token string) error {
	values := map[string]string{"refresh_token": token}
	jsonData, err := json.Marshal(values)
	if err != nil {
		return err
	}
	res, err := http.Post(authUrlBase+"/refresh", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	defer closenicely.OrDebug(res.Body)
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	err = WriteIDToken(string(body))
	if err != nil {
		return err
	}
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

func Authorize() error {
	return authorize(false)
}

func authorize(tokenRefreshed bool) error {
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
			err = authorize(true)
			if err != nil {
				return err
			}
		} else if !claims.ProEnabled {
			return fmt.Errorf("user %s is not authorized to use KlothoPro", claims.Email)
		} else if claims.ExpiresAt < time.Now().Unix() {
			if tokenRefreshed {
				return fmt.Errorf("user %s, does not have a valid token", claims.Email)
			}
			err := CallRefreshToken(creds.RefreshToken)
			if err != nil {
				return err
			}
			err = authorize(true)
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

func getAuthUrlBase() string {
	host := os.Getenv("KLOTHO_AUTH_BASE")
	if host == "" {
		host = "http://klotho-auth-service-alb-e22c092-466389525.us-east-1.elb.amazonaws.com"
	}
	return host
}
