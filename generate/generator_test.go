package generate

import (
	"encoding/json"
	"github.com/michaelperel/docker-lock/registry"
	"path/filepath"
	"testing"
)

func TestCompose(t *testing.T) {
	baseDir := filepath.Join("testdata", "generate")
	envFile := filepath.Join(baseDir, ".env")
	composefile := filepath.Join(baseDir, "docker-compose.yml")
	args := []string{"-e", envFile, "-cf", composefile}
	f, err := NewFlags(args)
	if err != nil {
		t.Fatal(err)
	}
	defaultWrapper := &registry.DockerWrapper{ConfigFile: f.ConfigFile}
	wm := registry.NewWrapperManager(defaultWrapper)
	g, err := NewGenerator(f)
	if err != nil {
		t.Fatal(err)
	}
	lByt, err := g.GenerateLockfileBytes(wm)
	if err != nil {
		t.Fatal(err)
	}
	var lFile Lockfile
	if err := json.Unmarshal(lByt, &lFile); err != nil {
		t.Fatal(err)
	}
	results := map[string][]ComposefileImage{composefile: []ComposefileImage{
		{Image: Image{Name: "nginx", Tag: "1.7"}, ServiceName: "simple", Dockerfile: ""},
		{Image: Image{Name: "python", Tag: "2.7"}, ServiceName: "verbose", Dockerfile: filepath.Join(baseDir, "verbose", "Dockerfile-verbose")},
		{Image: Image{Name: "python", Tag: "3.7"}, ServiceName: "verbose", Dockerfile: filepath.Join(baseDir, "verbose", "Dockerfile-verbose")},
	}}
	for foundComposefile, foundImages := range lFile.ComposefileImages {
		if filepath.FromSlash(foundComposefile) != composefile {
			t.Fatalf("Found '%s' composefile. Expected '%s'.", filepath.FromSlash(foundComposefile), composefile)
		}
		for i, foundImage := range foundImages {
			if results[composefile][i].Image.Name != foundImage.Image.Name ||
				results[composefile][i].Image.Tag != foundImage.Image.Tag {
				t.Fatalf("Found '%s:%s'. Expected '%s:%s'.",
					foundImage.Image.Name,
					foundImage.Image.Tag,
					results[composefile][i].Image.Name,
					results[composefile][i].Image.Tag)
			}
			if foundImage.Image.Digest == "" {
				t.Fatalf("%+v has an empty digest.", foundImage)
			}
			if foundImage.ServiceName != results[composefile][i].ServiceName {
				t.Fatalf("Found '%s' service. Expected '%s'.",
					foundImage.ServiceName,
					results[composefile][i].ServiceName)
			}
			if filepath.FromSlash(foundImage.Dockerfile) != results[composefile][i].Dockerfile {
				t.Fatalf("Found '%s' dockerfile. Expected '%s'.",
					filepath.FromSlash(foundImage.Dockerfile),
					results[composefile][i].Dockerfile)
			}
		}
	}
}
