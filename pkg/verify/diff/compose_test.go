package diff_test

import (
	"testing"

	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/verify/diff"
)

func TestComposefileDifferentiator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name        string
		Existing    map[string][]*parse.ComposefileImage
		New         map[string][]*parse.ComposefileImage
		ExcludeTags bool
		ShouldFail  bool
	}{
		{
			Name: "Different Number Of Paths",
			Existing: map[string][]*parse.ComposefileImage{
				"docker-compose.yml": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "busybox",
							Digest: "busybox",
						},
						ServiceName:    "svc",
						DockerfilePath: "Dockerfile",
					},
				},
				"docker-compose1.yml": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "busybox",
							Digest: "busybox",
						},
						ServiceName:    "svc",
						DockerfilePath: "Dockerfile",
					},
				},
			},
			New: map[string][]*parse.ComposefileImage{
				"docker-compose1.yml": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "busybox",
							Digest: "busybox",
						},
						ServiceName:    "svc",
						DockerfilePath: "Dockerfile",
					},
				},
			},
			ShouldFail: true,
		},
		{
			Name: "Different Paths",
			Existing: map[string][]*parse.ComposefileImage{
				"docker-compose.yml": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "busybox",
							Digest: "busybox",
						},
						ServiceName:    "svc",
						DockerfilePath: "Dockerfile",
					},
				},
			},
			New: map[string][]*parse.ComposefileImage{
				"docker-compose1.yml": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "busybox",
							Digest: "busybox",
						},
						ServiceName:    "svc",
						DockerfilePath: "Dockerfile",
					},
				},
			},
			ShouldFail: true,
		},
		{
			Name: "Different Images",
			Existing: map[string][]*parse.ComposefileImage{
				"docker-compose.yml": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "busybox",
							Digest: "busybox",
						},
						ServiceName:    "svc",
						DockerfilePath: "Dockerfile",
					},
				},
			},
			New: map[string][]*parse.ComposefileImage{
				"docker-compose.yml": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "notbusybox",
							Digest: "busybox",
						},
						ServiceName:    "svc",
						DockerfilePath: "Dockerfile",
					},
				},
			},
			ShouldFail: true,
		},
		{
			Name: "Different Service Names",
			Existing: map[string][]*parse.ComposefileImage{
				"docker-compose.yml": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "busybox",
							Digest: "busybox",
						},
						ServiceName:    "svc1",
						DockerfilePath: "Dockerfile",
					},
				},
			},
			New: map[string][]*parse.ComposefileImage{
				"docker-compose.yml": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "busybox",
							Digest: "busybox",
						},
						ServiceName:    "svc",
						DockerfilePath: "Dockerfile",
					},
				},
			},
			ShouldFail: true,
		},
		{
			Name: "Different Dockerfile Paths",
			Existing: map[string][]*parse.ComposefileImage{
				"docker-compose.yml": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "busybox",
							Digest: "busybox",
						},
						ServiceName:    "svc",
						DockerfilePath: "Dockerfile1",
					},
				},
			},
			New: map[string][]*parse.ComposefileImage{
				"docker-compose.yml": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "busybox",
							Digest: "busybox",
						},
						ServiceName:    "svc",
						DockerfilePath: "Dockerfile",
					},
				},
			},
			ShouldFail: true,
		},
		{
			Name: "Exclude Tags",
			Existing: map[string][]*parse.ComposefileImage{
				"docker-compose.yml": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "busybox",
							Digest: "busybox",
						},
						ServiceName:    "svc",
						DockerfilePath: "Dockerfile",
					},
				},
			},
			New: map[string][]*parse.ComposefileImage{
				"docker-compose.yml": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "notbusybox",
							Digest: "busybox",
						},
						ServiceName:    "svc",
						DockerfilePath: "Dockerfile",
					},
				},
			},
			ExcludeTags: true,
		},
		{
			Name: "Nil",
		},
		{
			Name: "Normal",
			Existing: map[string][]*parse.ComposefileImage{
				"docker-compose.yml": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "busybox",
							Digest: "busybox",
						},
						ServiceName:    "svc",
						DockerfilePath: "Dockerfile",
					},
				},
			},
			New: map[string][]*parse.ComposefileImage{
				"docker-compose.yml": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "busybox",
							Digest: "busybox",
						},
						ServiceName:    "svc",
						DockerfilePath: "Dockerfile",
					},
				},
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			differentiator := &diff.ComposefileDifferentiator{
				ExcludeTags: test.ExcludeTags,
			}

			done := make(chan struct{})
			defer close(done)

			errCh := differentiator.Differentiate(
				test.Existing,
				test.New,
				done,
			)

			err := <-errCh

			if test.ShouldFail {
				if err == nil {
					t.Fatal("expected error but did not get one")
				}

				return
			}

			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
