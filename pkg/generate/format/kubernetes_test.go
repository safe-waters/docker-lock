package format_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/safe-waters/docker-lock/pkg/generate/format"
	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/kind"
)

func TestKubernetesfileImageFormatter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name     string
		Images   []parse.IImage
		Expected map[string][]interface{}
	}{
		{
			Name: "Sort Kubernetesfile Images",
			Images: []parse.IImage{
				parse.NewImage(
					kind.Kubernetesfile, "redis", "latest", "",
					map[string]interface{}{
						"path":          "pod.yml",
						"imagePosition": 1,
						"docPosition":   0,
						"containerName": "redis",
					}, nil,
				),
				parse.NewImage(
					kind.Kubernetesfile, "golang", "latest", "",
					map[string]interface{}{
						"path":          "pod.yml",
						"imagePosition": 0,
						"docPosition":   0,
						"containerName": "golang",
					}, nil,
				),
				parse.NewImage(
					kind.Kubernetesfile, "busybox", "latest", "",
					map[string]interface{}{
						"path":          "pod.yml",
						"imagePosition": 0,
						"docPosition":   1,
						"containerName": "busybox",
					}, nil,
				),
				parse.NewImage(
					kind.Kubernetesfile, "golang", "latest", "",
					map[string]interface{}{
						"path":          "deployment.yml",
						"imagePosition": 0,
						"docPosition":   0,
						"containerName": "golang",
					}, nil,
				),
			},
			Expected: map[string][]interface{}{
				"pod.yml": {
					map[string]string{
						"name":      "golang",
						"tag":       "latest",
						"digest":    "",
						"container": "golang",
					},
					map[string]string{
						"name":      "redis",
						"tag":       "latest",
						"digest":    "",
						"container": "redis",
					},
					map[string]string{
						"name":      "busybox",
						"tag":       "latest",
						"digest":    "",
						"container": "busybox",
					},
				},
				"deployment.yml": {
					map[string]string{
						"name":      "golang",
						"tag":       "latest",
						"digest":    "",
						"container": "golang",
					},
				},
			},
		},
	}

	for _, test := range tests { // nolint: dupl
		test := test

		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			formatter := format.NewKubernetesfileImageFormatter()

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
