package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/michaelperel/docker-lock/cmd/internal/compose"
	"gopkg.in/yaml.v2"
)

var rewriteDockerBaseDir = filepath.Join("testdata", "rewrite", "docker")
var rewriteComposeBaseDir = filepath.Join("testdata", "rewrite", "compose")

type rewriteTestObject struct {
	wantPaths []string
	gotPaths  []string
	testFn    func(*testing.T, []string, []string)
}

// TestRewriteDockerfileArgsLocalArg replaces the ARG referenced in
// the FROM instruction with the image.
func TestRewriteDockerfileArgsLocalArg(t *testing.T) {
	baseDir := filepath.Join(rewriteDockerBaseDir, "args", "localarg")
	outPath := filepath.Join(baseDir, "docker-lock.json")
	tOs := []rewriteTestObject{
		{
			wantPaths: []string{filepath.Join(baseDir, "Dockerfile-want")},
			gotPaths:  []string{filepath.Join(baseDir, "Dockerfile-got")},
			testFn:    checkRewriteDockerfile,
		},
	}
	testRewrite(t, outPath, tOs, false)
}

// TestRewriteDockerfileArgsBuildStage replaces newly defined images
// in the case of multi-stage builds. For instance:
// # Dockerfile
// FROM busybox AS busy
// FROM busy AS anotherbusy
// should only replace the first busybox.
func TestRewriteDockerfileArgsBuildStage(t *testing.T) {
	baseDir := filepath.Join(rewriteDockerBaseDir, "args", "buildstage")
	outPath := filepath.Join(baseDir, "docker-lock.json")
	tOs := []rewriteTestObject{
		{
			wantPaths: []string{filepath.Join(baseDir, "Dockerfile-want")},
			gotPaths:  []string{filepath.Join(baseDir, "Dockerfile-got")},
			testFn:    checkRewriteDockerfile,
		},
	}
	testRewrite(t, outPath, tOs, false)
}

// TestRewriteMoreDockerfileImages ensures that when there are
// more images in a Dockerfile than in a Lockfile, an error occurs.
func TestRewriteMoreDockerfileImages(t *testing.T) {
	baseDir := filepath.Join(rewriteDockerBaseDir, "fail", "moreImagesDockerfile")
	outPath := filepath.Join(baseDir, "docker-lock.json")
	tOs := []rewriteTestObject{
		{
			wantPaths: []string{filepath.Join(baseDir, "Dockerfile-want")},
			gotPaths:  []string{filepath.Join(baseDir, "Dockerfile-got")},
			testFn:    checkRewriteDockerfile,
		},
	}
	testRewrite(t, outPath, tOs, true)
}

// TestRewriteMoreLockfileImages ensures that when there are
// more images in a Lockfile than in a Dockerfile, an error occurs.
func TestRewriteMoreLockfileImages(t *testing.T) {
	baseDir := filepath.Join(rewriteDockerBaseDir, "fail", "moreImagesLockfile")
	outPath := filepath.Join(baseDir, "docker-lock.json")
	tOs := []rewriteTestObject{
		{
			wantPaths: []string{filepath.Join(baseDir, "Dockerfile-want")},
			gotPaths:  []string{filepath.Join(baseDir, "Dockerfile-got")},
			testFn:    checkRewriteDockerfile,
		},
	}
	testRewrite(t, outPath, tOs, true)
}

// TestRewriteComposefileImage replaces the image line with the image.
func TestRewriteComposefileImage(t *testing.T) {
	baseDir := filepath.Join(rewriteComposeBaseDir, "image")
	outPath := filepath.Join(baseDir, "docker-lock.json")
	tOs := []rewriteTestObject{
		{
			wantPaths: []string{filepath.Join(baseDir, "docker-compose-want.yml")},
			gotPaths:  []string{filepath.Join(baseDir, "docker-compose-got.yml")},
			testFn:    checkRewriteComposefile,
		},
	}
	testRewrite(t, outPath, tOs, false)
}

// TestRewriteComposefileEnv replaces the environment variable
// referenced in the image line with the image.
func TestRewriteComposefileEnv(t *testing.T) {
	baseDir := filepath.Join(rewriteComposeBaseDir, "env")
	outPath := filepath.Join(baseDir, "docker-lock.json")
	tOs := []rewriteTestObject{
		{
			wantPaths: []string{filepath.Join(baseDir, "docker-compose-want.yml")},
			gotPaths:  []string{filepath.Join(baseDir, "docker-compose-got.yml")},
			testFn:    checkRewriteComposefile,
		},
	}
	testRewrite(t, outPath, tOs, false)
}

