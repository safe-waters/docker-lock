package generate

import (
	"path/filepath"
	"sync"
	"testing"

	"github.com/joho/godotenv"
)

func TestParseComposeFile(t *testing.T) {
	baseDir := filepath.Join("testdata", "parse", "composefile")
	if err := godotenv.Load(filepath.Join(baseDir, ".env")); err != nil {
		t.Errorf("Unable to load dotenv before running test.")
	}
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
	for imLine := range parsedImageLines {
		if imLine.err != nil {
			t.Errorf("Failed to parse filename: '%s' err: '%s'.", imLine.fileName, imLine.err)
		}
		if _, ok := results[imLine]; !ok {
			t.Errorf("parsedImageResult: '%+v' not in results: '%+v'.", imLine, results)
		}
		results[imLine] = true
		i++
	}
	if i != len(results) {
		t.Errorf("Got '%d' unique results. Want '%d' results.", i, len(results))
	}
	for imLine, seen := range results {
		if !seen {
			t.Errorf("Could not find expected '%+v'.", imLine)
		}
	}
}

func TestParseDockerfileOverride(t *testing.T) {
	// ARG in Dockerfile, also in composefile.
	// composefile should override.
	baseDir := filepath.Join("testdata", "parse", "dockerfile")
	dockerfile := filepath.Join(baseDir, "override", "Dockerfile")
	composeArgs := map[string]string{"IMAGE_NAME": "debian"}
	parsedImageLines := make(chan parsedImageLine)
	go parseDockerfile(dockerfile, composeArgs, parsedImageLines, nil)
	result := <-parsedImageLines
	if result.line != composeArgs["IMAGE_NAME"] {
		t.Errorf("Got '%s'. Want '%s'.", result.line, composeArgs["IMAGE_NAME"])
	}
}

func TestParseDockerfileEmpty(t *testing.T) {
	// Empty ARG in Dockerfile, definition in composefile.
	// composefile should override.
	baseDir := filepath.Join("testdata", "parse", "dockerfile")
	dockerfile := filepath.Join(baseDir, "empty", "Dockerfile")
	composeArgs := map[string]string{"IMAGE_NAME": "debian"}
	parsedImageLines := make(chan parsedImageLine)
	go parseDockerfile(dockerfile, composeArgs, parsedImageLines, nil)
	result := <-parsedImageLines
	if result.line != composeArgs["IMAGE_NAME"] {
		t.Errorf("Got '%s'. Want '%s'.", result.line, composeArgs["IMAGE_NAME"])
	}
}

func TestParseDockerfileNoArg(t *testing.T) {
	// ARG defined in Dockerfile, not in composefile.
	// Should behave as though no composefile existed.
	baseDir := filepath.Join("testdata", "parse", "dockerfile")
	dockerfile := filepath.Join(baseDir, "noarg", "Dockerfile")
	parsedImageLines := make(chan parsedImageLine)
	go parseDockerfile(dockerfile, nil, parsedImageLines, nil)
	result := <-parsedImageLines
	imageName := "busybox"
	if result.line != imageName {
		t.Errorf("Got '%s'. Want '%s'.", result.line, imageName)
	}
}

func TestParseDockerfileLocalArg(t *testing.T) {
	// ARG defined before FROM (aka global arg) should not
	// be overridden by ARG defined after FROM (aka local arg)
	baseDir := filepath.Join("testdata", "parse", "dockerfile")
	dockerfile := filepath.Join(baseDir, "localarg", "Dockerfile")
	parsedImageLines := make(chan parsedImageLine)
	go parseDockerfile(dockerfile, nil, parsedImageLines, nil)
	results := []parsedImageLine{<-parsedImageLines, <-parsedImageLines}
	imageName := "busybox"
	for _, result := range results {
		if result.line != imageName {
			t.Errorf("Got '%s'. Want '%s'.", result.line, imageName)
		}
	}
}

func TestParseDockerfileBuildStage(t *testing.T) {
	// Build stages should not be parsed.
	// For instance:
	// # Dockerfile
	// FROM busybox AS busy
	// FROM busy AS anotherbusy
	// should only parse 'busybox', the second field in the first line.
	baseDir := filepath.Join("testdata", "parse", "dockerfile")
	dockerfile := filepath.Join(baseDir, "buildstage", "Dockerfile")
	parsedImageLines := make(chan parsedImageLine)
	go parseDockerfile(dockerfile, nil, parsedImageLines, nil)
	results := []parsedImageLine{<-parsedImageLines, <-parsedImageLines}
	imageNames := []string{"busybox", "ubuntu"}
	for i, result := range results {
		if result.line != imageNames[i] {
			t.Errorf("Got '%s'. Want '%s'.", result.line, imageNames[i])
		}
	}

}
