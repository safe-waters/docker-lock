package rewrite

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"sync"

	"github.com/michaelperel/docker-lock/generate"

	"gopkg.in/yaml.v2"
)

type Rewriter struct {
	generate.Lockfile
	Suffix string
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
	return &Rewriter{Lockfile: lockfile, Suffix: flags.Suffix}, nil
}

// Rewrite rewrites base images to include their digests.
// Rewrites in the following order: Dockerfiles, Dockerfiles referenced by Composefiles, Composefiles.
func (r *Rewriter) Rewrite() {
	var dwg sync.WaitGroup
	for dpath, images := range r.DockerfileImages {
		dwg.Add(1)
		go r.rewriteDockerfile(dpath, images, &dwg)
	}
	dwg.Wait()
	var cwg sync.WaitGroup
	for cpath, images := range r.ComposefileImages {
		cwg.Add(1)
		r.rewriteComposefiles(cpath, images, &cwg)
	}
	cwg.Wait()
}

// rewriteDockerfile requires images to be passed in in the order that they should be replaced.
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
	var outpath string
	if r.Suffix == "" {
		outpath = dpath
	} else {
		outpath = fmt.Sprintf("%s-%s", dpath, r.Suffix)
	}
	if err := ioutil.WriteFile(outpath, []byte(outlines), 0644); err != nil {
		return err
	}
	return nil
}

// rewriteComposefiles requires images to be passed in in the order that they should be replaced.
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
	sImages := make(map[string][]generate.ComposefileImage)
	for _, image := range images {
		if _, ok := sImages[image.ServiceName]; !ok {
			sImages[image.ServiceName] = make([]generate.ComposefileImage, 0)
		}
		sImages[image.ServiceName] = append(sImages[image.ServiceName], image)
	}
	rewrittenImageLines := map[string]string{}
	for serviceName, service := range comp.Services {
		shouldRewriteDockerfile := false
		shouldRewriteImageline := false
		switch build := service.Build.(type) {
		case map[interface{}]interface{}:
			if build["context"] != nil || build["dockerfile"] != nil {
				shouldRewriteDockerfile = true
			} else {
				shouldRewriteImageline = true
			}
		case string:
			shouldRewriteDockerfile = true
		default:
			shouldRewriteImageline = true
		}
		if shouldRewriteImageline {
			image := sImages[serviceName][0]
			rewrittenImageLines[serviceName] = fmt.Sprintf("%s:%s@sha256:%s", image.Name, image.Tag, image.Digest)
		} else if shouldRewriteDockerfile {
			dFile, dImages := getDImageInfo(serviceName, sImages)
			r.rewriteDockerfile(dFile, dImages, nil)
		}
	}
	if len(rewrittenImageLines) != 0 {
		var rewrittenCFile map[string]interface{}
		if err := yaml.Unmarshal(cByt, &rewrittenCFile); err != nil {
			return err
		}
		services := rewrittenCFile["services"].(map[interface{}]interface{})
		for serviceName, serviceSpec := range services {
			serviceName := serviceName.(string)
			if rewrittenImageLines[serviceName] != "" {
				serviceSpec := serviceSpec.(map[interface{}]interface{})
				serviceSpec["image"] = rewrittenImageLines[serviceName]
			}
		}
		outByt, err := yaml.Marshal(&rewrittenCFile)
		if err != nil {
			return err
		}
		var outpath string
		if r.Suffix == "" {
			outpath = cpath
		} else {
			var ymlSuffix string
			if strings.HasSuffix(cpath, ".yml") {
				ymlSuffix = ".yml"
			}
			if strings.HasSuffix(cpath, ".yaml") {
				ymlSuffix = ".yaml"
			}
			outpath = fmt.Sprintf("%s-%s%s", cpath[:len(cpath)-len(ymlSuffix)], r.Suffix, ymlSuffix)
		}
		if err := ioutil.WriteFile(outpath, outByt, 0644); err != nil {
			return err
		}
	}
	return nil
}

func getDImageInfo(serviceName string, sImages map[string][]generate.ComposefileImage) (string, []generate.DockerfileImage) {
	dImages := make([]generate.DockerfileImage, len(sImages[serviceName]))
	dFile := sImages[serviceName][0].Dockerfile
	for i, cImage := range sImages[serviceName] {
		dImages[i] = generate.DockerfileImage{Image: cImage.Image}
	}
	return dFile, dImages
}
