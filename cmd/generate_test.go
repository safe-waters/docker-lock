package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/michaelperel/docker-lock/generate"
)

var composeBaseDir = filepath.Join("testdata", "generate", "compose")

func testCompose(t *testing.T, flags []string, results map[string][]generate.ComposefileImage) {
	tmpFile, err := ioutil.TempFile("", "test-docker-lock-*")
	if err != nil {
		t.Error(err)
	}
	defer os.Remove(tmpFile.Name())
	outPath := tmpFile.Name()
	generateCmd := NewGenerateCmd()
	args := append([]string{"lock", "generate", fmt.Sprintf("--outpath=%s", outPath)}, flags...)
	generateCmd.SetArgs(args)
	if err := generateCmd.Execute(); err != nil {
		t.Error(err)
	}
	lByt, err := ioutil.ReadFile(outPath)
	if err != nil {
		t.Error(err)
	}
	var lFile generate.Lockfile
	if err := json.Unmarshal(lByt, &lFile); err != nil {
		t.Error(err)
	}
	if len(lFile.ComposefileImages) != len(results) {
		t.Errorf("Found '%d' docker-compose files. Expected '%d'.", len(lFile.ComposefileImages), len(results))
	}
	// TODO: PRETTY
	for rCFile, rImages := range results {
		fImages, ok := lFile.ComposefileImages[rCFile]
		if !ok {
			t.Errorf("Expected '%s' key, but did not find it.", rCFile)
		}
		if len(fImages) != len(rImages) {
			t.Errorf("Found '%d' images for '%s'. Expected '%d'.", len(fImages), rCFile, len(rImages))
		}
		for i, fImage := range fImages {
			if results[rCFile][i].Image.Name != fImage.Image.Name ||
				results[rCFile][i].Image.Tag != fImage.Image.Tag {
				t.Errorf("Found '%s:%s'. Expected '%s:%s'.",
					fImage.Image.Name,
					fImage.Image.Tag,
					results[rCFile][i].Image.Name,
					results[rCFile][i].Image.Tag)
			}
			if fImage.Image.Digest == "" {
				t.Errorf("%+v has an empty digest.", fImage)
			}
			if fImage.ServiceName != results[rCFile][i].ServiceName {
				t.Errorf("Found '%s' service. Expected '%s'.",
					fImage.ServiceName,
					results[rCFile][i].ServiceName)
			}
			if fImage.Dockerfile != results[rCFile][i].Dockerfile {
				t.Errorf("Found '%s' dockerfile. Expected '%s'.",
					filepath.FromSlash(fImage.Dockerfile),
					results[rCFile][i].Dockerfile)
			}
		}
	}
}

// docker-compose files
func TestComposeImage(t *testing.T) {
	t.Parallel()
	composefile := filepath.Join(composeBaseDir, "image", "docker-compose.yml")
	flags := []string{fmt.Sprintf("--compose-files=%s", composefile)}
	results := map[string][]generate.ComposefileImage{filepath.ToSlash(composefile): []generate.ComposefileImage{
		{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "svc", Dockerfile: ""},
	}}
	testCompose(t, flags, results)
}

func TestComposeBuild(t *testing.T) {
	t.Parallel()
	composefile := filepath.Join(composeBaseDir, "build", "docker-compose.yml")
	dockerfile := filepath.ToSlash(filepath.Join(composeBaseDir, "build", "build", "Dockerfile"))
	flags := []string{fmt.Sprintf("--compose-files=%s", composefile)}
	results := map[string][]generate.ComposefileImage{filepath.ToSlash(composefile): []generate.ComposefileImage{
		{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "svc", Dockerfile: dockerfile},
	}}
	testCompose(t, flags, results)
}

func TestComposeDockerfile(t *testing.T) {
	t.Parallel()
	composefile := filepath.Join(composeBaseDir, "dockerfile", "docker-compose.yml")
	dockerfile := filepath.ToSlash(filepath.Join(composeBaseDir, "dockerfile", "dockerfile", "Dockerfile"))
	flags := []string{fmt.Sprintf("--compose-files=%s", composefile)}
	results := map[string][]generate.ComposefileImage{filepath.ToSlash(composefile): []generate.ComposefileImage{
		{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "svc", Dockerfile: dockerfile},
	}}
	testCompose(t, flags, results)
}

func TestComposeContext(t *testing.T) {
	t.Parallel()
	composefile := filepath.Join(composeBaseDir, "context", "docker-compose.yml")
	dockerfile := filepath.ToSlash(filepath.Join(composeBaseDir, "context", "context", "Dockerfile"))
	flags := []string{fmt.Sprintf("--compose-files=%s", composefile)}
	results := map[string][]generate.ComposefileImage{filepath.ToSlash(composefile): []generate.ComposefileImage{
		{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "svc", Dockerfile: dockerfile},
	}}
	testCompose(t, flags, results)
}

