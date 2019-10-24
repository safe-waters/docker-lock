package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/michaelperel/docker-lock/generate"
)

// TestRewriteDockerfileArgsLocalArg replaces the ARG referenced in
// the FROM instruction with the image.
func TestRewriteDockerfileArgsLocalArg(t *testing.T) {

}

func TestRewriteDockerfileArgsBuildStage(t *testing.T) {

}

// TestComposefileEnv replaces the environment variable
// referenced in the image line with the image.
func TestRewriteComposefileEnv(t *testing.T) {

}

// TestComposefileImage replaces the image line with the image.
func TestRewriteComposefileImage(t *testing.T) {

}

// TestComposefileDockerfiles ensures that Dockerfiles
// referenced in docker-compose files are rewritten.
func TestRewriteComposefileDockerfiles(t *testing.T) {

}

// TestIncorrectNumberOfImages ensures that when there is
// a mismatch of the number of images in the Dockerfile
// compared with the Lockfile, no rewrite occurs.
func TestRewriteIncorrectNumberOfImages(t *testing.T) {

}

// TestBuildStage makes sure that only the buildstages
// that reference new images are rewritten.
func TestRewriteBuildStage(t *testing.T) {

}

func testRewrite(t *testing.T, flags []string, results interface{}) {
	tmpFile, err := ioutil.TempFile("", "test-rewrite-docker-lock-*")
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
