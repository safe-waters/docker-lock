package diff_test

import (
	"testing"

	"github.com/safe-waters/docker-lock/pkg/verify/diff"
)

func TestComposefileDifferentiator(t *testing.T) {
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
				"name":    "busybox",
				"tag":     "latest",
				"digest":  "busybox",
				"service": "svc",
			},
			New: map[string]interface{}{
				"name":    "redis",
				"tag":     "latest",
				"digest":  "busybox",
				"service": "svc",
			},
			ShouldFail: true,
		},
		{
			Name: "Different Tag",
			Existing: map[string]interface{}{
				"name":    "busybox",
				"tag":     "latest",
				"digest":  "busybox",
				"service": "svc",
			},
			New: map[string]interface{}{
				"name":    "busybox",
				"tag":     "busybox",
				"digest":  "busybox",
				"service": "svc",
			},
			ShouldFail: true,
		},
		{
			Name: "Different Digest",
			Existing: map[string]interface{}{
				"name":    "busybox",
				"tag":     "latest",
				"digest":  "busybox",
				"service": "svc",
			},
			New: map[string]interface{}{
				"name":    "busybox",
				"tag":     "latest",
				"digest":  "unknown",
				"service": "svc",
			},
			ShouldFail: true,
		},
		{
			Name: "Different Service",
			Existing: map[string]interface{}{
				"name":    "busybox",
				"tag":     "latest",
				"digest":  "busybox",
				"service": "svc",
			},
			New: map[string]interface{}{
				"name":    "busybox",
				"tag":     "latest",
				"digest":  "busybox",
				"service": "svc1",
			},
			ShouldFail: true,
		},
		{
			Name: "Different Dockerfile",
			Existing: map[string]interface{}{
				"name":       "busybox",
				"tag":        "latest",
				"digest":     "busybox",
				"service":    "svc",
				"dockerfile": "Dockerfile",
			},
			New: map[string]interface{}{
				"name":       "busybox",
				"tag":        "latest",
				"digest":     "busybox",
				"service":    "svc",
				"dockerfile": "Dockerfile1",
			},
			ShouldFail: true,
		},
		{
			Name: "Exclude Tags",
			Existing: map[string]interface{}{
				"name":    "busybox",
				"tag":     "latest",
				"digest":  "busybox",
				"service": "svc",
			},
			New: map[string]interface{}{
				"name":    "busybox",
				"tag":     "unknown",
				"digest":  "busybox",
				"service": "svc",
			},
			ExcludeTags: true,
			ShouldFail:  false,
		},
		{
			Name: "Normal",
			Existing: map[string]interface{}{
				"name":    "busybox",
				"tag":     "latest",
				"digest":  "busybox",
				"service": "svc",
			},
			New: map[string]interface{}{
				"name":    "busybox",
				"tag":     "latest",
				"digest":  "busybox",
				"service": "svc",
			},
			ShouldFail: false,
		},
	}

	for _, test := range tests {
		test := test

		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			differentiator := diff.NewComposefileDifferentiator(
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
