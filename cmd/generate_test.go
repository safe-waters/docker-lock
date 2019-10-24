package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/michaelperel/docker-lock/generate"
)

var composeBaseDir = filepath.Join("testdata", "generate", "compose")
var dockerBaseDir = filepath.Join("testdata", "generate", "docker")

// docker-compose files
func TestGenerateComposeImage(t *testing.T) {
	t.Parallel()
	composefile := filepath.Join(composeBaseDir, "image", "docker-compose.yml")
	flags := []string{fmt.Sprintf("--compose-files=%s", composefile)}
	results := map[string][]generate.ComposefileImage{filepath.ToSlash(composefile): []generate.ComposefileImage{
		{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "svc", Dockerfile: ""},
	}}
	testGenerate(t, flags, results)
}

func TestGenerateComposeBuild(t *testing.T) {
	t.Parallel()
	composefile := filepath.Join(composeBaseDir, "build", "docker-compose.yml")
	dockerfile := filepath.ToSlash(filepath.Join(composeBaseDir, "build", "build", "Dockerfile"))
	flags := []string{fmt.Sprintf("--compose-files=%s", composefile)}
	results := map[string][]generate.ComposefileImage{filepath.ToSlash(composefile): []generate.ComposefileImage{
		{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "svc", Dockerfile: dockerfile},
	}}
	testGenerate(t, flags, results)
}

func TestGenerateComposeDockerfile(t *testing.T) {
	t.Parallel()
	composefile := filepath.Join(composeBaseDir, "dockerfile", "docker-compose.yml")
	dockerfile := filepath.ToSlash(filepath.Join(composeBaseDir, "dockerfile", "dockerfile", "Dockerfile"))
	flags := []string{fmt.Sprintf("--compose-files=%s", composefile)}
	results := map[string][]generate.ComposefileImage{filepath.ToSlash(composefile): []generate.ComposefileImage{
		{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "svc", Dockerfile: dockerfile},
	}}
	testGenerate(t, flags, results)
}

func TestGenerateComposeContext(t *testing.T) {
	t.Parallel()
	composefile := filepath.Join(composeBaseDir, "context", "docker-compose.yml")
	dockerfile := filepath.ToSlash(filepath.Join(composeBaseDir, "context", "context", "Dockerfile"))
	flags := []string{fmt.Sprintf("--compose-files=%s", composefile)}
	results := map[string][]generate.ComposefileImage{filepath.ToSlash(composefile): []generate.ComposefileImage{
		{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "svc", Dockerfile: dockerfile},
	}}
	testGenerate(t, flags, results)
}

func TestGenerateComposeEnv(t *testing.T) {
	t.Parallel()
	composefile := filepath.Join(composeBaseDir, "env", "docker-compose.yml")
	dockerfile := filepath.ToSlash(filepath.Join(composeBaseDir, "env", "env", "Dockerfile"))
	envFile := filepath.Join(composeBaseDir, "env", ".env")
	flags := []string{fmt.Sprintf("--compose-files=%s", composefile), fmt.Sprintf("--env-file=%s", envFile)}
	results := map[string][]generate.ComposefileImage{filepath.ToSlash(composefile): []generate.ComposefileImage{
		{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "svc", Dockerfile: dockerfile},
	}}
	testGenerate(t, flags, results)
}

func TestGenerateComposeMultipleComposefiles(t *testing.T) {
	t.Parallel()
	composefileOne := filepath.Join(composeBaseDir, "multiple", "docker-compose-one.yml")
	composefileTwo := filepath.Join(composeBaseDir, "multiple", "docker-compose-two.yml")
	dockerfilesOne := []string{filepath.ToSlash(filepath.Join(composeBaseDir, "multiple", "build", "Dockerfile"))}
	dockerfilesTwo := []string{
		filepath.ToSlash(filepath.Join(composeBaseDir, "multiple", "context", "Dockerfile")),
		filepath.ToSlash(filepath.Join(composeBaseDir, "multiple", "dockerfile", "Dockerfile")),
	}
	composefiles := strings.Join([]string{composefileOne, composefileTwo}, ",")
	flags := []string{fmt.Sprintf("--compose-files=%s", composefiles)}
	results := map[string][]generate.ComposefileImage{
		filepath.ToSlash(composefileOne): []generate.ComposefileImage{
			{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "build-svc", Dockerfile: dockerfilesOne[0]},
			{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "image-svc", Dockerfile: ""},
		},
		filepath.ToSlash(composefileTwo): []generate.ComposefileImage{
			{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "context-svc", Dockerfile: dockerfilesTwo[0]},
			{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "dockerfile-svc", Dockerfile: dockerfilesTwo[1]},
		}}
	testGenerate(t, flags, results)
}

