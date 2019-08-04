package generate

import (
	"os"
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

	baseDir := "testassets/parse/composefile/"

	results := map[parsedImageLine]bool{
		{line: "busybox", fileName: baseDir + "docker-compose.yml", err: nil}:           false,
		{line: "busybox", fileName: baseDir + "simple2build/Dockerfile", err: nil}:      false,
		{line: "busybox", fileName: baseDir + "simple3build/Dockerfile", err: nil}:      false,
		{line: "busybox", fileName: baseDir + "simple4build/Dockerfile", err: nil}:      false,
		{line: "busybox", fileName: baseDir + "verbose1build/Dockerfile", err: nil}:     false,
		{line: "busybox", fileName: baseDir + "verbose2build/Dockerfile", err: nil}:     false,
		{line: "busybox", fileName: baseDir + "verbose3build/Dockerfile", err: nil}:     false,
		{line: "busybox", fileName: baseDir + "verbose4build/Dockerfile-dev", err: nil}: false,
	}
	parsedImageLines := make(chan parsedImageLine)
	var wg sync.WaitGroup
	go func() {
		wg.Add(1)
		parseComposefile(baseDir+"docker-compose.yml", parsedImageLines, &wg)
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
}
