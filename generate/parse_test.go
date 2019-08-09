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
		t.Fatalf("Unable to load dotenv before running test.")
	}
	composefileName := filepath.Join(baseDir, "docker-compose.yml")
	results := map[parsedImageLine]bool{
		{line: "busybox", composefileName: composefileName, dockerfileName: "", serviceName: "simple1"}:                                                         false,
		{line: "busybox", composefileName: composefileName, dockerfileName: filepath.Join(baseDir, "simple2build", "Dockerfile"), serviceName: "simple2"}:       false,
		{line: "busybox", composefileName: composefileName, dockerfileName: filepath.Join(baseDir, "simple3build", "Dockerfile"), serviceName: "simple3"}:       false,
		{line: "busybox", composefileName: composefileName, dockerfileName: filepath.Join(baseDir, "simple4build", "Dockerfile"), serviceName: "simple4"}:       false,
		{line: "busybox", composefileName: composefileName, dockerfileName: filepath.Join(baseDir, "verbose1build", "Dockerfile"), serviceName: "verbose1"}:     false,
		{line: "busybox", composefileName: composefileName, dockerfileName: filepath.Join(baseDir, "verbose2build", "Dockerfile"), serviceName: "verbose2"}:     false,
		{line: "busybox", composefileName: composefileName, dockerfileName: filepath.Join(baseDir, "verbose3build", "Dockerfile"), serviceName: "verbose3"}:     false,
		{line: "busybox", composefileName: composefileName, dockerfileName: filepath.Join(baseDir, "verbose4build", "Dockerfile-dev"), serviceName: "verbose4"}: false,
	}
	parsedImageLines := make(chan parsedImageLine)
	var wg sync.WaitGroup
	go func() {
		wg.Add(1)
		parseComposefile(composefileName, parsedImageLines, &wg)
		wg.Wait()
		close(parsedImageLines)
	}()
	var i int
	for imLine := range parsedImageLines {
		if imLine.err != nil {
			t.Fatalf("Failed to parse. Composefile: '%s'. Dockerfile: '%s'. Err: '%s'.",
				imLine.composefileName,
				imLine.dockerfileName,
				imLine.err)
		}
		if _, ok := results[imLine]; !ok {
			t.Fatalf("parsedImageResult: '%+v' not in results: '%+v'.", imLine, results)
		}
		results[imLine] = true
		i++
	}
	if i != len(results) {
		t.Fatalf("Got '%d' unique results. Want '%d' results.", i, len(results))
	}
	for imLine, seen := range results {
		if !seen {
			t.Fatalf("Could not find expected '%+v'.", imLine)
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
	go parseDockerfile(dockerfile, composeArgs, "", "", parsedImageLines, nil)
	result := <-parsedImageLines
	if result.line != composeArgs["IMAGE_NAME"] {
		t.Fatalf("Got '%s'. Want '%s'.", result.line, composeArgs["IMAGE_NAME"])
	}
}

func TestParseDockerfileEmpty(t *testing.T) {
	// Empty ARG in Dockerfile, definition in composefile.
	// composefile should override.
	baseDir := filepath.Join("testdata", "parse", "dockerfile")
	dockerfile := filepath.Join(baseDir, "empty", "Dockerfile")
	composeArgs := map[string]string{"IMAGE_NAME": "debian"}
	parsedImageLines := make(chan parsedImageLine)
	go parseDockerfile(dockerfile, composeArgs, "", "", parsedImageLines, nil)
	result := <-parsedImageLines
	if result.line != composeArgs["IMAGE_NAME"] {
		t.Fatalf("Got '%s'. Want '%s'.", result.line, composeArgs["IMAGE_NAME"])
	}
}

func TestParseDockerfileNoArg(t *testing.T) {
	// ARG defined in Dockerfile, not in composefile.
	// Should behave as though no composefile existed.
	baseDir := filepath.Join("testdata", "parse", "dockerfile")
	dockerfile := filepath.Join(baseDir, "noarg", "Dockerfile")
	parsedImageLines := make(chan parsedImageLine)
	go parseDockerfile(dockerfile, nil, "", "", parsedImageLines, nil)
	result := <-parsedImageLines
	imageName := "busybox"
	if result.line != imageName {
		t.Fatalf("Got '%s'. Want '%s'.", result.line, imageName)
	}
}

func TestParseDockerfileLocalArg(t *testing.T) {
	// ARG defined before FROM (aka global arg) should not
	// be overridden by ARG defined after FROM (aka local arg)
	baseDir := filepath.Join("testdata", "parse", "dockerfile")
	dockerfile := filepath.Join(baseDir, "localarg", "Dockerfile")
	parsedImageLines := make(chan parsedImageLine)
	go parseDockerfile(dockerfile, nil, "", "", parsedImageLines, nil)
	results := []parsedImageLine{<-parsedImageLines, <-parsedImageLines}
	imageName := "busybox"
	for _, result := range results {
		if result.line != imageName {
			t.Fatalf("Got '%s'. Want '%s'.", result.line, imageName)
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
	go parseDockerfile(dockerfile, nil, "", "", parsedImageLines, nil)
	results := []parsedImageLine{<-parsedImageLines, <-parsedImageLines}
	imageNames := []string{"busybox", "ubuntu"}
	for i, result := range results {
		if result.line != imageNames[i] {
			t.Fatalf("Got '%s'. Want '%s'.", result.line, imageNames[i])
		}
	}

}