func TestGenerateComposeRecursive(t *testing.T) {
	t.Parallel()
	composefileTopLevel := filepath.Join(composeBaseDir, "recursive", "docker-compose.yml")
	composefileRecursiveLevel := filepath.Join(composeBaseDir, "recursive", "build", "docker-compose.yml")
	dockerfileRecursiveLevel := filepath.ToSlash(filepath.Join(composeBaseDir, "recursive", "build", "build", "Dockerfile"))
	recursiveBaseDir := filepath.Join(composeBaseDir, "recursive")
	flags := []string{fmt.Sprintf("--base-dir=%s", recursiveBaseDir), "--compose-file-recursive"}
	results := map[string][]generate.ComposefileImage{
		filepath.ToSlash(composefileTopLevel): []generate.ComposefileImage{
			{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "svc", Dockerfile: ""},
		},
		filepath.ToSlash(composefileRecursiveLevel): []generate.ComposefileImage{
			{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "svc", Dockerfile: dockerfileRecursiveLevel},
		},
	}
	testGenerate(t, flags, results)
}

func TestGenerateComposeNoFileSpecified(t *testing.T) {
	t.Parallel()
	baseDir := filepath.Join(composeBaseDir, "nofile")
	flags := []string{fmt.Sprintf("--base-dir=%s", baseDir)}
	composefiles := []string{filepath.Join(baseDir, "docker-compose.yml"), filepath.Join(baseDir, "docker-compose.yaml")}
	results := make(map[string][]generate.ComposefileImage)
	for _, composefile := range composefiles {
		results[filepath.ToSlash(composefile)] = append(results[filepath.ToSlash(composefile)], generate.ComposefileImage{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "svc", Dockerfile: ""})
	}
	testGenerate(t, flags, results)
}

func TestGenerateComposeGlobs(t *testing.T) {
	t.Parallel()
	globs := strings.Join([]string{filepath.Join(composeBaseDir, "globs", "**", "docker-compose.yml"), filepath.Join(composeBaseDir, "globs", "docker-compose.yml")}, ",")
	flags := []string{fmt.Sprintf("--compose-file-globs=%s", globs)}
	composefiles := []string{filepath.Join(composeBaseDir, "globs", "image", "docker-compose.yml"), filepath.Join(composeBaseDir, "globs", "docker-compose.yml")}
	results := make(map[string][]generate.ComposefileImage)
	for _, composefile := range composefiles {
		results[filepath.ToSlash(composefile)] = append(results[filepath.ToSlash(composefile)], generate.ComposefileImage{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "svc", Dockerfile: ""})
	}
	testGenerate(t, flags, results)
}

func TestGenerateComposeAssortment(t *testing.T) {
	t.Parallel()
	composefile := filepath.Join(composeBaseDir, "assortment", "docker-compose.yml")
	dockerfiles := []string{
		filepath.ToSlash(filepath.Join(composeBaseDir, "assortment", "build", "Dockerfile")),
		filepath.ToSlash(filepath.Join(composeBaseDir, "assortment", "context", "Dockerfile")),
		filepath.ToSlash(filepath.Join(composeBaseDir, "assortment", "dockerfile", "Dockerfile")),
	}
	flags := []string{fmt.Sprintf("--compose-files=%s", composefile)}
	results := map[string][]generate.ComposefileImage{filepath.ToSlash(composefile): []generate.ComposefileImage{
		{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "build-svc", Dockerfile: dockerfiles[0]},
		{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "context-svc", Dockerfile: dockerfiles[1]},
		{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "dockerfile-svc", Dockerfile: dockerfiles[2]},
		{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "image-svc", Dockerfile: ""},
	}}
	testGenerate(t, flags, results)
}

