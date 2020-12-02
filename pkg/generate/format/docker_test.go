package format_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/safe-waters/docker-lock/pkg/generate/format"
	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/kind"
)

func TestDockerfileImageFormatter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name     string
		Images   []parse.IImage
		Expected map[string][]interface{}
	}{
		{
			Name: "Sort Dockerfile Images",
			Images: []parse.IImage{
				parse.NewImage(
					kind.Dockerfile, "golang", "latest", "",
					map[string]interface{}{
						"position": 1,
						"path":     "Dockerfile1",
					}, nil,
				),
				parse.NewImage(
					kind.Dockerfile, "busybox", "latest", "",
					map[string]interface{}{
						"position": 0,
						"path":     "Dockerfile2",
					}, nil,
				),
				parse.NewImage(
					kind.Dockerfile, "redis", "latest", "",
					map[string]interface{}{
						"position": 0,
						"path":     "Dockerfile1",
					}, nil,
				),
				parse.NewImage(
					kind.Dockerfile, "busybox", "latest", "",
					map[string]interface{}{
						"position": 2,
						"path":     "Dockerfile1",
					}, nil,
				),
			},
			Expected: map[string][]interface{}{
				"Dockerfile1": {
					map[string]string{
						"name":   "redis",
						"tag":    "latest",
						"digest": "",
					},
					map[string]string{
						"name":   "golang",
						"tag":    "latest",
						"digest": "",
					},
					map[string]string{
						"name":   "busybox",
						"tag":    "latest",
						"digest": "",
					},
				},
				"Dockerfile2": {
					map[string]string{
						"name":   "busybox",
						"tag":    "latest",
						"digest": "",
					},
				},
			},
		},
	}

	for _, test := range tests { // nolint: dupl
		test := test

		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			formatter := format.NewDockerfileImageFormatter()

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
