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

var generateComposeBaseDir = filepath.Join("testdata", "generate", "compose")
var generateDockerBaseDir = filepath.Join("testdata", "generate", "docker")
var generateBothBaseDir = filepath.Join("testdata", "generate", "both")

// TestGenerateComposefileImage ensures Lockfiles from docker-compose files with
// the image key are correct.
func TestGenerateComposefileImage(t *testing.T) {
	t.Parallel()
	composefile := filepath.Join(generateComposeBaseDir, "image", "docker-compose.yml")
	flags := []string{fmt.Sprintf("--compose-files=%s", composefile)}
	results := map[string][]generate.ComposefileImage{filepath.ToSlash(composefile): []generate.ComposefileImage{
		{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "svc", Dockerfile: ""},
	}}
	testGenerate(t, flags, results)
}

// TestGenerateComposefileBuild ensures Lockfiles from docker-compose files with
// the build key are correct.
func TestGenerateComposefileBuild(t *testing.T) {
	t.Parallel()
	composefile := filepath.Join(generateComposeBaseDir, "build", "docker-compose.yml")
	dockerfile := filepath.ToSlash(filepath.Join(generateComposeBaseDir, "build", "build", "Dockerfile"))
	flags := []string{fmt.Sprintf("--compose-files=%s", composefile)}
	results := map[string][]generate.ComposefileImage{filepath.ToSlash(composefile): []generate.ComposefileImage{
		{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "svc", Dockerfile: dockerfile},
	}}
	testGenerate(t, flags, results)
}

// TestGenerateComposefileDockerfile ensures Lockfiles from docker-compose files with
// the dockerfile key are correct.
func TestGenerateComposefileDockerfile(t *testing.T) {
	t.Parallel()
	composefile := filepath.Join(generateComposeBaseDir, "dockerfile", "docker-compose.yml")
	dockerfile := filepath.ToSlash(filepath.Join(generateComposeBaseDir, "dockerfile", "dockerfile", "Dockerfile"))
	flags := []string{fmt.Sprintf("--compose-files=%s", composefile)}
	results := map[string][]generate.ComposefileImage{filepath.ToSlash(composefile): []generate.ComposefileImage{
		{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "svc", Dockerfile: dockerfile},
	}}
	testGenerate(t, flags, results)
}

// TestGenerateComposefileContext ensures Lockfiles from docker-compose files with
// the context key are correct.
func TestGenerateComposefileContext(t *testing.T) {
	t.Parallel()
	composefile := filepath.Join(generateComposeBaseDir, "context", "docker-compose.yml")
	dockerfile := filepath.ToSlash(filepath.Join(generateComposeBaseDir, "context", "context", "Dockerfile"))
	flags := []string{fmt.Sprintf("--compose-files=%s", composefile)}
	results := map[string][]generate.ComposefileImage{filepath.ToSlash(composefile): []generate.ComposefileImage{
		{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "svc", Dockerfile: dockerfile},
	}}
	testGenerate(t, flags, results)
}

// TestGenerateComposefileEnv ensures Lockfiles from docker-compose files with
// environment variables replaced by values in a .env file are correct.
func TestGenerateComposefileEnv(t *testing.T) {
	t.Parallel()
	composefile := filepath.Join(generateComposeBaseDir, "env", "docker-compose.yml")
	dockerfile := filepath.ToSlash(filepath.Join(generateComposeBaseDir, "env", "env", "Dockerfile"))
	envFile := filepath.Join(generateComposeBaseDir, "env", ".env")
	flags := []string{fmt.Sprintf("--compose-files=%s", composefile), fmt.Sprintf("--env-file=%s", envFile)}
	results := map[string][]generate.ComposefileImage{filepath.ToSlash(composefile): []generate.ComposefileImage{
		{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "svc", Dockerfile: dockerfile},
	}}
	testGenerate(t, flags, results)
}

