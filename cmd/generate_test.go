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

type generateTestObject struct {
	filePath   string
	wantImages interface{}
	testFn     func(*testing.T, generateTestObject, generate.Lockfile)
}

// TestGenerateComposefileImage ensures Lockfiles from docker-compose files with
// the image key are correct.
func TestGenerateComposefileImage(t *testing.T) {
	t.Parallel()
	composefile := filepath.Join(generateComposeBaseDir, "image", "docker-compose.yml")
	flags := []string{fmt.Sprintf("--compose-files=%s", composefile)}
	tOs := []generateTestObject{
		{
			filePath: filepath.ToSlash(composefile),
			wantImages: []generate.ComposefileImage{
				{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "svc", Dockerfile: ""},
			},
			testFn: checkGenerateComposefile,
		},
	}
	testGenerate(t, flags, tOs)
}

// TestGenerateComposefileBuild ensures Lockfiles from docker-compose files with
// the build key are correct.
func TestGenerateComposefileBuild(t *testing.T) {
	t.Parallel()
	composefile := filepath.Join(generateComposeBaseDir, "build", "docker-compose.yml")
	dockerfile := filepath.ToSlash(filepath.Join(generateComposeBaseDir, "build", "build", "Dockerfile"))
	flags := []string{fmt.Sprintf("--compose-files=%s", composefile)}
	tOs := []generateTestObject{
		{
			filePath: filepath.ToSlash(composefile),
			wantImages: []generate.ComposefileImage{
				{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "svc", Dockerfile: dockerfile},
			},
			testFn: checkGenerateComposefile,
		},
	}
	testGenerate(t, flags, tOs)
}

// TestGenerateComposefileDockerfile ensures Lockfiles from docker-compose files with
// the dockerfile key are correct.
func TestGenerateComposefileDockerfile(t *testing.T) {
	t.Parallel()
	composefile := filepath.Join(generateComposeBaseDir, "dockerfile", "docker-compose.yml")
	dockerfile := filepath.ToSlash(filepath.Join(generateComposeBaseDir, "dockerfile", "dockerfile", "Dockerfile"))
	flags := []string{fmt.Sprintf("--compose-files=%s", composefile)}
	tOs := []generateTestObject{
		{
			filePath: filepath.ToSlash(composefile),
			wantImages: []generate.ComposefileImage{
				{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "svc", Dockerfile: dockerfile},
			},
			testFn: checkGenerateComposefile,
		},
	}
	testGenerate(t, flags, tOs)
}

// TestGenerateComposefileContext ensures Lockfiles from docker-compose files with
// the context key are correct.
func TestGenerateComposefileContext(t *testing.T) {
	t.Parallel()
	composefile := filepath.Join(generateComposeBaseDir, "context", "docker-compose.yml")
	dockerfile := filepath.ToSlash(filepath.Join(generateComposeBaseDir, "context", "context", "Dockerfile"))
	flags := []string{fmt.Sprintf("--compose-files=%s", composefile)}
	tOs := []generateTestObject{
		{
			filePath: filepath.ToSlash(composefile),
			wantImages: []generate.ComposefileImage{
				{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "svc", Dockerfile: dockerfile},
			},
			testFn: checkGenerateComposefile,
		},
	}
	testGenerate(t, flags, tOs)
}

