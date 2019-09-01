package registry

import (
	"fmt"

	c "github.com/docker/docker-credential-helpers/client"
)

type Credentials struct {
	Username string
	Password string
}

// GetCredentials works for “osxkeychain” on macOS, “wincred” on windows, and “pass” on Linux.
func GetCredentials(credStore string) (creds *Credentials, err error) {
	credStore = fmt.Sprintf("%s-%s", "docker-credential", credStore)
	defer func() {
		if r := recover(); r != nil {
			creds = nil
			err = fmt.Errorf("%s not found.", credStore)
			return
		}
	}()
	p := c.NewShellProgramFunc(credStore)
	credResponse, err := c.Get(p, "https://index.docker.io/v1/")
	if err != nil {
		fmt.Println(err)
	}
	creds = &Credentials{Username: credResponse.Username, Password: credResponse.Secret}
	return creds, err
}
