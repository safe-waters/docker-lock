package diff_test

import (
	"testing"

	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/verify/diff"
)

func TestKubernetesfileDifferentiator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name        string
		Existing    map[string][]*parse.KubernetesfileImage
		New         map[string][]*parse.KubernetesfileImage
		ExcludeTags bool
		ShouldFail  bool
	}{
		{
			Name: "Different Number Of Paths",
			Existing: map[string][]*parse.KubernetesfileImage{
				"pod.yml": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "busybox",
							Digest: "busybox",
						},
						ContainerName: "svc",
					},
				},
				"pod1.yml": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "busybox",
							Digest: "busybox",
						},
						ContainerName: "svc1",
					},
				},
			},
			New: map[string][]*parse.KubernetesfileImage{
				"pod.yml": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "busybox",
							Digest: "busybox",
						},
						ContainerName: "svc",
					},
				},
			},
			ShouldFail: true,
		},
		{
			Name: "Different Paths",
			Existing: map[string][]*parse.KubernetesfileImage{
				"pod.yml": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "busybox",
							Digest: "busybox",
						},
						ContainerName: "svc",
					},
				},
			},
			New: map[string][]*parse.KubernetesfileImage{
				"pod1.yml": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "busybox",
							Digest: "busybox",
						},
						ContainerName: "svc",
					},
				},
			},
			ShouldFail: true,
		},
		{
			Name: "Different Images",
			Existing: map[string][]*parse.KubernetesfileImage{
				"pod.yml": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "busybox",
							Digest: "busybox",
						},
						ContainerName: "svc",
					},
				},
			},
			New: map[string][]*parse.KubernetesfileImage{
				"pod.yml": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "notbusybox",
							Digest: "busybox",
						},
						ContainerName: "svc",
					},
				},
			},
			ShouldFail: true,
		},
		{
			Name: "Different Container Names",
			Existing: map[string][]*parse.KubernetesfileImage{
				"pod.yml": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "busybox",
							Digest: "busybox",
						},
						ContainerName: "svc1",
					},
				},
			},
			New: map[string][]*parse.KubernetesfileImage{
				"pod.yml": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "busybox",
							Digest: "busybox",
						},
						ContainerName: "svc",
					},
				},
			},
			ShouldFail: true,
		},
		{
			Name: "Exclude Tags",
			Existing: map[string][]*parse.KubernetesfileImage{
				"pod.yml": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "busybox",
							Digest: "busybox",
						},
						ContainerName: "svc",
					},
				},
			},
			New: map[string][]*parse.KubernetesfileImage{
				"pod.yml": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "notbusybox",
							Digest: "busybox",
						},
						ContainerName: "svc",
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
			Existing: map[string][]*parse.KubernetesfileImage{
				"pod.yml": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "busybox",
							Digest: "busybox",
						},
						ContainerName: "svc",
					},
				},
			},
			New: map[string][]*parse.KubernetesfileImage{
				"pod.yml": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "busybox",
							Digest: "busybox",
						},
						ContainerName: "svc",
					},
				},
			},
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			differentiator := &diff.KubernetesfileDifferentiator{
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