// TestGenerateComposefileEnv ensures Lockfiles from docker-compose files with
// environment variables replaced by values in a .env file are correct.
func TestGenerateComposefileEnv(t *testing.T) {
	composefile := filepath.Join(generateComposeBaseDir, "env", "docker-compose.yml")
	dockerfile := filepath.ToSlash(filepath.Join(generateComposeBaseDir, "env", "env", "Dockerfile"))
	envFile := filepath.Join(generateComposeBaseDir, "env", ".env")
	flags := []string{fmt.Sprintf("--compose-files=%s", composefile), fmt.Sprintf("--env-file=%s", envFile)}
	tOs := []generateTestObject{
		{
			filePath: filepath.ToSlash(composefile),
			wantImages: []generate.ComposefileImage{
				{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "svc", Dockerfile: dockerfile},
			},
			testFn: checkGenerateComposefile,
		},
	}
	testGenerate(t, flags, tOs)
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
	tOs := []generateTestObject{
		{
			filePath: filepath.ToSlash(composefileOne),
			wantImages: []generate.ComposefileImage{
				{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "build-svc", Dockerfile: dockerfilesOne[0]},
				{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "image-svc", Dockerfile: ""},
			},
			testFn: checkGenerateComposefile,
		},
		{
			filePath: filepath.ToSlash(composefileTwo),
			wantImages: []generate.ComposefileImage{
				{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "context-svc", Dockerfile: dockerfilesTwo[0]},
				{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "dockerfile-svc", Dockerfile: dockerfilesTwo[1]},
			},
			testFn: checkGenerateComposefile,
		},
	}
	testGenerate(t, flags, tOs)
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
	tOs := []generateTestObject{
		{
			filePath: filepath.ToSlash(composefileTopLevel),
			wantImages: []generate.ComposefileImage{
				{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "svc", Dockerfile: ""},
			},
			testFn: checkGenerateComposefile,
		},
		{
			filePath: filepath.ToSlash(composefileRecursiveLevel),
			wantImages: []generate.ComposefileImage{
				{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "svc", Dockerfile: dockerfileRecursiveLevel},
			},
			testFn: checkGenerateComposefile,
		},
	}
	testGenerate(t, flags, tOs)
}

// TestGenerateComposefileNoFileSpecified ensures Lockfiles include docker-compose.yml
// and docker-compose.yaml files in the base directory, if no other files are specified.
func TestGenerateComposefileNoFileSpecified(t *testing.T) {
	t.Parallel()
	baseDir := filepath.Join(generateComposeBaseDir, "nofile")
	flags := []string{fmt.Sprintf("--base-dir=%s", baseDir)}
	composefiles := []string{filepath.Join(baseDir, "docker-compose.yml"), filepath.Join(baseDir, "docker-compose.yaml")}
	tOs := []generateTestObject{
		{
			filePath: filepath.ToSlash(composefiles[0]),
			wantImages: []generate.ComposefileImage{
				{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "svc", Dockerfile: ""},
			},
			testFn: checkGenerateComposefile,
		},
		{
			filePath: filepath.ToSlash(composefiles[1]),
			wantImages: []generate.ComposefileImage{
				{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "svc", Dockerfile: ""},
			},
			testFn: checkGenerateComposefile,
		},
	}
	testGenerate(t, flags, tOs)
}

// TestGenerateComposefileGlobs ensures Lockfiles include docker-compose files found
// via glob syntax.
func TestGenerateComposefileGlobs(t *testing.T) {
	t.Parallel()
	globs := strings.Join([]string{filepath.Join(generateComposeBaseDir, "globs", "**", "docker-compose.yml"), filepath.Join(generateComposeBaseDir, "globs", "docker-compose.yml")}, ",")
	flags := []string{fmt.Sprintf("--compose-file-globs=%s", globs)}
	composefiles := []string{filepath.Join(generateComposeBaseDir, "globs", "image", "docker-compose.yml"), filepath.Join(generateComposeBaseDir, "globs", "docker-compose.yml")}
	var tOs []generateTestObject
	for _, composefile := range composefiles {
		tO := generateTestObject{
			filePath: filepath.ToSlash(composefile),
			wantImages: []generate.ComposefileImage{
				{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "svc", Dockerfile: ""},
			},
			testFn: checkGenerateComposefile,
		}
		tOs = append(tOs, tO)
	}
	testGenerate(t, flags, tOs)
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
	tOs := []generateTestObject{
		{
			filePath: filepath.ToSlash(composefile),
			wantImages: []generate.ComposefileImage{
				{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "build-svc", Dockerfile: dockerfiles[0]},
				{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "context-svc", Dockerfile: dockerfiles[1]},
				{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "dockerfile-svc", Dockerfile: dockerfiles[2]},
				{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "image-svc", Dockerfile: ""},
			},
			testFn: checkGenerateComposefile,
		},
	}
	testGenerate(t, flags, tOs)
}

// TestGenerateComposefileArgsDockerfileOverride ensures that build args in docker-compose
// files override args defined in Dockerfiles.
func TestGenerateComposefileArgsDockerfileOverride(t *testing.T) {
	t.Parallel()
	composefile := filepath.Join(generateComposeBaseDir, "args", "override", "docker-compose.yml")
	dockerfile := filepath.ToSlash(filepath.Join(generateComposeBaseDir, "args", "override", "Dockerfile"))
	flags := []string{fmt.Sprintf("--compose-files=%s", composefile)}
	tOs := []generateTestObject{
		{
			filePath: filepath.ToSlash(composefile),
			wantImages: []generate.ComposefileImage{
				{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "svc", Dockerfile: dockerfile},
			},
			testFn: checkGenerateComposefile,
		},
	}
	testGenerate(t, flags, tOs)
}