func TestComposeEnv(t *testing.T) {
	t.Parallel()
	composefile := filepath.Join(composeBaseDir, "env", "docker-compose.yml")
	dockerfile := filepath.ToSlash(filepath.Join(composeBaseDir, "env", "env", "Dockerfile"))
	envFile := filepath.Join(composeBaseDir, "env", ".env")
	flags := []string{fmt.Sprintf("--compose-files=%s", composefile), fmt.Sprintf("--env-file=%s", envFile)}
	results := map[string][]generate.ComposefileImage{filepath.ToSlash(composefile): []generate.ComposefileImage{
		{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "svc", Dockerfile: dockerfile},
	}}
	testCompose(t, flags, results)
}

func TestComposeArgsDockerfileOverride(t *testing.T) {
	// ARG in Dockerfile, also in composefile.
	// composefile should override
	t.Parallel()
	composefile := filepath.Join(composeBaseDir, "args", "override", "docker-compose.yml")
	dockerfile := filepath.ToSlash(filepath.Join(composeBaseDir, "args", "override", "Dockerfile"))
	flags := []string{fmt.Sprintf("--compose-files=%s", composefile)}
	results := map[string][]generate.ComposefileImage{filepath.ToSlash(composefile): []generate.ComposefileImage{
		{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "svc", Dockerfile: dockerfile},
	}}
	testCompose(t, flags, results)
}

func TestComposeArgsDockerfileEmpty(t *testing.T) {
	// Empty ARG in Dockerfile, definition in composefile.
	// composefile should override.
	t.Parallel()
	composefile := filepath.Join(composeBaseDir, "args", "empty", "docker-compose.yml")
	dockerfile := filepath.ToSlash(filepath.Join(composeBaseDir, "args", "empty", "Dockerfile"))
	flags := []string{fmt.Sprintf("--compose-files=%s", composefile)}
	results := map[string][]generate.ComposefileImage{filepath.ToSlash(composefile): []generate.ComposefileImage{
		{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "svc", Dockerfile: dockerfile},
	}}
	testCompose(t, flags, results)
}

func TestComposeArgsDockerfileNoArg(t *testing.T) {
	// ARG defined in Dockerfile, not in composefile.
	// Should behave as though no composefile existed.
	t.Parallel()
	composefile := filepath.Join(composeBaseDir, "args", "noarg", "docker-compose.yml")
	dockerfile := filepath.ToSlash(filepath.Join(composeBaseDir, "args", "noarg", "Dockerfile"))
	flags := []string{fmt.Sprintf("--compose-files=%s", composefile)}
	results := map[string][]generate.ComposefileImage{filepath.ToSlash(composefile): []generate.ComposefileImage{
		{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "svc", Dockerfile: dockerfile},
	}}
	testCompose(t, flags, results)
}

// Dockerfiles
func TestDockerfileArgsBuildStage(t *testing.T) {
	// Build stages should not be parsed.
	// For instance:
	// # Dockerfile
	// FROM busybox AS busy
	// FROM busy AS anotherbusy
	// should only parse 'busybox', the second field in the first line.
	t.Parallel()
	baseDir := filepath.Join("testdata", "generate", "docker", "args")
	tmpFile, err := ioutil.TempFile("", "test-dockerfile-args-build-stage")
	if err != nil {
		t.Error(err)
	}
	generateCmd := NewGenerateCmd()
	generateCmd.SetArgs([]string{
		"lock",
		"generate",
		fmt.Sprintf("--dockerfiles=%s", filepath.Join(baseDir, "buildstage", "Dockerfile")),
		fmt.Sprintf("--outpath=%s", tmpFile.Name()),
	})
	generateCmd.Execute()
	outPath, err := generateCmd.Flags().GetString("outpath")
	if err != nil {
		t.Error(err)
	}
	dockerfiles, err := generateCmd.Flags().GetStringSlice("dockerfiles")
	if err != nil {
		t.Error(err)
	}
	lByt, err := ioutil.ReadFile(outPath)
	if err != nil {
		t.Error(err)
	}
	var lFile generate.Lockfile
	if err := json.Unmarshal(lByt, &lFile); err != nil {
		t.Error(err)
	}
	dockerfile := filepath.ToSlash(dockerfiles[0])

	results := map[string][]generate.DockerfileImage{dockerfile: []generate.DockerfileImage{
		{Image: generate.Image{Name: "busybox", Tag: "latest"}},
		{Image: generate.Image{Name: "busybox", Tag: "latest"}},
	}}

	checkDockerResults(t, results, lFile)
}

