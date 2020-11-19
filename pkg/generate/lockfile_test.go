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
				{
					KubernetesfileImage: &parse.KubernetesfileImage{
						Image: &parse.Image{
							Name: "busybox",
							Tag:  "latest",
						},
						ContainerName: "busybox",
						Path:          "pod.yml",
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
				KubernetesfileImages: map[string][]*parse.KubernetesfileImage{
					"pod.yml": {
						{
							Image: &parse.Image{
								Name: "busybox",
								Tag:  "latest",
							},
							ContainerName: "busybox",
							Path:          "pod.yml",
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
			Name: "Only Kubernetesfile Images",
			AnyImages: []*generate.AnyImage{
				{
					KubernetesfileImage: &parse.KubernetesfileImage{
						Image: &parse.Image{
							Name: "busybox",
							Tag:  "latest",
						},
						ContainerName: "busybox",
						Path:          "pod.yml",
					},
				},
			},
			Expected: &generate.Lockfile{
				KubernetesfileImages: map[string][]*parse.KubernetesfileImage{
					"pod.yml": {
						{
							Image: &parse.Image{
								Name: "busybox",
								Tag:  "latest",
							},
							ContainerName: "busybox",
							Path:          "pod.yml",
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
							Name: "golang",
							Tag:  "latest",
						},
						Position: 1,
						Path:     "Dockerfile",
					},
				},
				{
					ComposefileImage: &parse.ComposefileImage{
						Image: &parse.Image{
							Name: "bash",
							Tag:  "latest",
						},
						DockerfilePath: "Dockerfile",
						Position:       1,
						ServiceName:    "svc",
						Path:           "docker-compose.yml",
					},
				},
				{
					KubernetesfileImage: &parse.KubernetesfileImage{
						Image: &parse.Image{
							Name: "busybox",
							Tag:  "latest",
						},
						DocPosition:   1,
						ImagePosition: 0,
						ContainerName: "busybox",
						Path:          "pod.yml",
					},
				},
				{
					KubernetesfileImage: &parse.KubernetesfileImage{
						Image: &parse.Image{
							Name: "golang",
							Tag:  "latest",
						},
						DocPosition:   0,
						ImagePosition: 1,
						ContainerName: "golang",
						Path:          "pod.yml",
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
					KubernetesfileImage: &parse.KubernetesfileImage{
						Image: &parse.Image{
							Name: "golang",
							Tag:  "latest",
						},
						DocPosition:   1,
						ImagePosition: 1,
						ContainerName: "golang",
						Path:          "pod.yml",
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
				{
					KubernetesfileImage: &parse.KubernetesfileImage{
						Image: &parse.Image{
							Name: "redis",
							Tag:  "latest",
						},
						DocPosition:   0,
						ImagePosition: 0,
						ContainerName: "redis",
						Path:          "pod.yml",
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
								Name: "golang",
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
								Name: "bash",
								Tag:  "latest",
							},
							DockerfilePath: "Dockerfile",
							Position:       1,
							ServiceName:    "svc",
							Path:           "docker-compose.yml",
						},
					},
				},
				KubernetesfileImages: map[string][]*parse.KubernetesfileImage{
					"pod.yml": {
						{
							Image: &parse.Image{
								Name: "redis",
								Tag:  "latest",
							},
							DocPosition:   0,
							ImagePosition: 0,
							ContainerName: "redis",
							Path:          "pod.yml",
						},
						{
							Image: &parse.Image{
								Name: "golang",
								Tag:  "latest",
							},
							DocPosition:   0,
							ImagePosition: 1,
							ContainerName: "golang",
							Path:          "pod.yml",
						},
						{
							Image: &parse.Image{
								Name: "busybox",
								Tag:  "latest",
							},
							DocPosition:   1,
							ImagePosition: 0,
							ContainerName: "busybox",
							Path:          "pod.yml",
						},
						{
							Image: &parse.Image{
								Name: "golang",
								Tag:  "latest",
							},
							DocPosition:   1,
							ImagePosition: 1,
							ContainerName: "golang",
							Path:          "pod.yml",
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