// TestGenerateComposefileArgsDockerfileEmpty ensures that build args in docker-compose
// files override empty args in Dockerfiles.
func TestGenerateComposefileArgsDockerfileEmpty(t *testing.T) {
	t.Parallel()
	composefile := filepath.Join(generateComposeBaseDir, "args", "empty", "docker-compose.yml")
	dockerfile := filepath.ToSlash(filepath.Join(generateComposeBaseDir, "args", "empty", "Dockerfile"))
	flags := []string{fmt.Sprintf("--compose-files=%s", composefile)}
	tOs := []generateTestObject{
		{
			filePath: filepath.ToSlash(composefile),
			wantImages: []generate.ComposefileImage{
				{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "svc", Dockerfile: dockerfile},
			},
			testFn: checkGenerateComposefile,
		},
	}
	testGenerate(t, flags, tOs)
}

// TestGenerateComposefileArgsDockerfileNoArg ensures that args defined in Dockerfiles
// but not in docker-compose files behave as though no docker-compose files exist.
func TestGenerateComposefileArgsDockerfileNoArg(t *testing.T) {
	t.Parallel()
	composefile := filepath.Join(generateComposeBaseDir, "args", "noarg", "docker-compose.yml")
	dockerfile := filepath.ToSlash(filepath.Join(generateComposeBaseDir, "args", "noarg", "Dockerfile"))
	flags := []string{fmt.Sprintf("--compose-files=%s", composefile)}
	tOs := []generateTestObject{
		{
			filePath: filepath.ToSlash(composefile),
			wantImages: []generate.ComposefileImage{
				{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "svc", Dockerfile: dockerfile},
			},
			testFn: checkGenerateComposefile,
		},
	}
	testGenerate(t, flags, tOs)
}

// TestGenerateComposefileAndDockerfileDuplicates ensures that Lockfiles do not
// include the same file twice.
func TestGenerateComposefileAndDockerfileDuplicates(t *testing.T) {
	t.Parallel()
	composefile := filepath.Join(generateBothBaseDir, "both", "docker-compose.yml")
	dockerfile := filepath.ToSlash(filepath.Join(generateBothBaseDir, "both", "both", "Dockerfile"))
	flags := []string{fmt.Sprintf("--compose-files=%s,%s", composefile, composefile), fmt.Sprintf("--dockerfiles=%s,%s", dockerfile, dockerfile)}
	tOs := []generateTestObject{
		{
			filePath: filepath.ToSlash(dockerfile),
			wantImages: []generate.DockerfileImage{
				{Image: generate.Image{Name: "busybox", Tag: "latest"}},
			},
			testFn: checkGenerateDockerfile,
		},
		{
			filePath: filepath.ToSlash(composefile),
			wantImages: []generate.ComposefileImage{
				{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "both-svc", Dockerfile: dockerfile},
				{Image: generate.Image{Name: "busybox", Tag: "latest"}, ServiceName: "image-svc", Dockerfile: ""},
			},
			testFn: checkGenerateComposefile,
		},
	}
	testGenerate(t, flags, tOs)
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
	tOs := []generateTestObject{
		{
			filePath: filepath.ToSlash(dockerfile),
			wantImages: []generate.DockerfileImage{
				{Image: generate.Image{Name: "busybox", Tag: "latest"}},
				{Image: generate.Image{Name: "busybox", Tag: "latest"}},
			},
			testFn: checkGenerateDockerfile,
		},
	}
	testGenerate(t, flags, tOs)
}

// TestGenerateDockerfileArgsLocalArg ensures that args defined before from statements
// (aka global args) should not be overriden by args defined after from statements
// (aka local args).
func TestGenerateDockerfileArgsLocalArg(t *testing.T) {
	t.Parallel()
	dockerfile := filepath.Join(generateDockerBaseDir, "args", "localarg", "Dockerfile")
	flags := []string{fmt.Sprintf("--dockerfiles=%s", dockerfile)}
	tOs := []generateTestObject{
		{
			filePath: filepath.ToSlash(dockerfile),
			wantImages: []generate.DockerfileImage{
				{Image: generate.Image{Name: "busybox", Tag: "latest"}},
				{Image: generate.Image{Name: "busybox", Tag: "latest"}},
			},
			testFn: checkGenerateDockerfile,
		},
	}
	testGenerate(t, flags, tOs)
}

