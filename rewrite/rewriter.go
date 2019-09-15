package rewrite

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"sync"

	"github.com/michaelperel/docker-lock/generate"
	"github.com/spf13/cobra"
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

func NewRewriter(cmd *cobra.Command) (*Rewriter, error) {
	outfile, err := cmd.Flags().GetString("outfile")
	if err != nil {
		return nil, err
	}
	lByt, err := ioutil.ReadFile(outfile)
	if err != nil {
		return nil, err
	}
	var lockfile generate.Lockfile
	if err := json.Unmarshal(lByt, &lockfile); err != nil {
		return nil, err
	}
	suffix, err := cmd.Flags().GetString("suffix")
	if err != nil {
		return nil, err
	}
	return &Rewriter{Lockfile: lockfile, Suffix: suffix}, nil
}

// Rewrite rewrites base images, in the following order: Dockerfiles, Dockerfiles referenced by Composefiles, Composefiles, to include digests.
func (r *Rewriter) Rewrite() error {
	var dwg sync.WaitGroup
	dErr := make(chan error)
	for dpath, images := range r.DockerfileImages {
		dwg.Add(1)
		go r.rewriteDockerfile(dpath, images, dErr, &dwg)
	}
	go func() {
		dwg.Wait()
		close(dErr)
	}()
	for err := range dErr {
		if err != nil {
			return err
		}
	}
	var cwg sync.WaitGroup
	cErr := make(chan error)
	for cpath, images := range r.ComposefileImages {
		cwg.Add(1)
		go r.rewriteComposefiles(cpath, images, cErr, &cwg)
	}
	go func() {
		cwg.Wait()
		close(cErr)
	}()
	for err := range cErr {
		if err != nil {
			return err
		}
	}
	return nil
}

// rewriteDockerfile requires images to be passed in in the order that they should be replaced.
func (r *Rewriter) rewriteDockerfile(dpath string,
	images []generate.DockerfileImage,
	res chan<- error,
	wg *sync.WaitGroup) {
	if wg != nil {
		defer wg.Done()
	}
	dfile, err := ioutil.ReadFile(dpath)
	if err != nil {
		res <- err
		return
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
				if imageIndex > len(images) {
					res <- fmt.Errorf("more images exist in %s than in the Lockfile", dpath)
					return
				}
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
	if imageIndex != len(images) {
		res <- fmt.Errorf("more images exist in the Lockfile than in %s", dpath)
		return
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
		res <- err
		return
	}
	res <- nil
}

// rewriteComposefiles requires images to be passed in in the order that they should be replaced.
func (r *Rewriter) rewriteComposefiles(cpath string,
	images []generate.ComposefileImage,
	res chan<- error,
	wg *sync.WaitGroup) {
	if wg != nil {
		defer wg.Done()
	}
	cByt, err := ioutil.ReadFile(cpath)
	if err != nil {
		res <- err
		return
	}
	var comp compose
	if err := yaml.Unmarshal(cByt, &comp); err != nil {
		res <- err
		return
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
		var shouldRewriteDockerfile, shouldRewriteImageline bool
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
			dErr := make(chan error)
			go r.rewriteDockerfile(dFile, dImages, dErr, nil)
			if err := <-dErr; err != nil {
				res <- fmt.Errorf("%s from %s", err, cpath)
				return
			}
		}
	}
	if len(rewrittenImageLines) != 0 {
		var rewrittenCFile map[string]interface{}
		if err := yaml.Unmarshal(cByt, &rewrittenCFile); err != nil {
			res <- err
			return
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
			res <- err
			return
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
			res <- err
			return
		}
	}
	res <- nil
}

func getDImageInfo(serviceName string, sImages map[string][]generate.ComposefileImage) (string, []generate.DockerfileImage) {
	dImages := make([]generate.DockerfileImage, len(sImages[serviceName]))
	dFile := sImages[serviceName][0].Dockerfile
	for i, cImage := range sImages[serviceName] {
		dImages[i] = generate.DockerfileImage{Image: cImage.Image}
	}
	return dFile, dImages
}
