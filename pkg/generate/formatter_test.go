package generate_test

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/safe-waters/docker-lock/internal/testutils"
	"github.com/safe-waters/docker-lock/pkg/generate"
	"github.com/safe-waters/docker-lock/pkg/generate/format"
	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/kind"
)

func TestImageFormatter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name     string
		Images   []parse.IImage
		Expected map[string]map[string][]interface{}
	}{
		{
			Name: "All Images",
			Images: []parse.IImage{
				parse.NewImage(
					kind.Dockerfile, "golang", "latest", "",
					map[string]interface{}{
						"position": 1,
						"path":     "Dockerfile1",
					}, nil,
				),
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
					kind.Composefile, "busybox", "latest", "",
					map[string]interface{}{
						"servicePosition": 0,
						"path":            "docker-compose-one.yml",
						"serviceName":     "svc",
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
				parse.NewImage(
					kind.Dockerfile, "redis", "latest", "",
					map[string]interface{}{
						"position": 0,
						"path":     "Dockerfile1",
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
				parse.NewImage(
					kind.Dockerfile, "busybox", "latest", "",
					map[string]interface{}{
						"position": 2,
						"path":     "Dockerfile1",
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
			},
			Expected: map[string]map[string][]interface{}{
				"dockerfiles": {
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
				"composefiles": {
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
				"kubernetesfiles": {
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
		},
	}

	for _, test := range tests { // nolint: dupl
		test := test

		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			formatters := []format.IImageFormatter{
				format.NewDockerfileImageFormatter(),
				format.NewComposefileImageFormatter(),
				format.NewKubernetesfileImageFormatter(),
			}

			formatter, err := generate.NewImageFormatter(formatters...)
			if err != nil {
				t.Fatal(err)
			}

			images := make(chan parse.IImage, len(test.Images))

			for _, image := range test.Images {
				images <- image
			}
			close(images)

			done := make(chan struct{})
			defer close(done)

			formattedImages, err := formatter.FormatImages(images, done)
			if err != nil {
				t.Fatal(err)
			}

			formattedByt, err := json.Marshal(formattedImages)
			if err != nil {
				t.Fatal(err)
			}

			got := map[string]map[string][]interface{}{}
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
