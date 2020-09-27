package registry

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"strings"

	c "github.com/docker/docker-credential-helpers/client"
)

// AuthCredentials contains a username and password required to
// auth with a container registry.
type AuthCredentials struct {
	Username string
	Password string
}

// AuthStringExtractor allows different registry wrappers to control
// how the auth information is extracted from docker's config file.
type AuthStringExtractor interface {
	// ExtractAuthStr reads the base64 encoded auth string and the creds store
	// from the configuration file's bytes.
	ExtractAuthStr(confByt []byte) (authStr string, credStore string, err error)
	// ServerURL is the URL of the login server.
	ServerURL() string
}

// NewAuthCredentials returns AuthCredentials according to the following rules:
// (1) If a username and password are not empty, use them.
// (2) Else if a username and password are empty:
//		a. If the configPath is empty, return empty creds.
//		b. Else, attempt to extract the base64 creds from the config file.
//		c. If creds are empty, but the config file specifies a creds store,
//		   retrieve the creds from the creds store.
// (3) Return empty creds.
func NewAuthCredentials(
	username string,
	password string,
	configPath string,
	extractor AuthStringExtractor,
) (*AuthCredentials, error) {
	if username != "" && password != "" {
		return &AuthCredentials{
			Username: username,
			Password: password,
		}, nil
	}

	if configPath == "" {
		return &AuthCredentials{}, nil
	}

	confByt, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	authBase64, credsStore, err := extractor.ExtractAuthStr(confByt)
	if err != nil {
		return nil, err
	}

	authByt, err := base64.StdEncoding.DecodeString(authBase64)
	if err != nil {
		return nil, err
	}

	authStr := string(authByt)

	switch {
	case authStr != "":
		auth := strings.Split(authStr, ":")
		return &AuthCredentials{Username: auth[0], Password: auth[1]}, nil
	case credsStore != "":
		authCreds, err := authCredsFromStore(credsStore, extractor.ServerURL())
		if err != nil {
			return &AuthCredentials{}, nil
		}

		return authCreds, nil
	}

	return &AuthCredentials{}, nil
}

// authCredsFromStore reads auth from a creds store such as
// wincred, pass, or osxkeychain by shelling out to docker-credential-helper.
func authCredsFromStore(
	credsStore string,
	serverURL string,
) (authCreds *AuthCredentials, err error) {
	defer func() {
		if err := recover(); err != nil {
			authCreds = &AuthCredentials{}
			return
		}
	}()

	credsStore = fmt.Sprintf("%s-%s", "docker-credential", credsStore)
	p := c.NewShellProgramFunc(credsStore)

	credRes, err := c.Get(p, serverURL)
	if err != nil {
		return authCreds, err
	}

	return &AuthCredentials{
		Username: credRes.Username,
		Password: credRes.Secret,
	}, nil
}