// TestGenerateDockerfileMultipleDockerfiles ensures that Lockfiles from multiple
// Dockerfiles are correct.
func TestGenerateDockerfileMultipleDockerfiles(t *testing.T) {
	t.Parallel()
	dockerfiles := []string{filepath.Join(generateDockerBaseDir, "multiple", "DockerfileOne"), filepath.Join(generateDockerBaseDir, "multiple", "DockerfileTwo")}
	flags := []string{fmt.Sprintf("--dockerfiles=%s,%s", dockerfiles[0], dockerfiles[1])}
	var tOs []generateTestObject
	for _, dockerfile := range dockerfiles {
		tO := generateTestObject{
			filePath: filepath.ToSlash(dockerfile),
			wantImages: []generate.DockerfileImage{
				{Image: generate.Image{Name: "busybox", Tag: "latest"}},
			},
			testFn: checkGenerateDockerfile,
		}
		tOs = append(tOs, tO)
	}
	testGenerate(t, flags, tOs)
}

// TestGenerateDockerfileRecursive ensures Lockfiles from multiple Dockerfiles
// in subdirectories are correct.
func TestGenerateDockerfileRecursive(t *testing.T) {
	t.Parallel()
	recursiveBaseDir := filepath.Join(generateDockerBaseDir, "recursive")
	flags := []string{fmt.Sprintf("--base-dir=%s", recursiveBaseDir), "--dockerfile-recursive"}
	dockerfiles := []string{filepath.Join(generateDockerBaseDir, "recursive", "Dockerfile"), filepath.Join(generateDockerBaseDir, "recursive", "recursive", "Dockerfile")}
	var tOs []generateTestObject
	for _, dockerfile := range dockerfiles {
		tO := generateTestObject{
			filePath: filepath.ToSlash(dockerfile),
			wantImages: []generate.DockerfileImage{
				{Image: generate.Image{Name: "busybox", Tag: "latest"}},
			},
			testFn: checkGenerateDockerfile,
		}
		tOs = append(tOs, tO)
	}

	testGenerate(t, flags, tOs)
}

// TestGenerateDockerfileNoFileSpecified ensures Lockfiles include a Dockerfile
// in the base directory, if no other files are specified.
func TestGenerateDockerfileNoFileSpecified(t *testing.T) {
	t.Parallel()
	baseDir := filepath.Join(generateDockerBaseDir, "nofile")
	flags := []string{fmt.Sprintf("--base-dir=%s", baseDir)}
	dockerfile := filepath.Join(baseDir, "Dockerfile")
	tOs := []generateTestObject{
		{
			filePath: filepath.ToSlash(dockerfile),
			wantImages: []generate.DockerfileImage{
				{Image: generate.Image{Name: "busybox", Tag: "latest"}},
			},
			testFn: checkGenerateDockerfile,
		},
	}
	testGenerate(t, flags, tOs)
}

// TestGenerateDockerfileGlobs ensures Lockfiles include Dockerfiles files found
// via glob syntax.
func TestGenerateDockerfileGlobs(t *testing.T) {
	t.Parallel()
	globs := strings.Join([]string{filepath.Join(generateDockerBaseDir, "globs", "**", "Dockerfile"), filepath.Join(generateDockerBaseDir, "globs", "Dockerfile")}, ",")
	flags := []string{fmt.Sprintf("--dockerfile-globs=%s", globs)}
	dockerfiles := []string{filepath.Join(generateDockerBaseDir, "globs", "globs", "Dockerfile"), filepath.Join(generateDockerBaseDir, "globs", "Dockerfile")}
	var tOs []generateTestObject
	for _, dockerfile := range dockerfiles {
		tO := generateTestObject{
			filePath: filepath.ToSlash(dockerfile),
			wantImages: []generate.DockerfileImage{
				{Image: generate.Image{Name: "busybox", Tag: "latest"}},
			},
			testFn: checkGenerateDockerfile,
		}
		tOs = append(tOs, tO)
	}
	testGenerate(t, flags, tOs)
}

