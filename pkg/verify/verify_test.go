package verify_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	cmd_verify "github.com/safe-waters/docker-lock/cmd/verify"
	"github.com/safe-waters/docker-lock/pkg/generate"
	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/generate/registry"
)

func TestVerifier(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name        string
		Contents    [][]byte
		ExcludeTags bool
		ShouldFail  bool
	}{
		{
			Name: "Dockerfile Diff",
			Contents: [][]byte{
				[]byte(`
FROM busybox
FROM busybox
`,
				),
				[]byte(`
{
	"dockerfiles": {
		"Dockerfile": [
			{
				"name": "busybox",
				"tag": "latest",
				"digest": "busybox"
			}
		]
	}
}
`),
			},
			ShouldFail: true,
		},
		{
			Name: "Composefile Diff",
			Contents: [][]byte{
				[]byte(`
version: '3'
services:
  svc-one:
    image: busybox
  svc-two:
    image: busybox
`,
				),
				[]byte(`
{
	"composefiles": {
		"docker-compose.yml": [
			{
				"name": "busybox",
				"tag": "latest",
				"digest": "busybox",
				"service": "svc-two"
			}
		]
	}
}
`,
				),
			},
			ShouldFail: true,
		},
		{
			Name: "Normal",
			Contents: [][]byte{
				[]byte(`
version: '3'
services:
  svc:
    image: busybox
`,
				),
				[]byte(`
{
	"composefiles": {
		"docker-compose.yml": [
			{
				"name": "busybox",
				"tag": "latest",
				"digest": "busybox",
				"service": "svc"
			}
		]
	}
}
`,
				),
			},
		},
		{
			Name: "Exclude Tags",
			Contents: [][]byte{
				[]byte(`
version: '3'
services:
  svc:
    image: busybox
`,
				),
				[]byte(`
{
	"composefiles": {
		"docker-compose.yml": [
			{
				"name": "busybox",
				"tag": "",
				"digest": "busybox",
				"service": "svc"
			}
		]
	}
}
`,
				),
			},
			ExcludeTags: true,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			tempDir := makeTempDirInCurrentDir(t)
			defer os.RemoveAll(tempDir)

			var lockfile generate.Lockfile
			if err := json.Unmarshal(
				test.Contents[len(test.Contents)-1], &lockfile,
			); err != nil {
				t.Fatal(err)
			}

			uniquePathsToWrite := map[string]struct{}{}

			composefileImagesWithTempDir := map[string][]*parse.ComposefileImage{} // nolint: lll

			for composefilePath, images := range lockfile.ComposefileImages {
				for _, image := range images {
					if image.DockerfilePath != "" {
						uniquePathsToWrite[image.DockerfilePath] = struct{}{}
						image.DockerfilePath = filepath.ToSlash(
							filepath.Join(tempDir, image.DockerfilePath),
						)
					}
				}
				uniquePathsToWrite[composefilePath] = struct{}{}

				composefilePath = filepath.ToSlash(
					filepath.Join(tempDir, composefilePath),
				)
				composefileImagesWithTempDir[composefilePath] = images
			}

			dockerfileImagesWithTempDir := map[string][]*parse.DockerfileImage{}

			for dockerfilePath, images := range lockfile.DockerfileImages {
				uniquePathsToWrite[dockerfilePath] = struct{}{}

				dockerfilePath = filepath.ToSlash(
					filepath.Join(tempDir, dockerfilePath),
				)
				dockerfileImagesWithTempDir[dockerfilePath] = images
			}

			var pathsToWrite []string
			for path := range uniquePathsToWrite {
				pathsToWrite = append(pathsToWrite, path)
			}

			pathsToWrite = append(pathsToWrite, "docker-lock.json")

			lockfileWithTempDir := &generate.Lockfile{
				DockerfileImages:  dockerfileImagesWithTempDir,
				ComposefileImages: composefileImagesWithTempDir,
			}

			lockfileWithTempDirByt, err := json.Marshal(lockfileWithTempDir)
			if err != nil {
				t.Fatal(err)
			}

			test.Contents[len(test.Contents)-1] = lockfileWithTempDirByt

			tempPaths := writeFilesToTempDir(
				t, tempDir, pathsToWrite, test.Contents,
			)

			reader := bytes.NewReader(lockfileWithTempDirByt)

			server := mockServer(t)
			defer server.Close()

			client := &registry.HTTPClient{
				Client:      server.Client(),
				RegistryURL: server.URL,
				TokenURL:    server.URL + "?scope=repository%s",
			}

			flags := &cmd_verify.Flags{
				LockfileName: tempPaths[len(tempPaths)-1],
				EnvPath:      ".env",
				ExcludeTags:  test.ExcludeTags,
			}

			verifier, err := cmd_verify.SetupVerifier(client, flags)
			if err != nil {
				t.Fatal(err)
			}

			err = verifier.VerifyLockfile(reader)

			if test.ShouldFail {
				if err == nil {
					t.Fatal("expected error but did not get one")
				}

				return
			}

			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