// TestGenerateComposefileMultipleComposefiles ensures Lockfiles from multiple
// docker-compose files are correct.
func TestGenerateComposefileMultipleComposefiles(t *testing.T) {
	t.Parallel()
	composefileOne := filepath.Join(generateComposeBaseDir, "multiple", "docker-compose-one.yml")
	composefileTwo := filepath.Join(generateComposeBaseDir, "multiple", "docker-compose-two.yml")
	dockerfilesOne := []string{filepath.ToSlash(filepath.Join(generateComposeBaseDir, "multiple", "build", "Dockerfile"))}
	dockerfilesTwo := []string{
		filepath.ToSlash(filepath.Join(generateComposeBaseDir, "multiple", "context", "Dockerfile")),
		filepath.ToSlash(filepath.Join(generateComposeBaseDir, "multiple", "dockerfile", "Dockerfile")),
	}
	flags := []string{fmt.Sprintf("--compose-files=%s,%s", composefileOne, composefileTwo)}
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

// TestGenerateComposefileRecursive ensures Lockfiles from multiple docker-compose
// files in subdirectories are correct.
func TestGenerateComposefileRecursive(t *testing.T) {
	t.Parallel()
	composefileTopLevel := filepath.Join(generateComposeBaseDir, "recursive", "docker-compose.yml")
	composefileRecursiveLevel := filepath.Join(generateComposeBaseDir, "recursive", "build", "docker-compose.yml")
	dockerfileRecursiveLevel := filepath.ToSlash(filepath.Join(generateComposeBaseDir, "recursive", "build", "build", "Dockerfile"))
	recursiveBaseDir := filepath.Join(generateComposeBaseDir, "recursive")
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

// TestGenerateComposefileNoFileSpecified ensures Lockfiles include docker-compose.yml
// and docker-compose.yaml files in the base directory, if no other files are specified.
func TestGenerateComposefileNoFileSpecified(t *testing.T) {
	t.Parallel()
	baseDir := filepath.Join(generateComposeBaseDir, "nofile")
	flags := []string{fmt.Sprintf("--base-dir=%s", baseDir)}
	composefiles := []string{filepath.Join(baseDir, "docker-compose.yml"), filepath.Join(baseDir, "docker-compose.yaml")}
	results := make(map[string][]generate.ComposefileImage)
	for _, composefile := range composefiles {
		results[filepath.ToSlash(composefile)] = append(results[filepath.ToSlash(composefile)], generate.ComposefileImage{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "svc", Dockerfile: ""})
	}
	testGenerate(t, flags, results)
}

// TestGenerateComposefileGlobs ensures Lockfiles include docker-compose files found
// via glob syntax.
func TestGenerateComposefileGlobs(t *testing.T) {
	t.Parallel()
	globs := strings.Join([]string{filepath.Join(generateComposeBaseDir, "globs", "**", "docker-compose.yml"), filepath.Join(generateComposeBaseDir, "globs", "docker-compose.yml")}, ",")
	flags := []string{fmt.Sprintf("--compose-file-globs=%s", globs)}
	composefiles := []string{filepath.Join(generateComposeBaseDir, "globs", "image", "docker-compose.yml"), filepath.Join(generateComposeBaseDir, "globs", "docker-compose.yml")}
	results := make(map[string][]generate.ComposefileImage)
	for _, composefile := range composefiles {
		results[filepath.ToSlash(composefile)] = append(results[filepath.ToSlash(composefile)], generate.ComposefileImage{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "svc", Dockerfile: ""})
	}
	testGenerate(t, flags, results)
}

// TestGenerateComposefileAssortment ensures that Lockfiles with an assortment of keys
// are correct.
func TestGenerateComposefileAssortment(t *testing.T) {
	t.Parallel()
	composefile := filepath.Join(generateComposeBaseDir, "assortment", "docker-compose.yml")
	dockerfiles := []string{
		filepath.ToSlash(filepath.Join(generateComposeBaseDir, "assortment", "build", "Dockerfile")),
		filepath.ToSlash(filepath.Join(generateComposeBaseDir, "assortment", "context", "Dockerfile")),
		filepath.ToSlash(filepath.Join(generateComposeBaseDir, "assortment", "dockerfile", "Dockerfile")),
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

// TestGenerateComposefileArgsDockerfileOverride ensures that build args in docker-compose
// files override args defined in Dockerfiles.
func TestGenerateComposefileArgsDockerfileOverride(t *testing.T) {
	t.Parallel()
	composefile := filepath.Join(generateComposeBaseDir, "args", "override", "docker-compose.yml")
	dockerfile := filepath.ToSlash(filepath.Join(generateComposeBaseDir, "args", "override", "Dockerfile"))
	flags := []string{fmt.Sprintf("--compose-files=%s", composefile)}
	results := map[string][]generate.ComposefileImage{filepath.ToSlash(composefile): []generate.ComposefileImage{
		{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "svc", Dockerfile: dockerfile},
	}}
	testGenerate(t, flags, results)
}

// TestGenerateComposefileArgsDockerfileEmpty ensures that build args in docker-compose
// files override empty args in Dockerfiles.
func TestGenerateComposefileArgsDockerfileEmpty(t *testing.T) {
	t.Parallel()
	composefile := filepath.Join(generateComposeBaseDir, "args", "empty", "docker-compose.yml")
	dockerfile := filepath.ToSlash(filepath.Join(generateComposeBaseDir, "args", "empty", "Dockerfile"))
	flags := []string{fmt.Sprintf("--compose-files=%s", composefile)}
	results := map[string][]generate.ComposefileImage{filepath.ToSlash(composefile): []generate.ComposefileImage{
		{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "svc", Dockerfile: dockerfile},
	}}
	testGenerate(t, flags, results)
}

// TestGenerateComposefileArgsDockerfileNoArg ensures that args defined in Dockerfiles
// but not in docker-compose files behave as though no docker-compose files exist.
func TestGenerateComposefileArgsDockerfileNoArg(t *testing.T) {
	t.Parallel()
	composefile := filepath.Join(generateComposeBaseDir, "args", "noarg", "docker-compose.yml")
	dockerfile := filepath.ToSlash(filepath.Join(generateComposeBaseDir, "args", "noarg", "Dockerfile"))
	flags := []string{fmt.Sprintf("--compose-files=%s", composefile)}
	results := map[string][]generate.ComposefileImage{filepath.ToSlash(composefile): []generate.ComposefileImage{
		{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "svc", Dockerfile: dockerfile},
	}}
	testGenerate(t, flags, results)
}

// TestGenerateComposefileAndDockerfileDuplicates ensures that Lockfiles do not
// include the same file twice.
func TestGenerateComposefileAndDockerfileDuplicates(t *testing.T) {
	t.Parallel()
	composefile := filepath.Join(generateBothBaseDir, "both", "docker-compose.yml")
	dockerfile := filepath.ToSlash(filepath.Join(generateBothBaseDir, "both", "both", "Dockerfile"))
	flags := []string{fmt.Sprintf("--compose-files=%s,%s", composefile, composefile), fmt.Sprintf("--dockerfiles=%s,%s", dockerfile, dockerfile)}
	results := map[string][]generate.ComposefileImage{filepath.ToSlash(composefile): []generate.ComposefileImage{
		{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "both-svc", Dockerfile: dockerfile},
		{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "image-svc", Dockerfile: ""},
	}}
	testGenerate(t, flags, results)
}

// TestGenerateDockerfileArgsBuildStage ensures that previously defined build stages
// are not included in Lockfiles. For instance:
// # Dockerfile
// FROM busybox AS busy
// FROM busy AS anotherbusy
// should only parse the first 'busybox'.
func TestGenerateDockerfileArgsBuildStage(t *testing.T) {
	t.Parallel()
	dockerfile := filepath.Join(generateDockerBaseDir, "args", "buildstage", "Dockerfile")
	flags := []string{fmt.Sprintf("--dockerfiles=%s", dockerfile)}
	results := map[string][]generate.DockerfileImage{filepath.ToSlash(dockerfile): []generate.DockerfileImage{
		{Image: generate.Image{Name: "busybox", Tag: "latest"}},
		{Image: generate.Image{Name: "busybox", Tag: "latest"}},
	}}
	testGenerate(t, flags, results)
}

// TestGenerateDockerfileArgsLocalArg ensures that args defined before from statements
// (aka global args) should not be overriden by args defined after from statements
// (aka local args).
func TestGenerateDockerfileArgsLocalArg(t *testing.T) {
	t.Parallel()
	dockerfile := filepath.Join(generateDockerBaseDir, "args", "localarg", "Dockerfile")
	flags := []string{fmt.Sprintf("--dockerfiles=%s", dockerfile)}
	results := map[string][]generate.DockerfileImage{filepath.ToSlash(dockerfile): []generate.DockerfileImage{
		{Image: generate.Image{Name: "busybox", Tag: "latest"}},
		{Image: generate.Image{Name: "busybox", Tag: "latest"}},
	}}
	testGenerate(t, flags, results)
}

// TestGenerateDockerfileMultipleDockerfiles ensures that Lockfiles from multiple
// Dockerfiles are correct.
func TestGenerateDockerfileMultipleDockerfiles(t *testing.T) {
	t.Parallel()
	dockerfiles := []string{filepath.Join(generateDockerBaseDir, "multiple", "DockerfileOne"), filepath.Join(generateDockerBaseDir, "multiple", "DockerfileTwo")}
	flags := []string{fmt.Sprintf("--dockerfiles=%s,%s", dockerfiles[0], dockerfiles[1])}
	results := make(map[string][]generate.DockerfileImage)
	for _, dockerfile := range dockerfiles {
		results[filepath.ToSlash(dockerfile)] = append(results[filepath.ToSlash(dockerfile)], generate.DockerfileImage{Image: generate.Image{Name: "busybox", Tag: "latest"}})
	}
	testGenerate(t, flags, results)
}

// TestGenerateDockerfileRecursive ensures Lockfiles from multiple Dockerfiles
// in subdirectories are correct.
func TestGenerateDockerfileRecursive(t *testing.T) {
	t.Parallel()
	recursiveBaseDir := filepath.Join(generateDockerBaseDir, "recursive")
	flags := []string{fmt.Sprintf("--base-dir=%s", recursiveBaseDir), "--dockerfile-recursive"}
	results := make(map[string][]generate.DockerfileImage)
	dockerfiles := []string{filepath.Join(generateDockerBaseDir, "recursive", "Dockerfile"), filepath.Join(generateDockerBaseDir, "recursive", "recursive", "Dockerfile")}
	for _, dockerfile := range dockerfiles {
		results[filepath.ToSlash(dockerfile)] = append(results[filepath.ToSlash(dockerfile)], generate.DockerfileImage{Image: generate.Image{Name: "busybox", Tag: "latest"}})
	}
	testGenerate(t, flags, results)
}

// TestGenerateDockerfileNoFileSpecified ensures Lockfiles include a Dockerfile
// in the base directory, if no other files are specified.
func TestGenerateDockerfileNoFileSpecified(t *testing.T) {
	t.Parallel()
	baseDir := filepath.Join(generateDockerBaseDir, "nofile")
	flags := []string{fmt.Sprintf("--base-dir=%s", baseDir)}
	dockerfile := filepath.Join(baseDir, "Dockerfile")
	results := map[string][]generate.DockerfileImage{filepath.ToSlash(dockerfile): []generate.DockerfileImage{
		{Image: generate.Image{Name: "busybox", Tag: "latest"}},
	}}
	testGenerate(t, flags, results)
}

// TestGenerateDockerfileGlobs ensures Lockfiles include Dockerfiles files found
// via glob syntax.
func TestGenerateDockerfileGlobs(t *testing.T) {
	t.Parallel()
	globs := strings.Join([]string{filepath.Join(generateDockerBaseDir, "globs", "**", "Dockerfile"), filepath.Join(generateDockerBaseDir, "globs", "Dockerfile")}, ",")
	flags := []string{fmt.Sprintf("--dockerfile-globs=%s", globs)}
	dockerfiles := []string{filepath.Join(generateDockerBaseDir, "globs", "globs", "Dockerfile"), filepath.Join(generateDockerBaseDir, "globs", "Dockerfile")}
	results := make(map[string][]generate.DockerfileImage)
	for _, dockerfile := range dockerfiles {
		results[filepath.ToSlash(dockerfile)] = append(results[filepath.ToSlash(dockerfile)], generate.DockerfileImage{Image: generate.Image{Name: "busybox", Tag: "latest"}})
	}
	testGenerate(t, flags, results)
}

// TestGenerateDockerfilePrivate ensures Lockfiles work with private images
// hosted on Dockerhub.
func TestGenerateDockerfilePrivate(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI_SERVER") != "TRUE" {
		t.Skip("Only runs on CI server.")
	}
	dockerfile := filepath.Join(generateDockerBaseDir, "private", "Dockerfile")
	flags := []string{fmt.Sprintf("--dockerfiles=%s", dockerfile)}
	results := map[string][]generate.DockerfileImage{filepath.ToSlash(dockerfile): []generate.DockerfileImage{
		{Image: generate.Image{Name: "dockerlocktestaccount/busybox", Tag: "latest"}},
	}}
	testGenerate(t, flags, results)
}

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
