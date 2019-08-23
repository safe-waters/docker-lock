package rewrite

import (
	"encoding/json"
	"fmt"
	"github.com/michaelperel/docker-lock/generate"
	"io/ioutil"
	"strings"
	"sync"
)

type Rewriter struct {
	generate.Lockfile
	Postfix string
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
	var wg sync.WaitGroup
	for dpath, images := range r.DockerfileImages {
		wg.Add(1)
		go r.rewriteDockerfile(dpath, images, &wg)
	}
	wg.Wait()
	r.rewriteComposefiles()
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

func (r *Rewriter) rewriteComposefiles() {

}