func TestGenerateComposeArgsDockerfileOverride(t *testing.T) {
	// ARG in Dockerfile, also in composefile.
	// composefile should override
	t.Parallel()
	composefile := filepath.Join(composeBaseDir, "args", "override", "docker-compose.yml")
	dockerfile := filepath.ToSlash(filepath.Join(composeBaseDir, "args", "override", "Dockerfile"))
	flags := []string{fmt.Sprintf("--compose-files=%s", composefile)}
	results := map[string][]generate.ComposefileImage{filepath.ToSlash(composefile): []generate.ComposefileImage{
		{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "svc", Dockerfile: dockerfile},
	}}
	testGenerate(t, flags, results)
}

func TestGenerateComposeArgsDockerfileEmpty(t *testing.T) {
	// Empty ARG in Dockerfile, definition in composefile.
	// composefile should override.
	t.Parallel()
	composefile := filepath.Join(composeBaseDir, "args", "empty", "docker-compose.yml")
	dockerfile := filepath.ToSlash(filepath.Join(composeBaseDir, "args", "empty", "Dockerfile"))
	flags := []string{fmt.Sprintf("--compose-files=%s", composefile)}
	results := map[string][]generate.ComposefileImage{filepath.ToSlash(composefile): []generate.ComposefileImage{
		{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "svc", Dockerfile: dockerfile},
	}}
	testGenerate(t, flags, results)
}

func TestGenerateComposeArgsDockerfileNoArg(t *testing.T) {
	// ARG defined in Dockerfile, not in composefile.
	// Should behave as though no composefile existed.
	t.Parallel()
	composefile := filepath.Join(composeBaseDir, "args", "noarg", "docker-compose.yml")
	dockerfile := filepath.ToSlash(filepath.Join(composeBaseDir, "args", "noarg", "Dockerfile"))
	flags := []string{fmt.Sprintf("--compose-files=%s", composefile)}
	results := map[string][]generate.ComposefileImage{filepath.ToSlash(composefile): []generate.ComposefileImage{
		{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "svc", Dockerfile: dockerfile},
	}}
	testGenerate(t, flags, results)
}

// Dockerfiles
func TestGenerateDockerfileArgsBuildStage(t *testing.T) {
	// Build stages should not be parsed.
	// For instance:
	// # Dockerfile
	// FROM busybox AS busy
	// FROM busy AS anotherbusy
	// should only parse 'busybox', the second field in the first line.
	t.Parallel()
	dockerfile := filepath.Join(dockerBaseDir, "args", "buildstage", "Dockerfile")
	flags := []string{fmt.Sprintf("--dockerfiles=%s", dockerfile)}
	results := map[string][]generate.DockerfileImage{filepath.ToSlash(dockerfile): []generate.DockerfileImage{
		{Image: generate.Image{Name: "busybox", Tag: "latest"}},
		{Image: generate.Image{Name: "busybox", Tag: "latest"}},
	}}
	testGenerate(t, flags, results)
}

func TestGenerateDockerfileArgsLocalArg(t *testing.T) {
	// ARG defined before FROM (aka global arg) should not
	// be overridden by ARG defined after FROM (aka local arg)
	t.Parallel()
	dockerfile := filepath.Join(dockerBaseDir, "args", "localarg", "Dockerfile")
	flags := []string{fmt.Sprintf("--dockerfiles=%s", dockerfile)}
	results := map[string][]generate.DockerfileImage{filepath.ToSlash(dockerfile): []generate.DockerfileImage{
		{Image: generate.Image{Name: "busybox", Tag: "latest"}},
		{Image: generate.Image{Name: "busybox", Tag: "latest"}},
	}}
	testGenerate(t, flags, results)
}

