package diff_test

import (
	"testing"

	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/verify/diff"
)

func TestDockerfileDifferentiator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name        string
		Existing    map[string][]*parse.DockerfileImage
		New         map[string][]*parse.DockerfileImage
		ExcludeTags bool
		ShouldFail  bool
	}{
		{
			Name: "Different Number Of Paths",
			Existing: map[string][]*parse.DockerfileImage{
				"Dockerfile": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "busybox",
							Digest: "busybox",
						},
					},
				},
				"Dockerfile1": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "busybox",
							Digest: "busybox",
						},
					},
				},
			},
			New: map[string][]*parse.DockerfileImage{
				"Dockerfile1": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "busybox",
							Digest: "busybox",
						},
					},
				},
			},
			ShouldFail: true,
		},
		{
			Name: "Different Paths",
			Existing: map[string][]*parse.DockerfileImage{
				"Dockerfile": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "busybox",
							Digest: "busybox",
						},
					},
				},
			},
			New: map[string][]*parse.DockerfileImage{
				"Dockerfile1": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "busybox",
							Digest: "busybox",
						},
					},
				},
			},
			ShouldFail: true,
		},
		{
			Name: "Different Images",
			Existing: map[string][]*parse.DockerfileImage{
				"Dockerfile": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "busybox",
							Digest: "busybox",
						},
					},
				},
			},
			New: map[string][]*parse.DockerfileImage{
				"Dockerfile": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "notbusybox",
							Digest: "busybox",
						},
					},
				},
			},
			ShouldFail: true,
		},
		{
			Name: "Exclude Tags",
			Existing: map[string][]*parse.DockerfileImage{
				"Dockerfile": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "busybox",
							Digest: "busybox",
						},
					},
				},
			},
			New: map[string][]*parse.DockerfileImage{
				"Dockerfile": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "notbusybox",
							Digest: "busybox",
						},
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
			Existing: map[string][]*parse.DockerfileImage{
				"Dockerfile": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "busybox",
							Digest: "busybox",
						},
					},
				},
			},
			New: map[string][]*parse.DockerfileImage{
				"Dockerfile": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "busybox",
							Digest: "busybox",
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

			differentiator := &diff.DockerfileDifferentiator{
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