// TestRewriteDockerfilesReferencedByComposefiles ensures that Dockerfiles
// referenced in docker-compose files are rewritten.
func TestRewriteDockerfilesReferencedByComposefiles(t *testing.T) {
	baseDir := filepath.Join(rewriteComposeBaseDir, "dockerfile")
	outPath := filepath.Join(baseDir, "docker-lock.json")
	tOs := []rewriteTestObject{
		{
			wantPaths: []string{filepath.Join(baseDir, "dockerfile", "Dockerfile-want")},
			gotPaths:  []string{filepath.Join(baseDir, "dockerfile", "Dockerfile-got")},
			testFn:    checkRewriteDockerfile,
		},
	}
	testRewrite(t, outPath, tOs, false)
}

// TestRewriteAssortment tests rewrite for a collection of arbitrary
// docker-compose files and Dockerfiles.
func TestRewriteAssortment(t *testing.T) {
	baseDir := filepath.Join(rewriteComposeBaseDir, "assortment")
	outPath := filepath.Join(baseDir, "docker-lock.json")
	tOs := []rewriteTestObject{
		{
			wantPaths: []string{filepath.Join(baseDir, "build", "Dockerfile-want")},
			gotPaths:  []string{filepath.Join(baseDir, "build", "Dockerfile-got")},
			testFn:    checkRewriteDockerfile,
		},
		{
			wantPaths: []string{filepath.Join(baseDir, "context", "Dockerfile-want")},
			gotPaths:  []string{filepath.Join(baseDir, "context", "Dockerfile-got")},
			testFn:    checkRewriteDockerfile,
		},
		{
			wantPaths: []string{filepath.Join(baseDir, "dockerfile", "Dockerfile-want")},
			gotPaths:  []string{filepath.Join(baseDir, "dockerfile", "Dockerfile-got")},
			testFn:    checkRewriteDockerfile,
		},
		{
			wantPaths: []string{filepath.Join(baseDir, "docker-compose-want.yml")},
			gotPaths:  []string{filepath.Join(baseDir, "docker-compose-got.yml")},
			testFn:    checkRewriteComposefile,
		},
		{
			wantPaths: []string{filepath.Join(baseDir, "docker-compose-want.yaml")},
			gotPaths:  []string{filepath.Join(baseDir, "docker-compose-got.yaml")},
			testFn:    checkRewriteComposefile,
		},
	}
	testRewrite(t, outPath, tOs, false)
}

func testRewrite(t *testing.T, outPath string, tOs []rewriteTestObject, shouldErr bool) {
	rewriteCmd := NewRewriteCmd()
	tmpDir := filepath.Join("testdata", "rewrite", "tmp")
	rewriteArgs := append([]string{"lock", "rewrite", fmt.Sprintf("--outpath=%s", outPath), fmt.Sprintf("--tempdir=%s", tmpDir), "--suffix=got"})
	rewriteCmd.SetArgs(rewriteArgs)
	if err := rewriteCmd.Execute(); err != nil {
		if shouldErr {
			return
		}
		t.Error(err)
	}
	for _, tO := range tOs {
		for _, gotPath := range tO.gotPaths {
			defer os.Remove(gotPath)
		}
		tO.testFn(t, tO.wantPaths, tO.gotPaths)
	}
}

func checkRewriteDockerfile(t *testing.T, wantPaths []string, gotPaths []string) {
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

func checkRewriteComposefile(t *testing.T, wantPaths []string, gotPaths []string) {
	for i := range gotPaths {
		gotByt, err := ioutil.ReadFile(gotPaths[i])
		if err != nil {
			t.Error(err)
		}
		wantByt, err := ioutil.ReadFile(wantPaths[i])
		if err != nil {
			t.Error(err)
		}
		var gotComp compose.Compose
		if err := yaml.Unmarshal(gotByt, &gotComp); err != nil {
			t.Error(err)
		}
		var wantComp compose.Compose
		if err := yaml.Unmarshal(wantByt, &wantComp); err != nil {
			t.Error(err)
		}
		if len(wantComp.Services) != len(gotComp.Services) {
			t.Errorf("%s and %s have a different number of services.", gotPaths[i], wantPaths[i])
		}
		for serviceName := range gotComp.Services {
			gotImage := gotComp.Services[serviceName].ImageName
			wantImage := wantComp.Services[serviceName].ImageName
			if gotImage != wantImage {
				t.Errorf("Got %s. Want %s.", gotImage, wantImage)
			}
		}
	}
}