func TestGenerateDockerfileMultipleDockerfiles(t *testing.T) {
	t.Parallel()
	dockerfiles := []string{filepath.Join(dockerBaseDir, "multiple", "DockerfileOne"), filepath.Join(dockerBaseDir, "multiple", "DockerfileTwo")}
	flags := []string{fmt.Sprintf("--dockerfiles=%s", strings.Join(dockerfiles, ","))}
	results := make(map[string][]generate.DockerfileImage)
	for _, dockerfile := range dockerfiles {
		results[filepath.ToSlash(dockerfile)] = append(results[filepath.ToSlash(dockerfile)], generate.DockerfileImage{Image: generate.Image{Name: "busybox", Tag: "latest"}})
	}
	testGenerate(t, flags, results)
}

func TestGenerateDockerfileRecursive(t *testing.T) {
	t.Parallel()
	recursiveBaseDir := filepath.Join(dockerBaseDir, "recursive")
	flags := []string{fmt.Sprintf("--base-dir=%s", recursiveBaseDir), "--dockerfile-recursive"}
	results := make(map[string][]generate.DockerfileImage)
	dockerfiles := []string{filepath.Join(dockerBaseDir, "recursive", "Dockerfile"), filepath.Join(dockerBaseDir, "recursive", "recursive", "Dockerfile")}
	for _, dockerfile := range dockerfiles {
		results[filepath.ToSlash(dockerfile)] = append(results[filepath.ToSlash(dockerfile)], generate.DockerfileImage{Image: generate.Image{Name: "busybox", Tag: "latest"}})
	}
	testGenerate(t, flags, results)
}

func TestGenerateDockerfileNoFileSpecified(t *testing.T) {
	t.Parallel()
	baseDir := filepath.Join(dockerBaseDir, "nofile")
	flags := []string{fmt.Sprintf("--base-dir=%s", baseDir)}
	dockerfile := filepath.Join(baseDir, "Dockerfile")
	results := map[string][]generate.DockerfileImage{filepath.ToSlash(dockerfile): []generate.DockerfileImage{
		{Image: generate.Image{Name: "busybox", Tag: "latest"}},
	}}
	testGenerate(t, flags, results)
}

func TestGenerateDockerfileGlobs(t *testing.T) {
	t.Parallel()
	globs := strings.Join([]string{filepath.Join(dockerBaseDir, "globs", "**", "Dockerfile"), filepath.Join(dockerBaseDir, "globs", "Dockerfile")}, ",")
	flags := []string{fmt.Sprintf("--dockerfile-globs=%s", globs)}
	dockerfiles := []string{filepath.Join(dockerBaseDir, "globs", "globs", "Dockerfile"), filepath.Join(dockerBaseDir, "globs", "Dockerfile")}
	results := make(map[string][]generate.DockerfileImage)
	for _, dockerfile := range dockerfiles {
		results[filepath.ToSlash(dockerfile)] = append(results[filepath.ToSlash(dockerfile)], generate.DockerfileImage{Image: generate.Image{Name: "busybox", Tag: "latest"}})
	}
	testGenerate(t, flags, results)
}

func TestGenerateDockerfilePrivate(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI_SERVER") != "TRUE" {
		t.Skip("Only runs on CI server.")
	}
	dockerfile := filepath.Join(dockerBaseDir, "private", "Dockerfile")
	flags := []string{fmt.Sprintf("--dockerfiles=%s", dockerfile)}
	results := map[string][]generate.DockerfileImage{filepath.ToSlash(dockerfile): []generate.DockerfileImage{
		{Image: generate.Image{Name: "dockerlocktestaccount/busybox", Tag: "latest"}},
	}}
	testGenerate(t, flags, results)
}

// helpers
func testGenerate(t *testing.T, flags []string, results interface{}) {
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
	switch r := results.(type) {
	case map[string][]generate.ComposefileImage:
		checkComposeResults(t, r, lFile)
	case map[string][]generate.DockerfileImage:
		checkDockerResults(t, r, lFile)
	default:
		t.Fatalf("Incorrect result type: %v", r)
	}
}

func checkComposeResults(t *testing.T, results map[string][]generate.ComposefileImage, lFile generate.Lockfile) {
	if len(lFile.ComposefileImages) != len(results) {
		t.Errorf("Found '%d' docker-compose files. Expected '%d'.", len(lFile.ComposefileImages), len(results))
	}
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
