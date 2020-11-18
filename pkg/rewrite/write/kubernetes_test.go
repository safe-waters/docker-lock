package write_test

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/safe-waters/docker-lock/pkg/generate/parse"
	"github.com/safe-waters/docker-lock/pkg/rewrite/write"
)

func TestKubernetesfileWriter(t *testing.T) {
	t.Parallel()

	tests := []struct {
		Name        string
		Contents    [][]byte
		Expected    [][]byte
		PathImages  map[string][]*parse.KubernetesfileImage
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
			PathImages: map[string][]*parse.KubernetesfileImage{
				"pod.yaml": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "latest",
							Digest: "busybox",
						},
						ContainerName: "busybox",
					},
					{
						Image: &parse.Image{
							Name:   "golang",
							Tag:    "latest",
							Digest: "golang",
						},
						ContainerName: "golang",
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
			PathImages: map[string][]*parse.KubernetesfileImage{
				"deployment.yaml": {
					{
						Image: &parse.Image{
							Name:   "golang",
							Tag:    "latest",
							Digest: "golang",
						},
						ContainerName: "golang",
					},
					{
						Image: &parse.Image{
							Name:   "python",
							Tag:    "latest",
							Digest: "python",
						},
						ContainerName: "python",
					},
					{
						Image: &parse.Image{
							Name:   "redis",
							Tag:    "latest",
							Digest: "redis",
						},
						ContainerName: "redis",
					},
					{
						Image: &parse.Image{
							Name:   "bash",
							Tag:    "latest",
							Digest: "bash",
						},
						ContainerName: "bash",
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
			PathImages: map[string][]*parse.KubernetesfileImage{
				"deployment.yaml": {
					{
						Image: &parse.Image{
							Name:   "golang",
							Tag:    "latest",
							Digest: "golang",
						},
						ContainerName: "golang",
					},
					{
						Image: &parse.Image{
							Name:   "python",
							Tag:    "latest",
							Digest: "python",
						},
						ContainerName: "python",
					},
					{
						Image: &parse.Image{
							Name:   "redis",
							Tag:    "latest",
							Digest: "redis",
						},
						ContainerName: "redis",
					},
					{
						Image: &parse.Image{
							Name:   "bash",
							Tag:    "latest",
							Digest: "bash",
						},
						ContainerName: "bash",
					},
				},
				"deployment1.yaml": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "latest",
							Digest: "busybox",
						},
						ContainerName: "busybox",
					},
					{
						Image: &parse.Image{
							Name:   "java",
							Tag:    "latest",
							Digest: "java",
						},
						ContainerName: "java",
					},
					{
						Image: &parse.Image{
							Name:   "alpine",
							Tag:    "latest",
							Digest: "alpine",
						},
						ContainerName: "alpine",
					},
					{
						Image: &parse.Image{
							Name:   "ruby",
							Tag:    "latest",
							Digest: "ruby",
						},
						ContainerName: "ruby",
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
			PathImages: map[string][]*parse.KubernetesfileImage{
				"pod.yaml": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "latest",
							Digest: "busybox",
						},
						ContainerName: "busybox",
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
			PathImages: map[string][]*parse.KubernetesfileImage{
				"pod.yaml": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "latest",
							Digest: "busybox",
						},
					},
					{
						Image: &parse.Image{
							Name:   "golang",
							Tag:    "latest",
							Digest: "golang",
						},
					},
					{
						Image: &parse.Image{
							Name:   "extra",
							Tag:    "latest",
							Digest: "extra",
						},
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
			PathImages: map[string][]*parse.KubernetesfileImage{
				"pod.yaml": {
					{
						Image: &parse.Image{
							Name:   "busybox",
							Tag:    "latest",
							Digest: "busybox",
						},
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

			tempDir := makeTempDirInCurrentDir(t)
			defer os.RemoveAll(tempDir)

			var pathsToWrite []string

			tempPathImages := map[string][]*parse.KubernetesfileImage{}

			for path, images := range test.PathImages {
				pathsToWrite = append(pathsToWrite, path)

				path = filepath.Join(tempDir, path)
				tempPathImages[path] = images
			}

			sort.Strings(pathsToWrite)

			writeFilesToTempDir(
				t, tempDir, pathsToWrite, test.Contents,
			)

			writer := &write.KubernetesfileWriter{
				Directory:   tempDir,
				ExcludeTags: test.ExcludeTags,
			}
			done := make(chan struct{})
			writtenPathResults := writer.WriteFiles(
				tempPathImages, done,
			)

			var got []string

			var err error

			for writtenPath := range writtenPathResults {
				if writtenPath.Err != nil {
					err = writtenPath.Err
				}
				got = append(got, writtenPath.Path)
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

			assertWrittenFiles(t, test.Expected, got)
		})
	}
}
