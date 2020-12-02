package format_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/safe-waters/docker-lock/internal/testutils"
	"github.com/safe-waters/docker-lock/pkg/generate/format"
	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/kind"
)

func TestComposefileImageFormatter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name     string
		Images   []parse.IImage
		Expected map[string][]interface{}
	}{
		{
			Name: "Sort Composefile Images",
			Images: []parse.IImage{
				parse.NewImage(
					kind.Composefile, "busybox", "latest", "",
					map[string]interface{}{
						"servicePosition": 0,
						"path":            "docker-compose-one.yml",
						"serviceName":     "svc",
					}, nil,
				),
				parse.NewImage(
					kind.Composefile, "busybox", "latest", "",
					map[string]interface{}{
						"servicePosition": 0,
						"path":            "docker-compose-two.yml",
						"serviceName":     "svc",
					}, nil,
				),
				parse.NewImage(
					kind.Composefile, "redis", "latest", "",
					map[string]interface{}{
						"servicePosition": 1,
						"path":            "docker-compose-one.yml",
						"serviceName":     "anothersvc",
						"dockerfilePath":  "Dockerfile",
					}, nil,
				),
				parse.NewImage(
					kind.Composefile, "golang", "latest",
					testutils.GolangLatestSHA, map[string]interface{}{
						"servicePosition": 0,
						"path":            "docker-compose-one.yml",
						"serviceName":     "anothersvc",
						"dockerfilePath":  "Dockerfile",
					}, nil,
				),
			},
			Expected: map[string][]interface{}{
				"docker-compose-one.yml": {
					map[string]string{
						"name":       "golang",
						"tag":        "latest",
						"digest":     testutils.GolangLatestSHA,
						"dockerfile": "Dockerfile",
						"service":    "anothersvc",
					},
					map[string]string{
						"name":       "redis",
						"tag":        "latest",
						"digest":     "",
						"dockerfile": "Dockerfile",
						"service":    "anothersvc",
					},
					map[string]string{
						"name":    "busybox",
						"tag":     "latest",
						"digest":  "",
						"service": "svc",
					},
				},
				"docker-compose-two.yml": {
					map[string]string{
						"name":    "busybox",
						"tag":     "latest",
						"digest":  "",
						"service": "svc",
					},
				},
			},
		},
	}

	for _, test := range tests { // nolint: dupl
		test := test

		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			formatter := format.NewComposefileImageFormatter()

			images := make(chan parse.IImage, len(test.Images))

			for _, image := range test.Images {
				images <- image
			}
			close(images)

			formattedImages, err := formatter.FormatImages(images)
			if err != nil {
				t.Fatal(err)
			}

			formattedByt, err := json.Marshal(formattedImages)
			if err != nil {
				t.Fatal(err)
			}

			got := map[string][]interface{}{}
			if err = json.Unmarshal(formattedByt, &got); err != nil {
				t.Fatal(err)
			}

			gotByt, err := json.MarshalIndent(got, "", "\t")
			if err != nil {
				t.Fatal(err)
			}

			expectedByt, err := json.MarshalIndent(test.Expected, "", "\t")
			if err != nil {
				t.Fatal(err)
			}

			if !bytes.Equal(expectedByt, gotByt) {
				t.Fatalf(
					"expected %s\ngot %s",
					string(expectedByt), string(gotByt),
				)
			}
		})
	}
}
