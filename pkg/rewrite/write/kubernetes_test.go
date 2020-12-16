package write_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/safe-waters/docker-lock/internal/testutils"
	"github.com/safe-waters/docker-lock/pkg/rewrite/write"
)

func TestKubernetesfileWriter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name        string
		Contents    [][]byte
		Expected    [][]byte
		PathImages  map[string][]interface{}
		ExcludeTags bool
		ShouldFail  bool
	}{
		{
			Name: "Single Doc",
			Contents: [][]byte{
				[]byte(`apiVersion: v1
kind: Pod
metadata:
  name: test
  labels:
    app: test
spec:
  containers:
  - name: busybox
    image: busybox
    ports:
    - containerPort: 80
  - name: golang
    image: golang
    ports:
    - containerPort: 88
`),
			},
			PathImages: map[string][]interface{}{
				"pod.yml": {
					map[string]interface{}{
						"name":      "busybox",
						"tag":       "latest",
						"digest":    "busybox",
						"container": "busybox",
					},
					map[string]interface{}{
						"name":      "golang",
						"tag":       "latest",
						"digest":    "golang",
						"container": "golang",
					},
				},
			},
			Expected: [][]byte{
				[]byte(`apiVersion: v1
kind: Pod
metadata:
  name: test
  labels:
    app: test
spec:
  containers:
  - name: busybox
    image: busybox:latest@sha256:busybox
    ports:
    - containerPort: 80
  - name: golang
    image: golang:latest@sha256:golang
    ports:
    - containerPort: 88
`),
			},
		},
		{
			Name: "Multiple Docs",
			Contents: [][]byte{
				[]byte(`apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: test
  name: test
spec:
  replicas: 3
  selector:
    matchLabels:
      app: test
  template:
    metadata:
      labels:
        app: test
    spec:
      containers:
      - name: golang
        image: golang
        ports:
        - containerPort: 80
      - name: python
        image: python
        ports:
        - containerPort: 81
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: test
  name: test
spec:
  replicas: 3
  selector:
    matchLabels:
      app: test
  template:
    metadata:
      labels:
        app: test
    spec:
      containers:
      - name: redis
        image: redis
        ports:
        - containerPort: 80
      - image: bash
        name: bash
`),
			},
			PathImages: map[string][]interface{}{
				"deployment.yaml": {
					map[string]interface{}{
						"name":      "golang",
						"tag":       "latest",
						"digest":    "golang",
						"container": "golang",
					},
					map[string]interface{}{
						"name":      "python",
						"tag":       "latest",
						"digest":    "python",
						"container": "python",
					},
					map[string]interface{}{
						"name":      "redis",
						"tag":       "latest",
						"digest":    "redis",
						"container": "redis",
					},
					map[string]interface{}{
						"name":      "bash",
						"tag":       "latest",
						"digest":    "bash",
						"container": "bash",
					},
				},
			},
			Expected: [][]byte{
				[]byte(`apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: test
  name: test
spec:
  replicas: 3
  selector:
    matchLabels:
      app: test
  template:
    metadata:
      labels:
        app: test
    spec:
      containers:
      - name: golang
        image: golang:latest@sha256:golang
        ports:
        - containerPort: 80
      - name: python
        image: python:latest@sha256:python
        ports:
        - containerPort: 81
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: test
  name: test
spec:
  replicas: 3
  selector:
    matchLabels:
      app: test
  template:
    metadata:
      labels:
        app: test
    spec:
      containers:
      - name: redis
        image: redis:latest@sha256:redis
        ports:
        - containerPort: 80
      - image: bash:latest@sha256:bash
        name: bash
`),
			},
		},
		{
			Name: "Multiple Files",
			Contents: [][]byte{
				[]byte(`apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: test
  name: test
spec:
  replicas: 3
  selector:
    matchLabels:
      app: test
  template:
    metadata:
      labels:
        app: test
    spec:
      containers:
      - name: golang
        image: golang
        ports:
        - containerPort: 80
      - name: python
        image: python
        ports:
        - containerPort: 81
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: test
  name: test
spec:
  replicas: 3
  selector:
    matchLabels:
      app: test
  template:
    metadata:
      labels:
        app: test
    spec:
      containers:
      - name: redis
        image: redis
        ports:
        - containerPort: 80
      - image: bash
        name: bash
`),
				[]byte(`apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: test
  name: test
spec:
  replicas: 3
  selector:
    matchLabels:
      app: test
  template:
    metadata:
      labels:
        app: test
    spec:
      containers:
      - name: busybox
        image: busybox
        ports:
        - containerPort: 80
      - name: java
        image: java
        ports:
        - containerPort: 81
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: test
  name: test
spec:
  replicas: 3
  selector:
    matchLabels:
      app: test
  template:
    metadata:
      labels:
        app: test
    spec:
      containers:
      - name: alpine
        image: alpine
        ports:
        - containerPort: 80
      - image: ruby
        name: ruby
`),
			},
			PathImages: map[string][]interface{}{
				"deployment.yaml": {
					map[string]interface{}{
						"name":      "golang",
						"tag":       "latest",
						"digest":    "golang",
						"container": "golang",
					},
					map[string]interface{}{
						"name":      "python",
						"tag":       "latest",
						"digest":    "python",
						"container": "python",
					},
					map[string]interface{}{
						"name":      "redis",
						"tag":       "latest",
						"digest":    "redis",
						"container": "redis",
					},
					map[string]interface{}{
						"name":      "bash",
						"tag":       "latest",
						"digest":    "bash",
						"container": "bash",
					},
				},
				"deployment1.yaml": {
					map[string]interface{}{
						"name":      "busybox",
						"tag":       "latest",
						"digest":    "busybox",
						"container": "busybox",
					},
					map[string]interface{}{
						"name":      "java",
						"tag":       "latest",
						"digest":    "java",
						"container": "java",
					},
					map[string]interface{}{
						"name":      "alpine",
						"tag":       "latest",
						"digest":    "alpine",
						"container": "alpine",
					},
					map[string]interface{}{
						"name":      "ruby",
						"tag":       "latest",
						"digest":    "ruby",
						"container": "ruby",
					},
				},
			},
			Expected: [][]byte{
				[]byte(`apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: test
  name: test
spec:
  replicas: 3
  selector:
    matchLabels:
      app: test
  template:
    metadata:
      labels:
        app: test
    spec:
      containers:
      - name: golang
        image: golang:latest@sha256:golang
        ports:
        - containerPort: 80
      - name: python
        image: python:latest@sha256:python
        ports:
        - containerPort: 81
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: test
  name: test
spec:
  replicas: 3
  selector:
    matchLabels:
      app: test
  template:
    metadata:
      labels:
        app: test
    spec:
      containers:
      - name: redis
        image: redis:latest@sha256:redis
        ports:
        - containerPort: 80
      - image: bash:latest@sha256:bash
        name: bash
`),
				[]byte(`apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: test
  name: test
spec:
  replicas: 3
  selector:
    matchLabels:
      app: test
  template:
    metadata:
      labels:
        app: test
    spec:
      containers:
      - name: busybox
        image: busybox:latest@sha256:busybox
        ports:
        - containerPort: 80
      - name: java
        image: java:latest@sha256:java
        ports:
        - containerPort: 81
---
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: test
  name: test
spec:
  replicas: 3
  selector:
    matchLabels:
      app: test
  template:
    metadata:
      labels:
        app: test
    spec:
      containers:
      - name: alpine
        image: alpine:latest@sha256:alpine
        ports:
        - containerPort: 80
      - image: ruby:latest@sha256:ruby
        name: ruby
`),
			},
		},
		{
			Name: "Exclude Tags",
			Contents: [][]byte{
				[]byte(`apiVersion: v1
kind: Pod
metadata:
  name: test
  labels:
    app: test
spec:
  containers:
  - name: busybox
    image: busybox
    ports:
    - containerPort: 80
`),
			},
			PathImages: map[string][]interface{}{
				"pod.yaml": {
					map[string]interface{}{
						"name":      "busybox",
						"tag":       "latest",
						"digest":    "busybox",
						"container": "busybox",
					},
				},
			},
			Expected: [][]byte{
				[]byte(`apiVersion: v1
kind: Pod
metadata:
  name: test
  labels:
    app: test
spec:
  containers:
  - name: busybox
    image: busybox@sha256:busybox
    ports:
    - containerPort: 80
`),
			},
			ExcludeTags: true,
		},
		{
			Name: "Fewer Images In Kubernetesfile",
			Contents: [][]byte{
				[]byte(`apiVersion: v1
kind: Pod
metadata:
  name: test
  labels:
    app: test
spec:
  containers:
  - name: busybox
    image: busybox
    ports:
    - containerPort: 80
  - name: golang
    image: golang
    ports:
    - containerPort: 88
`),
			},
			PathImages: map[string][]interface{}{
				"pod.yaml": {
					map[string]interface{}{
						"name":      "busybox",
						"tag":       "latest",
						"digest":    "busybox",
						"container": "busybox",
					},
					map[string]interface{}{
						"name":      "golang",
						"tag":       "latest",
						"digest":    "golang",
						"container": "golang",
					},
					map[string]interface{}{
						"name":      "extra",
						"tag":       "latest",
						"digest":    "extra",
						"container": "extra",
					},
				},
			},
			ShouldFail: true,
		},
		{
			Name: "More Images In Kubernetesfile",
			Contents: [][]byte{
				[]byte(`apiVersion: v1
kind: Pod
metadata:
  name: test
  labels:
    app: test
spec:
  containers:
  - name: busybox
    image: busybox
    ports:
    - containerPort: 80
  - name: golang
    image: golang
    ports:
    - containerPort: 88
`),
			},
			PathImages: map[string][]interface{}{
				"pod.yml": {
					map[string]interface{}{
						"name":      "busybox",
						"tag":       "latest",
						"digest":    "busybox",
						"container": "busybox",
					},
				},
			},
			ShouldFail: true,
		},
	}

	for _, test := range tests { // nolint: dupl
		test := test

		t.Run(test.Name, func(t *testing.T) {
			t.Parallel()

			tempDir := testutils.MakeTempDirInCurrentDir(t)
			defer os.RemoveAll(tempDir)

			var pathsToWrite []string

			tempPathImages := map[string][]interface{}{}

			for path, images := range test.PathImages {
				pathsToWrite = append(pathsToWrite, path)

				path = filepath.Join(tempDir, path)
				tempPathImages[path] = images
			}

			sort.Strings(pathsToWrite)

			testutils.WriteFilesToTempDir(
				t, tempDir, pathsToWrite, test.Contents,
			)

			writer := write.NewKubernetesfileWriter(test.ExcludeTags)

			done := make(chan struct{})
			defer close(done)

			writtenPathResults := writer.WriteFiles(
				tempPathImages, tempDir, done,
			)

			var got []string

			var err error

			for writtenPath := range writtenPathResults {
				if writtenPath.Err() != nil {
					err = writtenPath.Err()
				}
				got = append(got, writtenPath.NewPath())
			}

			if test.ShouldFail {
				if err == nil {
					t.Fatal("expected error but did not get one")
				}

				return
			}

			if err != nil {
				t.Fatal(err)
			}

			sort.Strings(got)

			testutils.AssertWrittenFilesEqual(t, test.Expected, got)
		})
	}
}