// TestGenerateDockerfileEnvBuildArgs ensures environment variables are used as
// build args.
func TestGenerateDockerfileEnvBuildArgs(t *testing.T) {
	dockerfile := filepath.Join(generateDockerBaseDir, "args", "buildargs", "Dockerfile")
	envFile := filepath.Join(generateDockerBaseDir, "args", "buildargs", ".env")
	flags := []string{fmt.Sprintf("--dockerfiles=%s", dockerfile), fmt.Sprintf("--env-file=%s", envFile), fmt.Sprintf("--dockerfile-env-build-args")}
	tOs := []generateTestObject{
		{
			filePath: filepath.ToSlash(dockerfile),
			wantImages: []generate.DockerfileImage{
				{Image: generate.Image{Name: "busybox", Tag: "latest"}},
			},
			testFn: checkGenerateDockerfile,
		},
	}
	testGenerate(t, flags, tOs)
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
	tOs := []generateTestObject{
		{
			filePath: filepath.ToSlash(dockerfile),
			wantImages: []generate.DockerfileImage{
				{Image: generate.Image{Name: "dockerlocktestaccount/busybox", Tag: "latest"}},
			},
			testFn: checkGenerateDockerfile,
		},
	}
	testGenerate(t, flags, tOs)
}

func testGenerate(t *testing.T, flags []string, tOs []generateTestObject) {
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
	numFiles := len(lFile.ComposefileImages) + len(lFile.DockerfileImages)
	if numFiles != len(tOs) {
		t.Errorf("Got '%d' files in the Lockfile. Want '%d'.", numFiles, len(tOs))
	}
	for _, tO := range tOs {
		tO.testFn(t, tO, lFile)
	}
}

func checkGenerateComposefile(t *testing.T, tO generateTestObject, lFile generate.Lockfile) {
	wantImages := tO.wantImages.([]generate.ComposefileImage)
	fImages, ok := lFile.ComposefileImages[tO.filePath]
	if !ok {
		t.Errorf("Want '%s' filepath, but did not find it.", tO.filePath)
	}
	if len(fImages) != len(wantImages) {
		t.Errorf("Got '%d' images for '%s'. Want '%d'.", len(fImages), tO.filePath, len(wantImages))
	}
	for i, fImage := range fImages {
		if wantImages[i].Image.Name != fImage.Image.Name ||
			wantImages[i].Image.Tag != fImage.Image.Tag {
			t.Errorf("Got '%s:%s'. Want '%s:%s'.",
				fImage.Image.Name,
				fImage.Image.Tag,
				wantImages[i].Image.Name,
				wantImages[i].Image.Tag)
		}
		if fImage.Image.Digest == "" {
			t.Errorf("Want digest. Got image with empty digest %+v.", fImage)
		}
		if fImage.ServiceName != wantImages[i].ServiceName {
			t.Errorf("Got '%s' service. Want '%s'.",
				fImage.ServiceName,
				wantImages[i].ServiceName)
		}
		if fImage.Dockerfile != wantImages[i].Dockerfile {
			t.Errorf("Got '%s' dockerfile. Want '%s'.",
				filepath.FromSlash(fImage.Dockerfile),
				wantImages[i].Dockerfile)
		}
	}
}

func checkGenerateDockerfile(t *testing.T, tO generateTestObject, lFile generate.Lockfile) {
	wantImages := tO.wantImages.([]generate.DockerfileImage)
	fImages, ok := lFile.DockerfileImages[tO.filePath]
	if !ok {
		t.Errorf("Got '%s' filepath, but did not find it.", tO.filePath)
	}
	if len(fImages) != len(wantImages) {
		t.Errorf("Got '%d' images for '%s'. Want '%d'.", len(fImages), tO.filePath, len(wantImages))
	}
	for i, fImage := range fImages {
		if wantImages[i].Image.Name != fImage.Image.Name ||
			wantImages[i].Image.Tag != fImage.Image.Tag {
			t.Errorf("Got '%s:%s'. Want '%s:%s'.",
				fImage.Image.Name,
				fImage.Image.Tag,
				wantImages[i].Image.Name,
				wantImages[i].Image.Tag)
		}
		if fImage.Image.Digest == "" {
			t.Errorf("Want digest. Got image with empty digest %+v.", fImage)
		}
	}
}
