package generate_test

import (
	"testing"

	"github.com/safe-waters/docker-lock/pkg/generate"
	"github.com/safe-waters/docker-lock/pkg/generate/parse"
)

func TestLockfile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name      string
		AnyImages []*generate.AnyImage
		Expected  *generate.Lockfile
	}{
		{
			Name:     "Nil Images",
			Expected: &generate.Lockfile{},
		},
		{
			Name: "Non Nil Images",
			AnyImages: []*generate.AnyImage{
				{
					DockerfileImage: &parse.DockerfileImage{
						Image: &parse.Image{
							Name: "busybox",
							Tag:  "latest",
						},
						Path: "Dockerfile",
					},
				},
				{
					ComposefileImage: &parse.ComposefileImage{
						Image: &parse.Image{
							Name: "busybox",
							Tag:  "latest",
						},
						DockerfilePath: "Dockerfile",
						ServiceName:    "svc",
						Path:           "docker-compose.yml",
					},
				},
			},
			Expected: &generate.Lockfile{
				DockerfileImages: map[string][]*parse.DockerfileImage{
					"Dockerfile": {
						{
							Image: &parse.Image{
								Name: "busybox",
								Tag:  "latest",
							},
							Path: "Dockerfile",
						},
					},
				},
				ComposefileImages: map[string][]*parse.ComposefileImage{
					"docker-compose.yml": {
						{
							Image: &parse.Image{
								Name: "busybox",
								Tag:  "latest",
							},
							DockerfilePath: "Dockerfile",
							ServiceName:    "svc",
							Path:           "docker-compose.yml",
						},
					},
				},
			},
		},
		{
			Name: "Only Composefile Images",
			AnyImages: []*generate.AnyImage{
				{
					ComposefileImage: &parse.ComposefileImage{
						Image: &parse.Image{
							Name: "busybox",
							Tag:  "latest",
						},
						DockerfilePath: "Dockerfile",
						ServiceName:    "svc",
						Path:           "docker-compose.yml",
					},
				},
			},
			Expected: &generate.Lockfile{
				ComposefileImages: map[string][]*parse.ComposefileImage{
					"docker-compose.yml": {
						{
							Image: &parse.Image{
								Name: "busybox",
								Tag:  "latest",
							},
							DockerfilePath: "Dockerfile",
							ServiceName:    "svc",
							Path:           "docker-compose.yml",
						},
					},
				},
			},
		},
		{
			Name: "Only Dockerfile Images",
			AnyImages: []*generate.AnyImage{
				{
					DockerfileImage: &parse.DockerfileImage{
						Image: &parse.Image{
							Name: "busybox",
							Tag:  "latest",
						},
						Path: "Dockerfile",
					},
				},
			},
			Expected: &generate.Lockfile{
				DockerfileImages: map[string][]*parse.DockerfileImage{
					"Dockerfile": {
						{
							Image: &parse.Image{
								Name: "busybox",
								Tag:  "latest",
							},
							Path: "Dockerfile",
						},
					},
				},
			},
		},
		{
			Name: "Sorted Images",
			AnyImages: []*generate.AnyImage{
				{
					DockerfileImage: &parse.DockerfileImage{
						Image: &parse.Image{
							Name: "busybox",
							Tag:  "latest",
						},
						Position: 1,
						Path:     "Dockerfile",
					},
				},
				{
					ComposefileImage: &parse.ComposefileImage{
						Image: &parse.Image{
							Name: "busybox",
							Tag:  "latest",
						},
						DockerfilePath: "Dockerfile",
						Position:       1,
						ServiceName:    "svc",
						Path:           "docker-compose.yml",
					},
				},
				{
					ComposefileImage: &parse.ComposefileImage{
						Image: &parse.Image{
							Name: "busybox",
							Tag:  "latest",
						},
						DockerfilePath: "Dockerfile",
						Position:       0,
						ServiceName:    "svc",
						Path:           "docker-compose.yml",
					},
				},
				{
					DockerfileImage: &parse.DockerfileImage{
						Image: &parse.Image{
							Name: "busybox",
							Tag:  "latest",
						},
						Position: 0,
						Path:     "Dockerfile",
					},
				},
			},
			Expected: &generate.Lockfile{
				DockerfileImages: map[string][]*parse.DockerfileImage{
					"Dockerfile": {
						{
							Image: &parse.Image{
								Name: "busybox",
								Tag:  "latest",
							},
							Position: 0,
							Path:     "Dockerfile",
						},
						{
							Image: &parse.Image{
								Name: "busybox",
								Tag:  "latest",
							},
							Position: 1,
							Path:     "Dockerfile",
						},
					},
				},
				ComposefileImages: map[string][]*parse.ComposefileImage{
					"docker-compose.yml": {
						{
							Image: &parse.Image{
								Name: "busybox",
								Tag:  "latest",
							},
							DockerfilePath: "Dockerfile",
							Position:       0,
							ServiceName:    "svc",
							Path:           "docker-compose.yml",
						},
						{
							Image: &parse.Image{
								Name: "busybox",
								Tag:  "latest",
							},
							DockerfilePath: "Dockerfile",
							Position:       1,
							ServiceName:    "svc",
							Path:           "docker-compose.yml",
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			var anyImagesCh chan *generate.AnyImage

			if len(test.AnyImages) != 0 {
				anyImagesCh = make(chan *generate.AnyImage, len(test.AnyImages))

				for _, anyImage := range test.AnyImages {
					anyImagesCh <- anyImage
				}
				close(anyImagesCh)
			}

			got, err := generate.NewLockfile(anyImagesCh)
			if err != nil {
				t.Fatal(err)
			}

			assertLockfilesEqual(t, test.Expected, got)

			assertDefaultValuesForOmittedJSONReadFromLockfile(t, got)
		})
	}
}
