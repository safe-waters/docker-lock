package rewrite

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
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

type rewriteInfo struct {
	outPath          string
	outPathExists    bool
	originalContent  []byte
	rewrittenContent []byte
	err              error
}

type renameInfo struct {
	tmpFilePath     string
	outPathExists   bool
	originalContent []byte
}

type compose struct {
	Services map[string]struct {
		Image string      `yaml:"image"`
		Build interface{} `yaml:"build"`
	} `yaml:"services"`
}

func NewRewriter(cmd *cobra.Command) (*Rewriter, error) {
	outPath, err := cmd.Flags().GetString("outpath")
	if err != nil {
		return nil, err
	}
	lByt, err := ioutil.ReadFile(outPath)
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

// Rewrite rewrites base images referenced in the Lockfile to include digests.
// The order of rewriting is: Dockerfiles, Dockerfiles referenced by
// docker-compose files, and lastly, docker-compose files.
// Rewrite has "transaction"-like properties. All files are first
// written to temporary files. If all writes succeed, each temporary
// file is renamed to the original file's name (+ suffix, if one is provided).
// If a problem occurs while renaming any file, previously existing files will be reverted
// to their original content and new files will be deleted. All temporary files are written
// to a temporary directory that is removed when the function completes.
func (r *Rewriter) Rewrite() (err error) {
	var dwg sync.WaitGroup
	dRwInfo := make(chan rewriteInfo)
	for dPath, images := range r.DockerfileImages {
		dwg.Add(1)
		go r.getDockerfileRewriteInfo(dPath, images, dRwInfo, &dwg)
	}
	go func() {
		dwg.Wait()
		close(dRwInfo)
	}()
	outPathRnInfo := make(map[string]renameInfo)
	tmpDirPath, err := ioutil.TempDir("", "docker-lock-tmp")
	if err != nil {
		return err
	}
	defer func() {
		if rmErr := os.RemoveAll(tmpDirPath); rmErr != nil {
			err = fmt.Errorf("%s Failed to remove temp dir %s with err: %s.", err, tmpDirPath, rmErr)
		}
	}()
	for rwInfo := range dRwInfo {
		if rwInfo.err != nil {
			err = rwInfo.err
			return err
		}
		tmpFilePath, err := writeToTemp(rwInfo.rewrittenContent, tmpDirPath)
		if err != nil {
			return err
		}
		outPathRnInfo[rwInfo.outPath] = renameInfo{tmpFilePath: tmpFilePath, outPathExists: rwInfo.outPathExists, originalContent: rwInfo.originalContent}
	}
	var cwg sync.WaitGroup
	cRwInfo := make(chan rewriteInfo)
	for cPath, images := range r.ComposefileImages {
		cwg.Add(1)
		go r.getComposefileRewriteInfo(cPath, images, cRwInfo, &cwg)
	}
	go func() {
		cwg.Wait()
		close(cRwInfo)
	}()
	for rwInfo := range cRwInfo {
		if rwInfo.err != nil {
			err = rwInfo.err
			return err
		}
		tmpFilePath, err := writeToTemp(rwInfo.rewrittenContent, tmpDirPath)
		if err != nil {
			return err
		}
		outPathRnInfo[rwInfo.outPath] = renameInfo{tmpFilePath: tmpFilePath, outPathExists: rwInfo.outPathExists, originalContent: rwInfo.originalContent}
	}
	rnOutPaths := make(map[string]struct{})
	for outPath, rnInfo := range outPathRnInfo {
		if err = os.Rename(rnInfo.tmpFilePath, outPath); err != nil {
			err = fmt.Errorf("Error renaming tmp file: %s.", err)
			var failedRevertOutPaths []string
			for outPath := range rnOutPaths {
				if outPathRnInfo[outPath].outPathExists {
					if rwErr := ioutil.WriteFile(outPath, outPathRnInfo[outPath].originalContent, 0644); rwErr != nil {
						failedRevertOutPaths = append(failedRevertOutPaths, outPath)
					}
				} else {
					if rmErr := os.Remove(outPath); rmErr != nil {
						failedRevertOutPaths = append(failedRevertOutPaths, outPath)
					}
				}
			}
			if len(failedRevertOutPaths) != 0 {
				err = fmt.Errorf("%s Failed to revert %s", err, failedRevertOutPaths)
			}
			return err
		}
		rnOutPaths[outPath] = struct{}{}
	}
	return nil
}

// getDockerfileRewriteInfo requires images to be passed in in the order that they should be replaced.
func (r *Rewriter) getDockerfileRewriteInfo(dPath string, images []generate.DockerfileImage, rwInfo chan<- rewriteInfo, wg *sync.WaitGroup) {
	if wg != nil {
		defer wg.Done()
	}
	dByt, err := ioutil.ReadFile(dPath)
	if err != nil {
		rwInfo <- rewriteInfo{err: err}
		return
	}
	stageNames := make(map[string]bool)
	lines := strings.Split(string(dByt), "\n")
	imageIndex := 0
	for i, line := range lines {
		fields := strings.Fields(line)
		if len(fields) > 0 && strings.ToLower(fields[0]) == "from" {
			// FROM <image>
			// FROM <image> AS <stage>
			// FROM <stage> AS <another stage>
			if !stageNames[fields[1]] {
				if imageIndex > len(images) {
					rwInfo <- rewriteInfo{err: fmt.Errorf("More images exist in %s than in the Lockfile.", dPath)}
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
		rwInfo <- rewriteInfo{err: fmt.Errorf("More images exist in the Lockfile than in %s.", dPath)}
		return
	}
	rwContent := strings.Join(lines, "\n")
	var outPath string
	if r.Suffix == "" {
		outPath = dPath
		rwInfo <- rewriteInfo{outPath: outPath, outPathExists: true, originalContent: dByt, rewrittenContent: []byte(rwContent)}
	} else {
		outPath = fmt.Sprintf("%s-%s", dPath, r.Suffix)
		var origByt []byte
		var outPathExists bool
		if _, err := os.Stat(outPath); err == nil {
			outPathExists = true
			origByt, err = ioutil.ReadFile(outPath)
			if err != nil {
				rwInfo <- rewriteInfo{err: err}
				return
			}
		}
		rwInfo <- rewriteInfo{outPath: outPath, outPathExists: outPathExists, originalContent: origByt, rewrittenContent: []byte(rwContent)}
	}
}

func (r *Rewriter) getComposefileRewriteInfo(cPath string, images []generate.ComposefileImage, rwInfo chan<- rewriteInfo, wg *sync.WaitGroup) {
	if wg != nil {
		defer wg.Done()
	}
	cByt, err := ioutil.ReadFile(cPath)
	if err != nil {
		rwInfo <- rewriteInfo{err: err}
		return
	}
	var comp compose
	if err := yaml.Unmarshal(cByt, &comp); err != nil {
		rwInfo <- rewriteInfo{err: err}
		return
	}
	sImages := make(map[string][]generate.ComposefileImage)
	for _, image := range images {
		sImages[image.ServiceName] = append(sImages[image.ServiceName], image)
	}
	rwImageLines := map[string]string{}
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
			rwImageLines[serviceName] = fmt.Sprintf("%s:%s@sha256:%s", image.Name, image.Tag, image.Digest)
		} else if shouldRewriteDockerfile {
			dPath, dImages := getDImageInfo(serviceName, sImages)
			dRwInfo := make(chan rewriteInfo)
			go r.getDockerfileRewriteInfo(dPath, dImages, dRwInfo, nil)
			dRes := <-dRwInfo
			if dRes.err != nil {
				rwInfo <- rewriteInfo{err: fmt.Errorf("%s from %s", dRes.err, cPath)}
				return
			}
			rwInfo <- dRes
		}
	}
	if len(rwImageLines) != 0 {
		var rwCFile map[string]interface{}
		if err := yaml.Unmarshal(cByt, &rwCFile); err != nil {
			rwInfo <- rewriteInfo{err: err}
			return
		}
		services := rwCFile["services"].(map[interface{}]interface{})
		for serviceName, serviceSpec := range services {
			serviceName := serviceName.(string)
			if rwImageLines[serviceName] != "" {
				serviceSpec := serviceSpec.(map[interface{}]interface{})
				serviceSpec["image"] = rwImageLines[serviceName]
			}
		}
		outByt, err := yaml.Marshal(&rwCFile)
		if err != nil {
			rwInfo <- rewriteInfo{err: err}
			return
		}
		var outPath string
		if r.Suffix == "" {
			outPath = cPath
			rwInfo <- rewriteInfo{outPath: outPath, outPathExists: true, originalContent: cByt, rewrittenContent: outByt}
		} else {
			var ymlSuffix string
			if strings.HasSuffix(cPath, ".yml") {
				ymlSuffix = ".yml"
			} else if strings.HasSuffix(cPath, ".yaml") {
				ymlSuffix = ".yaml"
			}
			outPath = fmt.Sprintf("%s-%s%s", cPath[:len(cPath)-len(ymlSuffix)], r.Suffix, ymlSuffix)
			var origByt []byte
			var outPathExists bool
			if _, err := os.Stat(outPath); err == nil {
				outPathExists = true
				origByt, err = ioutil.ReadFile(outPath)
				if err != nil {
					rwInfo <- rewriteInfo{err: err}
					return
				}
			}
			rwInfo <- rewriteInfo{outPath: outPath, outPathExists: outPathExists, originalContent: origByt, rewrittenContent: outByt}
		}
	}
}

func writeToTemp(content []byte, tempDir string) (string, error) {
	// writes bytes to temporary file, returning the name of the temp file
	file, err := ioutil.TempFile(tempDir, "docker-lock-")
	defer file.Close()
	if err != nil {
		return "", err
	}
	if _, err := file.Write(content); err != nil {
		return "", err
	}
	return file.Name(), nil
}

func getDImageInfo(serviceName string, sImages map[string][]generate.ComposefileImage) (string, []generate.DockerfileImage) {
	dImages := make([]generate.DockerfileImage, len(sImages[serviceName]))
	dPath := sImages[serviceName][0].Dockerfile
	for i, cImage := range sImages[serviceName] {
		dImages[i] = generate.DockerfileImage{Image: cImage.Image}
	}
	return dPath, dImages
}
