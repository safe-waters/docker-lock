package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRewriteDockerfileArgsLocalArg replaces the ARG referenced in
// the FROM instruction with the image.

var rewriteDockerBaseDir = filepath.Join("testdata", "rewrite", "docker")

func TestRewriteDockerfileArgsLocalArg(t *testing.T) {

}

func TestRewriteDockerfileArgsBuildStage(t *testing.T) {
	baseDir := filepath.Join(rewriteDockerBaseDir, "args", "buildstage")
	outPath := filepath.Join(baseDir, "docker-lock.json")
	wantPaths := []string{filepath.Join(baseDir, "Dockerfile-want")}
	gotPaths := []string{filepath.Join(baseDir, "Dockerfile-got")}
	testRewrite(t, outPath, wantPaths, gotPaths)
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

func testRewrite(t *testing.T, outPath string, wantPaths []string, gotPaths []string) {
	rewriteCmd := NewRewriteCmd()
	rewriteArgs := append([]string{"lock", "rewrite", fmt.Sprintf("--outpath=%s", outPath), "--suffix=got"})
	rewriteCmd.SetArgs(rewriteArgs)
	if err := rewriteCmd.Execute(); err != nil {
		t.Error(err)
	}
	for _, gotPath := range gotPaths {
		defer os.Remove(gotPath)
	}
	for i := range gotPaths {
		gotByt, err := ioutil.ReadFile(gotPaths[i])
		if err != nil {
			t.Error(err)
		}
		wantByt, err := ioutil.ReadFile(wantPaths[i])
		if err != nil {
			t.Error(err)
		}
		if bytes.Compare(gotByt, wantByt) != 0 {
			t.Errorf("Files %s and %s differ.", gotPaths[i], wantPaths[i])
		}
		gotLines := strings.Split(string(gotByt), "\n")
		wantLines := strings.Split(string(wantByt), "\n")
		if len(gotLines) != len(wantLines) {
			t.Errorf("%s and %s have a different number of lines.", gotPaths[i], wantPaths[i])
		}
		for j := range gotLines {
			if gotLines[j] != wantLines[j] {
				t.Errorf("Got %s, want %s.", gotLines[j], wantLines[j])
			}
		}
	}
}
