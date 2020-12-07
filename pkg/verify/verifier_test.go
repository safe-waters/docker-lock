package verify_test

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	cmd_generate "github.com/safe-waters/docker-lock/cmd/generate"
	cmd_verify "github.com/safe-waters/docker-lock/cmd/verify"
	"github.com/safe-waters/docker-lock/internal/testutils"
	"github.com/safe-waters/docker-lock/pkg/generate"
	"github.com/safe-waters/docker-lock/pkg/generate/update"
	"github.com/safe-waters/docker-lock/pkg/kind"
	"github.com/safe-waters/docker-lock/pkg/verify"
	"github.com/safe-waters/docker-lock/pkg/verify/diff"
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
			Name: "Diff Num Images",
			Contents: [][]byte{
				[]byte(`
FROM busybox
FROM busybox
`,
				),
				// nolint: lll
				[]byte(`
{
	"dockerfiles": {
		"Dockerfile": [
			{
				"name": "busybox",
				"tag": "latest",
				"digest": "bae015c28bc7cdee3b7ef20d35db4299e3068554a769070950229d9f53f58572"
			}
		]
	}
}
`),
			},
			ShouldFail: true,
		},
		{
			Name: "Dockerfile Diff",
			Contents: [][]byte{
				[]byte(`
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
			ShouldFail: true,
		},
		{
			Name: "Kubernetesfile Diff",
			Contents: [][]byte{
				[]byte(`
apiVersion: v1
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
	"kubernetesfiles": {
		"pod.yaml": [
			{
			"name": "busybox",
			"tag": "latest",
			"digest": "busybox",
			"container": "busybox"
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
				// nolint: lll
				[]byte(`
{
	"composefiles": {
		"docker-compose.yml": [
			{
				"name": "busybox",
				"tag": "latest",
				"digest": "bae015c28bc7cdee3b7ef20d35db4299e3068554a769070950229d9f53f58572",
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
				// nolint: lll
				[]byte(`
{
	"composefiles": {
		"docker-compose.yml": [
			{
				"name": "busybox",
				"tag": "",
				"digest": "bae015c28bc7cdee3b7ef20d35db4299e3068554a769070950229d9f53f58572",
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
						image["dockerfile"] = filepath.ToSlash(
							filepath.Join(tempDir, dockerfilePath),
						)
					}
				}
				uniquePathsToWrite[composefilePath] = struct{}{}

				composefilePath = filepath.ToSlash(
					filepath.Join(tempDir, composefilePath),
				)
				composefileImagesWithTempDir[composefilePath] = images
			}

			dockerfileImagesWithTempDir := map[string][]interface{}{}

			for dockerfilePath, images := range lockfile[kind.Dockerfile] {
				uniquePathsToWrite[dockerfilePath] = struct{}{}

				dockerfilePath = filepath.ToSlash(
					filepath.Join(tempDir, dockerfilePath),
				)
				dockerfileImagesWithTempDir[dockerfilePath] = images
			}

			kubernetesfileImagesWithTempDir := map[string][]interface{}{}

			for kubernetesfilePath, images := range lockfile[kind.Kubernetesfile] { // nolint: lll
				uniquePathsToWrite[kubernetesfilePath] = struct{}{}

				kubernetesfilePath = filepath.ToSlash(
					filepath.Join(tempDir, kubernetesfilePath),
				)
				kubernetesfileImagesWithTempDir[kubernetesfilePath] = images
			}

			var pathsToWrite []string
			for path := range uniquePathsToWrite {
				pathsToWrite = append(pathsToWrite, path)
			}

			pathsToWrite = append(pathsToWrite, "docker-lock.json")

			lockfileWithTempDir := map[kind.Kind]map[string][]interface{}{}
			if len(dockerfileImagesWithTempDir) != 0 {
				lockfileWithTempDir[kind.Dockerfile] = dockerfileImagesWithTempDir // nolint: lll
			}

			if len(composefileImagesWithTempDir) != 0 {
				lockfileWithTempDir[kind.Composefile] = composefileImagesWithTempDir // nolint: lll
			}

			if len(kubernetesfileImagesWithTempDir) != 0 {
				lockfileWithTempDir[kind.Kubernetesfile] = kubernetesfileImagesWithTempDir // nolint: lll
			}

			lockfileWithTempDirByt, err := json.Marshal(lockfileWithTempDir)
			if err != nil {
				t.Fatal(err)
			}

			test.Contents[len(test.Contents)-1] = lockfileWithTempDirByt

			tempPaths := testutils.WriteFilesToTempDir(
				t, tempDir, pathsToWrite, test.Contents,
			)

			reader := bytes.NewReader(lockfileWithTempDirByt)

			flags := &cmd_verify.Flags{
				LockfileName: tempPaths[len(tempPaths)-1],
				ExcludeTags:  test.ExcludeTags,
			}

			existingLByt, err := ioutil.ReadFile(flags.LockfileName)
			if err != nil {
				t.Fatal(err)
			}

			var existingLockfile map[kind.Kind]map[string][]interface{}
			if err = json.Unmarshal(
				existingLByt, &existingLockfile,
			); err != nil {
				t.Fatal(err)
			}

			dockerfilePaths := make(
				[]string, len(existingLockfile[kind.Dockerfile]),
			)
			composefilePaths := make(
				[]string, len(existingLockfile[kind.Composefile]),
			)
			kubernetesfilePaths := make(
				[]string, len(existingLockfile[kind.Kubernetesfile]),
			)

			var i, j, k int

			for p := range existingLockfile[kind.Dockerfile] {
				dockerfilePaths[i] = p
				i++
			}

			for p := range existingLockfile[kind.Composefile] {
				composefilePaths[j] = p
				j++
			}

			for p := range existingLockfile[kind.Kubernetesfile] {
				kubernetesfilePaths[k] = p
				k++
			}

			generatorFlags, err := cmd_generate.NewFlags(
				".", "",
				flags.IgnoreMissingDigests, flags.UpdateExistingDigests,
				dockerfilePaths, composefilePaths,
				kubernetesfilePaths, nil, nil, nil, false, false, false,
				len(dockerfilePaths) == 0, len(composefilePaths) == 0,
				len(kubernetesfilePaths) == 0,
			)
			if err != nil {
				t.Fatal(err)
			}

			collector, err := cmd_generate.DefaultPathCollector(generatorFlags)
			if err != nil {
				t.Fatal(err)
			}

			parser, err := cmd_generate.DefaultImageParser(generatorFlags)
			if err != nil {
				t.Fatal(err)
			}

			digestRequester := testutils.NewMockDigestRequester(t, nil)

			imageDigestUpdater, err := update.NewImageDigestUpdater(
				digestRequester,
				generatorFlags.FlagsWithSharedValues.IgnoreMissingDigests,
				generatorFlags.FlagsWithSharedValues.UpdateExistingDigests,
			)
			if err != nil {
				t.Fatal(err)
			}

			updater, err := generate.NewImageDigestUpdater(imageDigestUpdater)
			if err != nil {
				t.Fatal(err)
			}

			sorter, err := cmd_generate.DefaultImageFormatter(generatorFlags)
			if err != nil {
				t.Fatal(err)
			}

			generator, err := generate.NewGenerator(
				collector, parser, updater, sorter,
			)
			if err != nil {
				t.Fatal(err)
			}

			dockerfileDifferentiator := diff.NewDockerfileDifferentiator(
				flags.ExcludeTags,
			)

			composefileDifferentiator := diff.NewComposefileDifferentiator(
				flags.ExcludeTags,
			)

			kubernetesfileDifferentiator := diff.NewKubernetesfileDifferentiator( // nolint: lll
				flags.ExcludeTags,
			)

			verifier, err := verify.NewVerifier(
				generator, dockerfileDifferentiator, composefileDifferentiator,
				kubernetesfileDifferentiator,
			)
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
