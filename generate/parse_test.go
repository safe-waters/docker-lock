package generate

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestParseComposeFile(t *testing.T) {
	const (
		SIMPLE3IMAGE       = "simple3image"
		SIMPLE3BUILD       = "simple3build"
		VERBOSE4IMAGE      = "verbose4image"
		VERBOSE4CONTEXT    = "verbose4build"
		VERBOSE4DOCKERFILE = "Dockerfile-dev"
	)
	os.Setenv("SIMPLE3IMAGE", SIMPLE3IMAGE)
	os.Setenv("SIMPLE3BUILD", SIMPLE3BUILD)
	os.Setenv("VERBOSE4IMAGE", VERBOSE4IMAGE)
	os.Setenv("VERBOSE4CONTEXT", VERBOSE4CONTEXT)
	os.Setenv("VERBOSE4DOCKERFILE", VERBOSE4DOCKERFILE)

	baseDir := filepath.Join("testassets", "parse", "composefile")

	results := map[parsedImageLine]bool{
		{line: "busybox", fileName: filepath.Join(baseDir, "docker-compose.yml"), err: nil}:              false,
		{line: "busybox", fileName: filepath.Join(baseDir, "simple2build", "Dockerfile"), err: nil}:      false,
		{line: "busybox", fileName: filepath.Join(baseDir, "simple3build", "Dockerfile"), err: nil}:      false,
		{line: "busybox", fileName: filepath.Join(baseDir, "simple4build", "Dockerfile"), err: nil}:      false,
		{line: "busybox", fileName: filepath.Join(baseDir, "verbose1build", "Dockerfile"), err: nil}:     false,
		{line: "busybox", fileName: filepath.Join(baseDir, "verbose2build", "Dockerfile"), err: nil}:     false,
		{line: "busybox", fileName: filepath.Join(baseDir, "verbose3build", "Dockerfile"), err: nil}:     false,
		{line: "busybox", fileName: filepath.Join(baseDir, "verbose4build", "Dockerfile-dev"), err: nil}: false,
	}
	parsedImageLines := make(chan parsedImageLine)
	var wg sync.WaitGroup
	go func() {
		wg.Add(1)
		parseComposefile(filepath.Join(baseDir, "docker-compose.yml"), parsedImageLines, &wg)
		wg.Wait()
		close(parsedImageLines)
	}()
	var i int
	for parsedImageLine := range parsedImageLines {
		if parsedImageLine.err != nil {
			t.Errorf("Failed to parse: '%+v'.", parsedImageLine)
		}
		if _, ok := results[parsedImageLine]; !ok {
			t.Errorf("parsedImageResult: '%+v' not in results: '%+v'", parsedImageLine, results)
		}
		results[parsedImageLine] = true
		i++
	}
	if i != len(results) {
		t.Errorf("Want '%d' unique results. Got '%d' results.", len(results), i)
	}
	for parsedImageLine, seen := range results {
		if !seen {
			t.Errorf("Could not find expected '%+v'.", parsedImageLine)
		}
	}
}

func TestParseDockerfile(t *testing.T) {
	baseDir := filepath.Join("testassets", "parse", "dockerfile")

	// tests that compose build args overrides the ones defined in the dockerfile (simulate by passing in build vars)
	dockerfile := filepath.Join(baseDir, "override", "Dockerfile")
	//parseDockerfile(fileName string, buildArgs map[string]string, parsedImageLines chan<- parsedImageLine, wg *sync.WaitGroup)
	//parseDockerfile
	buildArgs := map[string]string{"IMAGE_NAME": "debian"}
	_ = buildArgs
	_ = dockerfile

	// test that build vars reset on each new from line

	// tests that ARG vars without values will be filled in by compose build args (similute by passing in build vars)

	// tests that compose build args, which are not defined in the Dockerfile cannot be used
	// have vars in docker-compose and try to use them in a Dockerfile where they are not defined.

	// ENV in dockerfile overrides environment variables from compose files
}
