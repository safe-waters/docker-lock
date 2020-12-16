package rewrite_test

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/safe-waters/docker-lock/internal/testutils"
	"github.com/safe-waters/docker-lock/pkg/kind"

	cmd_rewrite "github.com/safe-waters/docker-lock/cmd/rewrite"
)

func TestRewriter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name       string
		Contents   [][]byte
		Expected   [][]byte
		ShouldFail bool
	}{
		{
			Name: "Composefile Overrides Dockerfile",
			Contents: [][]byte{
				[]byte(`FROM golang
`,
				),
				[]byte(`
version: '3'

services:
  svc:
    build: .
`,
				),
				[]byte(`
{
	"dockerfiles": {
		"Dockerfile": [
			{
				"name": "not_used",
				"tag": "latest",
				"digest": "not_used"
			}
		]
	},
	"composefiles": {
		"docker-compose.yml": [
			{
				"name": "golang",
				"tag": "latest",
				"digest": "golang",
				"dockerfile": "Dockerfile",
				"service": "svc"
			}
		]
	}
}
`,
				),
			},
			Expected: [][]byte{
				[]byte(`FROM golang:latest@sha256:golang
`,
				),
				[]byte(`
version: '3'

services:
  svc:
    build: .
`,
				),
			},
		},
		{
			Name: "Dockerfile, Composefile, and Kubernetesfile",
			Contents: [][]byte{
				[]byte(`FROM golang
`,
				),
				[]byte(`
version: '3'

services:
  svc:
    build: .
`,
				),
				[]byte(`apiVersion: v1
kind: Pod
metadata:
  name: test
  labels:
    app: test
spec:
  containers:
  - name: redis
    image: redis
    ports:
    - containerPort: 80
`),
				[]byte(`
{
	"dockerfiles": {
		"Dockerfile": [
			{
				"name": "not_used",
				"tag": "latest",
				"digest": "not_used"
			}
		]
	},
	"composefiles": {
		"docker-compose.yml": [
			{
				"name": "golang",
				"tag": "latest",
				"digest": "golang",
				"dockerfile": "Dockerfile",
				"service": "svc"
			}
		]
	},
	"kubernetesfiles": {
		"pod.yaml": [
			{
				"name": "redis",
				"tag": "latest",
				"digest": "redis",
				"container": "redis"
			}
		]
	}
}
`,
				),
			},
			Expected: [][]byte{
				[]byte(`FROM golang:latest@sha256:golang
`,
				),
				[]byte(`
version: '3'

services:
  svc:
    build: .
`,
				),
				[]byte(`apiVersion: v1
kind: Pod
metadata:
  name: test
  labels:
    app: test
spec:
  containers:
  - name: redis
    image: redis:latest@sha256:redis
    ports:
    - containerPort: 80
`),
			},
		},
		{
			Name: "Duplicate Services Same Dockerfile Images",
			Contents: [][]byte{
				[]byte(`FROM golang
`,
				),
				[]byte(`
version: '3'

services:
  svc:
    build: .
  another-svc:
    build: .
`,
				),
				[]byte(`
{
	"composefiles": {
		"docker-compose.yml": [
			{
				"name": "golang",
				"tag": "latest",
				"digest": "golang",
				"dockerfile": "Dockerfile",
				"service": "another-svc"
			},
			{
				"name": "golang",
				"tag": "latest",
				"digest": "golang",
				"dockerfile": "Dockerfile",
				"service": "svc"
			}
		]
	}
}
`,
				),
			},
			Expected: [][]byte{
				[]byte(`FROM golang:latest@sha256:golang
`,
				),
				[]byte(`
version: '3'

services:
  svc:
    build: .
  another-svc:
    build: .
`,
				),
			},
		},
		{
			Name: "Different Composefiles Same Dockerfile Images",
			Contents: [][]byte{
				[]byte(`FROM golang
`,
				),
				[]byte(`
version: '3'

services:
  svc:
    build: .
`,
				),
				[]byte(`
version: '3'

services:
  svc:
    build: .
`,
				),
				[]byte(`
{
	"composefiles": {
		"docker-compose-1.yml": [
			{
				"name": "golang",
				"tag": "latest",
				"digest": "golang",
				"dockerfile": "Dockerfile",
				"service": "svc"
			}
		],
		"docker-compose-2.yml": [
			{
				"name": "golang",
				"tag": "latest",
				"digest": "golang",
				"dockerfile": "Dockerfile",
				"service": "svc"
			}
		]
	}
}
`,
				),
			},
			Expected: [][]byte{
				[]byte(`FROM golang:latest@sha256:golang
`,
				),
				[]byte(`
version: '3'

services:
  svc:
    build: .
`,
				),
				[]byte(`
version: '3'

services:
  svc:
    build: .
`,
				),
			},
		},
		{
			Name: "Duplicate Services Different Dockerfile Images",
			Contents: [][]byte{
				[]byte(`FROM golang
`,
				),
				[]byte(`
version: '3'

services:
  svc:
    build: .
  another-svc:
    build: .
`,
				),
				[]byte(`
{
	"composefiles": {
		"docker-compose.yml": [
			{
				"name": "golang",
				"tag": "latest",
				"digest": "golang",
				"dockerfile": "Dockerfile",
				"service": "another-svc"
			},
			{
				"name": "notgolang",
				"tag": "latest",
				"digest": "notgolang",
				"dockerfile": "Dockerfile",
				"service": "svc"
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
			Name: "Different Composefiles Different Dockerfile Images",
			Contents: [][]byte{
				[]byte(`FROM golang
`,
				),
				[]byte(`
version: '3'

services:
  svc:
    build: .
`,
				),
				[]byte(`
version: '3'

services:
  svc:
    build: .
`,
				),
				[]byte(`
{
	"composefiles": {
		"docker-compose-one.yml": [
			{
				"name": "golang",
				"tag": "latest",
				"digest": "golang",
				"dockerfile": "Dockerfile",
				"service": "svc"
			}
		],
		"docker-compose-two.yml": [
			{
				"name": "notgolang",
				"tag": "latest",
				"digest": "notgolang",
				"dockerfile": "Dockerfile",
				"service": "svc"
			}
		]
	}
}
`,
				),
			},
			ShouldFail: true,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			tempDir := testutils.MakeTempDirInCurrentDir(t)
			defer os.RemoveAll(tempDir)

			var lockfile map[kind.Kind]map[string][]interface{}
			if err := json.Unmarshal(
				test.Contents[len(test.Contents)-1], &lockfile,
			); err != nil {
				t.Fatal(err)
			}

			uniquePathsToWrite := map[string]struct{}{}

			composefileImagesWithTempDir := map[string][]interface{}{}

			for composefilePath, images := range lockfile[kind.Composefile] {
				for _, image := range images {
					image := image.(map[string]interface{})
					if image["dockerfile"] != nil {
						dockerfilePath := image["dockerfile"].(string)
						uniquePathsToWrite[dockerfilePath] = struct{}{}
						image["dockerfile"] = filepath.Join(
							tempDir, dockerfilePath,
						)
					}
				}
				uniquePathsToWrite[composefilePath] = struct{}{}

				composefilePath = filepath.Join(tempDir, composefilePath)
				composefileImagesWithTempDir[composefilePath] = images
			}

			dockerfileImagesWithTempDir := map[string][]interface{}{}

			for dockerfilePath, images := range lockfile[kind.Dockerfile] {
				uniquePathsToWrite[dockerfilePath] = struct{}{}

				dockerfilePath = filepath.Join(tempDir, dockerfilePath)
				dockerfileImagesWithTempDir[dockerfilePath] = images
			}

			kubernetesfileImagesWithTempDir := map[string][]interface{}{}

			for kubernetesfilePath, images := range lockfile[kind.Kubernetesfile] { // nolint: lll
				uniquePathsToWrite[kubernetesfilePath] = struct{}{}

				kubernetesfilePath = filepath.Join(tempDir, kubernetesfilePath)
				kubernetesfileImagesWithTempDir[kubernetesfilePath] = images
			}

			var pathsToWrite []string
			for path := range uniquePathsToWrite {
				pathsToWrite = append(pathsToWrite, path)
			}

			sort.Strings(pathsToWrite)

			got := testutils.WriteFilesToTempDir(
				t, tempDir, pathsToWrite, test.Contents[:len(test.Contents)-1],
			)

			noopFile := filepath.Base("rewriter_test.go")

			flags, err := cmd_rewrite.NewFlags(noopFile, tempDir, false)
			if err != nil {
				t.Fatal(err)
			}

			rewriter, err := cmd_rewrite.SetupRewriter(flags)
			if err != nil {
				t.Fatal(err)
			}

			lockfileWithTempDir := map[kind.Kind]map[string][]interface{}{
				kind.Dockerfile:     dockerfileImagesWithTempDir,
				kind.Composefile:    composefileImagesWithTempDir,
				kind.Kubernetesfile: kubernetesfileImagesWithTempDir,
			}

			lockfileByt, err := json.Marshal(lockfileWithTempDir)
			if err != nil {
				t.Fatal(err)
			}
			reader := bytes.NewReader(lockfileByt)

			err = rewriter.RewriteLockfile(reader, tempDir)

			if test.ShouldFail {
				if err == nil {
					t.Fatal("expected error but did not get one")
				}

				return
			}

			if err != nil {
				t.Fatal(err)
			}

			testutils.AssertWrittenFilesEqual(t, test.Expected, got)
		})
	}
}