func TestDockerfileArgsLocalArg(t *testing.T) {
	// ARG defined before FROM (aka global arg) should not
	// be overridden by ARG defined after FROM (aka local arg)
	t.Parallel()
	baseDir := filepath.Join("testdata", "generate", "docker", "args")
	tmpFile, err := ioutil.TempFile("", "test-dockerfile-args-local-arg")
	if err != nil {
		t.Error(err)
	}
	generateCmd := NewGenerateCmd()
	generateCmd.SetArgs([]string{
		"lock",
		"generate",
		fmt.Sprintf("--dockerfiles=%s", filepath.Join(baseDir, "localarg", "Dockerfile")),
		fmt.Sprintf("--outpath=%s", tmpFile.Name()),
	})
	generateCmd.Execute()
	outPath, err := generateCmd.Flags().GetString("outpath")
	if err != nil {
		t.Error(err)
	}
	dockerfiles, err := generateCmd.Flags().GetStringSlice("dockerfiles")
	if err != nil {
		t.Error(err)
	}
	lByt, err := ioutil.ReadFile(outPath)
	if err != nil {
		t.Error(err)
	}
	var lFile generate.Lockfile
	if err := json.Unmarshal(lByt, &lFile); err != nil {
		t.Error(err)
	}
	dockerfile := filepath.ToSlash(dockerfiles[0])

	results := map[string][]generate.DockerfileImage{dockerfile: []generate.DockerfileImage{
		{Image: generate.Image{Name: "busybox", Tag: "latest"}},
		{Image: generate.Image{Name: "busybox", Tag: "latest"}},
	}}

	checkDockerResults(t, results, lFile)
}

func TestDockerfilePrivate(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI_SERVER") != "TRUE" {
		t.Skip("Only runs on CI server.")
	}
	baseDir := filepath.Join("testdata", "generate", "docker")
	tmpFile, err := ioutil.TempFile("", "test-dockerfile-private")
	if err != nil {
		t.Error(err)
	}
	generateCmd := NewGenerateCmd()
	generateCmd.SetArgs([]string{
		"lock",
		"generate",
		fmt.Sprintf("--dockerfiles=%s", filepath.Join(baseDir, "private", "Dockerfile")),
		fmt.Sprintf("--outpath=%s", tmpFile.Name()),
	})
	generateCmd.Execute()
	outPath, err := generateCmd.Flags().GetString("outpath")
	if err != nil {
		t.Error(err)
	}
	dockerfiles, err := generateCmd.Flags().GetStringSlice("dockerfiles")
	if err != nil {
		t.Error(err)
	}
	lByt, err := ioutil.ReadFile(outPath)
	if err != nil {
		t.Error(err)
	}
	var lFile generate.Lockfile
	if err := json.Unmarshal(lByt, &lFile); err != nil {
		t.Error(err)
	}
	dockerfile := filepath.ToSlash(dockerfiles[0])
	results := map[string][]generate.DockerfileImage{dockerfile: []generate.DockerfileImage{
		{Image: generate.Image{Name: "dockerlocktestaccount/busybox", Tag: "latest"}},
	}}
	checkDockerResults(t, results, lFile)
}

func checkDockerResults(t *testing.T, results map[string][]generate.DockerfileImage, lFile generate.Lockfile) {
	if len(lFile.DockerfileImages) != len(results) {
		t.Errorf("Found '%d' Dockerfiles. Expected '%d'.", len(lFile.DockerfileImages), len(results))
	}
	for rDFile, rImages := range results {
		fImages, ok := lFile.DockerfileImages[rDFile]
		if !ok {
			t.Errorf("Expected '%s' key, but did not find it.", rDFile)
		}
		if len(fImages) != len(rImages) {
			t.Errorf("Found '%d' images for '%s'. Expected '%d'.", len(fImages), rDFile, len(rImages))
		}

		for i, fImage := range fImages {
			if results[rDFile][i].Image.Name != fImage.Image.Name ||
				results[rDFile][i].Image.Tag != fImage.Image.Tag {
				t.Errorf("Found '%s:%s'. Expected '%s:%s'.",
					fImage.Image.Name,
					fImage.Image.Tag,
					results[rDFile][i].Image.Name,
					results[rDFile][i].Image.Tag)
			}
			if fImage.Image.Digest == "" {
				t.Errorf("%+v has an empty digest.", fImage)
			}
		}
	}
}
