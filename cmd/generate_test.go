package cmd

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/michaelperel/docker-lock/generate"
)

func TestCompose(t *testing.T) {
	t.Parallel()
	baseDir := filepath.Join("testdata", "generate")
	tmpFile, err := ioutil.TempFile("", "test-compose")
	if err != nil {
		t.Error(err)
	}
	generateCmd := NewGenerateCmd()
	generateCmd.SetArgs([]string{
		"lock",
		"generate",
		fmt.Sprintf("--compose-files=%s", filepath.Join(baseDir, "docker-compose.yml")),
		fmt.Sprintf("--env-file=%s", filepath.Join(baseDir, ".env")),
		fmt.Sprintf("--outfile=%s", tmpFile.Name()),
	})
	generateCmd.Execute()
	outfile, err := generateCmd.Flags().GetString("outfile")
	if err != nil {
		t.Error(err)
	}
	composefiles, err := generateCmd.Flags().GetStringSlice("compose-files")
	if err != nil {
		t.Error(err)
	}
	lByt, err := ioutil.ReadFile(outfile)
	if err != nil {
		t.Error(err)
	}
	var lFile generate.Lockfile
	if err := json.Unmarshal(lByt, &lFile); err != nil {
		t.Error(err)
	}
	composefile := composefiles[0]
	results := map[string][]generate.ComposefileImage{composefile: []generate.ComposefileImage{
		{Image: generate.Image{Name: "nginx", Tag: "1.7"}, ServiceName: "simple", Dockerfile: ""},
		{Image: generate.Image{Name: "python", Tag: "2.7"}, ServiceName: "verbose", Dockerfile: filepath.Join(baseDir, "verbose", "Dockerfile-verbose")},
		{Image: generate.Image{Name: "python", Tag: "3.7"}, ServiceName: "verbose", Dockerfile: filepath.Join(baseDir, "verbose", "Dockerfile-verbose")},
	}}
	// TODO: Loop through the results, not the found images because len could be < len(results).
	for foundComposefile, foundImages := range lFile.ComposefileImages {
		if filepath.FromSlash(foundComposefile) != composefile {
			t.Errorf("Found '%s' composefile. Expected '%s'.", filepath.FromSlash(foundComposefile), composefile)
		}
		for i, foundImage := range foundImages {
			if results[composefile][i].Image.Name != foundImage.Image.Name ||
				results[composefile][i].Image.Tag != foundImage.Image.Tag {
				t.Errorf("Found '%s:%s'. Expected '%s:%s'.",
					foundImage.Image.Name,
					foundImage.Image.Tag,
					results[composefile][i].Image.Name,
					results[composefile][i].Image.Tag)
			}
			if foundImage.Image.Digest == "" {
				t.Errorf("%+v has an empty digest.", foundImage)
			}
			if foundImage.ServiceName != results[composefile][i].ServiceName {
				t.Errorf("Found '%s' service. Expected '%s'.",
					foundImage.ServiceName,
					results[composefile][i].ServiceName)
			}
			if filepath.FromSlash(foundImage.Dockerfile) != results[composefile][i].Dockerfile {
				t.Errorf("Found '%s' dockerfile. Expected '%s'.",
					filepath.FromSlash(foundImage.Dockerfile),
					results[composefile][i].Dockerfile)
			}
		}
	}
}

func TestPrivate(t *testing.T) {
	t.Parallel()
	baseDir := filepath.Join("testdata", "generate")
	generateCmd := NewGenerateCmd()
	tmpFile, err := ioutil.TempFile("", "test-private")
	if err != nil {
		t.Error(err)
	}
	generateCmd.SetArgs([]string{
		"lock",
		"generate",
		fmt.Sprintf("--dockerfiles=%s", filepath.Join(baseDir, "private", "Dockerfile")),
		fmt.Sprintf("--outfile=%s", tmpFile.Name()),
	})
	generateCmd.Execute()
}
