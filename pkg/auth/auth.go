package auth

import (
	"bytes"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"github.com/klothoplatform/klotho/pkg/closenicely"
	"github.com/pkg/errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/klothoplatform/klotho/pkg/cli_config"
	"github.com/pkg/browser"
	"go.uber.org/zap"
)

var authUrlBase = EnvVar("KLOTHO_AUTH_BASE").GetOr(`http://klotho-auth-service-alb-e22c092-466389525.us-east-1.elb.amazonaws.com`)

var pemUrl = EnvVar("KLOTHO_AUTH_PEM").GetOr(`https://klotho.us.auth0.com/pem`)

type LoginResponse struct {
	Url   string
	State string
}

type Authorizer interface {
	Authorize() (*KlothoClaims, error)
}

func DefaultIfNil(auth Authorizer) Authorizer {
	if auth == nil {
		return standardAuthorizer{}
	}
	return auth
}

type standardAuthorizer struct{}

func (s standardAuthorizer) Authorize() (*KlothoClaims, error) {
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

type KlothoClaims struct {
	ProEnabled    bool
	ProTier       int
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	jwt.RegisteredClaims
}

func Authorize() (*KlothoClaims, error) {
	return authorize(false)
}

func authorize(tokenRefreshed bool) (*KlothoClaims, error) {
	creds, claims, err := getClaims()
	if err != nil {
		return nil, err
	}

	if !claims.EmailVerified {
		if tokenRefreshed {
			return nil, fmt.Errorf("user %s, has not verified their email", claims.Email)
		}
		err := CallRefreshToken(creds.RefreshToken)
		if err != nil {
			return nil, err
		}
		claims, err = authorize(true)
		if err != nil {
			return nil, err
		}
	} else if !claims.ProEnabled {
		return nil, fmt.Errorf("user %s is not authorized to use KlothoPro", claims.Email)
	} else if claims.ExpiresAt.Before(time.Now()) {
		if tokenRefreshed {
			return nil, fmt.Errorf("user %s, does not have a valid token", claims.Email)
		}
		err := CallRefreshToken(creds.RefreshToken)
		if err != nil {
			return nil, err
		}
		claims, err = authorize(true)
		if err != nil {
			return nil, err
		}
	}
	return claims, nil
}

func getClaims() (*Credentials, *KlothoClaims, error) {
	errMsg := `Failed to get credentials for user. Please run "klotho --login"`
	creds, err := GetIDToken()
	if err != nil {
		return nil, nil, errors.New(errMsg)
	}
	token, err := jwt.ParseWithClaims(creds.IdToken, &KlothoClaims{}, func(token *jwt.Token) (interface{}, error) {
		return getPem()
	})
	if err != nil {
		return nil, nil, errors.Wrap(err, errMsg)
	}
	if claims, ok := token.Claims.(*KlothoClaims); ok {
		return creds, claims, nil
	} else {
		return nil, nil, errors.Wrap(err, errMsg)
	}
}

func getPem() (*rsa.PublicKey, error) {
	var authServerPemCacheFile = path.Join("pem", url.PathEscape(pemUrl))

	writePemCache := false
	// Try to read the PEM from local cache
	configPath, err := cli_config.KlothoConfigPath(authServerPemCacheFile)
	if err != nil {
		return nil, err
	}
	bs, err := os.ReadFile(configPath)
	// Couldn't read it from cache, so (a) try to fetch it from URL and (b) mark down that we should write it on success
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			zap.L().Debug("Couldn't read PEM cache file. Will download it.", zap.Error(err))
		}
		pemResp, err := http.Get(pemUrl)
		if err != nil {
			return nil, err
		}
		defer closenicely.OrDebug(pemResp.Body)
		bs, err = io.ReadAll(pemResp.Body)
		if err != nil {
			return nil, err
		}
		writePemCache = true
	}
	// okay, we have the PEM bytes. Try to decode them into a PublicKey.
	block, _ := pem.Decode(bs)
	if block == nil {
		return nil, errors.New("Couldn't parse PEM certificate")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, err
	}
	pub, ok := cert.PublicKey.(*rsa.PublicKey)
	if !ok {
		return nil, errors.New("Couldn't parse PEM certificate block")
	}
	// Finally, if we'd fetched the PEM bytes from URL, save them now.
	if writePemCache {
		configPath, err := cli_config.KlothoConfigPath(authServerPemCacheFile)
		if err == nil {
			_ = os.MkdirAll(path.Dir(configPath), 0777)
			err = os.WriteFile(configPath, bs, 0644)
		}
		if err != nil {
			zap.L().Debug("Couldn't write PEM to local cache", zap.Error(err))
		}
	}
	return pub, nil
}

// EnvVar represents an environment variable, specified by its key name. This is a
// wrapper around os.Getenv. This string's value is the env var key. Use GetOr to get its value.
type EnvVar string

// GetOr uses os.Getenv to get the env var specified by the target EnvVar. If that env var's value is unset or empty,
// it returns the defaultValue.
func (s EnvVar) GetOr(defaultValue string) string {
	value := os.Getenv(string(s))
	if value == "" {
		return defaultValue
	} else {
		return value
	}
}
