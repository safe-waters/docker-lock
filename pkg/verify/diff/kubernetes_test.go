package diff_test

import (
	"testing"

	"github.com/safe-waters/docker-lock/pkg/verify/diff"
)

func TestKubernetesfileDifferentiator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name        string
		Existing    map[string]interface{}
		New         map[string]interface{}
		ExcludeTags bool
		ShouldFail  bool
	}{
		{
			Name: "Different Name",
			Existing: map[string]interface{}{
				"name":      "busybox",
				"tag":       "latest",
				"digest":    "busybox",
				"container": "busybox",
			},
			New: map[string]interface{}{
				"name":      "redis",
				"tag":       "latest",
				"digest":    "busybox",
				"container": "busybox",
			},
			ShouldFail: true,
		},
		{
			Name: "Different Tag",
			Existing: map[string]interface{}{
				"name":      "busybox",
				"tag":       "latest",
				"digest":    "busybox",
				"container": "busybox",
			},
			New: map[string]interface{}{
				"name":      "busybox",
				"tag":       "busybox",
				"digest":    "busybox",
				"container": "busybox",
			},
			ShouldFail: true,
		},
		{
			Name: "Different Digest",
			Existing: map[string]interface{}{
				"name":      "busybox",
				"tag":       "latest",
				"digest":    "busybox",
				"container": "busybox",
			},
			New: map[string]interface{}{
				"name":      "busybox",
				"tag":       "latest",
				"digest":    "unknown",
				"container": "busybox",
			},
			ShouldFail: true,
		},
		{
			Name: "Different Container",
			Existing: map[string]interface{}{
				"name":      "busybox",
				"tag":       "latest",
				"digest":    "busybox",
				"container": "busybox",
			},
			New: map[string]interface{}{
				"name":      "busybox",
				"tag":       "latest",
				"digest":    "busybox",
				"container": "busybox1",
			},
			ShouldFail: true,
		},
		{
			Name: "Exclude Tags",
			Existing: map[string]interface{}{
				"name":      "busybox",
				"tag":       "latest",
				"digest":    "busybox",
				"container": "busybox",
			},
			New: map[string]interface{}{
				"name":      "busybox",
				"tag":       "unknown",
				"digest":    "busybox",
				"container": "busybox",
			},
			ExcludeTags: true,
			ShouldFail:  false,
		},
		{
			Name: "Normal",
			Existing: map[string]interface{}{
				"name":      "busybox",
				"tag":       "latest",
				"digest":    "busybox",
				"container": "busybox",
			},
			New: map[string]interface{}{
				"name":      "busybox",
				"tag":       "latest",
				"digest":    "busybox",
				"container": "busybox",
			},
			ShouldFail: false,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			differentiator := diff.NewKubernetesfileDifferentiator(
				test.ExcludeTags,
			)
			err := differentiator.DifferentiateImage(test.Existing, test.New)

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
