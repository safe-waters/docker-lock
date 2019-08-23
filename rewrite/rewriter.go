package rewrite

import (
	"encoding/json"
	"fmt"
	"github.com/michaelperel/docker-lock/generate"
	"io/ioutil"
	"strings"
	"sync"

	"gopkg.in/yaml.v2"
)

type Rewriter struct {
	generate.Lockfile
	Postfix string
}

type compose struct {
	Services map[string]struct {
		Image string      `yaml:"image"`
		Build interface{} `yaml:"build"`
	} `yaml:"services"`
}

func NewRewriter(flags *Flags) (*Rewriter, error) {
	lByt, err := ioutil.ReadFile(flags.Outfile)
	if err != nil {
		return nil, err
	}
	var lockfile generate.Lockfile
	if err := json.Unmarshal(lByt, &lockfile); err != nil {
		return nil, err
	}
	return &Rewriter{Lockfile: lockfile, Postfix: flags.Postfix}, nil
}

// Rewrite rewrites base images to include their digests.
// The order of rewrite is: Dockerfiles, Composefiles, Dockerfiles referenced by Composefiles.
func (r *Rewriter) Rewrite() {
	// var dwg sync.WaitGroup
	// for dpath, images := range r.DockerfileImages {
	// 	dwg.Add(1)
	// 	go r.rewriteDockerfile(dpath, images, &dwg)
	// }
	// dwg.Wait()

    var cwg sync.WaitGroup
	for cpath, images := range r.ComposefileImages {
		cwg.Add(1)
		r.rewriteComposefiles(cpath, images, &cwg)
	}
	cwg.Wait()
}

func (r *Rewriter) rewriteDockerfile(dpath string, images []generate.DockerfileImage, wg *sync.WaitGroup) error {
	if wg != nil {
		defer wg.Done()
	}
	dfile, err := ioutil.ReadFile(dpath)
	if err != nil {
		return err
	}
	stageNames := make(map[string]bool)
	lines := strings.Split(string(dfile), "\n")
	imageIndex := 0
	for i, line := range lines {
		fields := strings.Fields(line)
		if len(fields) > 0 && strings.ToLower(fields[0]) == "from" {
			// FROM <image>
			// FROM <image> AS <stage>
			// FROM <stage> AS <another stage>
			if !stageNames[fields[1]] {
				fields[1] = fmt.Sprintf("%s:%s@sha256:%s", images[imageIndex].Name, images[imageIndex].Tag, images[imageIndex].Digest)
				imageIndex++
			}
			if len(fields) == 4 {
				stageName := fields[3]
				stageNames[stageName] = true
			}
			lines[i] = strings.Join(fields, " ")
		}
	}
	// write lines
	outlines := strings.Join(lines, "\n")
	outpath := dpath + r.Postfix
	if err := ioutil.WriteFile(outpath, []byte(outlines), 0644); err != nil {
		return err
	}
	return nil
}

func (r *Rewriter) rewriteComposefiles(cpath string, images []generate.ComposefileImage, wg *sync.WaitGroup) error {
	if wg != nil {
		defer wg.Done()
	}
	cByt, err := ioutil.ReadFile(cpath)
	if err != nil {
		return err
	}
	var comp compose
	if err := yaml.Unmarshal(cByt, &comp); err != nil {
		return err
	}
	// TODO: change docker-lock file so this step is unnecessary
	lServices := make(map[string][]generate.ComposefileImage)
	for _, image := range images {
		if _, ok := lServices[image.ServiceName]; !ok {
			lServices[image.ServiceName] = make([]generate.ComposefileImage, 0)
		}
		lServices[image.ServiceName] = append(lServices[image.ServiceName], image)
	}
	for serviceName, service := range comp.Services {
		switch build := service.Build.(type) {
		case map[interface{}]interface{}:
			if build["context"] != nil || build["dockerfile"] != nil {
				dFile, dImages := getDImageInfo(serviceName, lServices)
				r.rewriteDockerfile(dFile, dImages, nil)
			} else {
				// record the service name, and the replacement image
			}
		case string:
			dFile, dImages := getDImageInfo(serviceName, lServices)
			r.rewriteDockerfile(dFile, dImages, nil)
		default:
			// record the service name, and the replacement image
			fmt.Println(service, "does not exist")
		}
	}

	// replace the service name with replacement image, rewrite the file.
	return nil
}

func getDImageInfo(serviceName string, lServices map[string][]generate.ComposefileImage) (string, []generate.DockerfileImage) {
	dImages := make([]generate.DockerfileImage, len(lServices[serviceName]))
	dFile := lServices[serviceName][0].Dockerfile
	for i, cImage := range lServices[serviceName] {
		dImages[i] = generate.DockerfileImage{Image: cImage.Image}
	}
	return dFile, dImages
}
