package generate

import (
	"bufio"
	"errors"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path"
	"strings"
	"sync"
)

type parsedImageLine struct {
	line     string
	fileName string
	err      error
}

type compose struct {
	Services map[string]struct {
		ImageName    string        `yaml:"image"`
		BuildWrapper *buildWrapper `yaml:"build"`
	} `yaml:"services"`
}

type verbose struct {
	Context    string   `yaml:"context"`
	Dockerfile string   `yaml:"dockerfile"`
	Args       []string `yaml:"args"`
}

type simple string

type buildWrapper struct {
	Build interface{}
}

func (b *buildWrapper) UnmarshalYAML(unmarshal func(interface{}) error) error {
	*b = buildWrapper{}
	var v verbose
	if err := unmarshal(&v); err == nil {
		b.Build = v
		return nil
	}
	var s simple
	if err := unmarshal(&s); err == nil {
		b.Build = s
		return nil
	}
	return errors.New("Unable to parse service.")
}

func parseComposefile(fileName string, parsedImageLines chan<- parsedImageLine, wg *sync.WaitGroup) {
	if wg != nil {
		defer wg.Done()
	}
	yamlByt, err := ioutil.ReadFile(fileName)
	if err != nil {
		parsedImageLines <- parsedImageLine{fileName: fileName, err: err}
		return
	}
	var comp compose
	if err := yaml.Unmarshal(yamlByt, &comp); err != nil {
		parsedImageLines <- parsedImageLine{fileName: fileName, err: err}
	}
	for _, service := range comp.Services {
		if service.BuildWrapper == nil {
			line := os.ExpandEnv(service.ImageName)
			parsedImageLines <- parsedImageLine{line: line, fileName: fileName}
			continue
		}
		switch build := service.BuildWrapper.Build.(type) {
		case simple:
			dockerfile := path.Join(path.Dir(fileName), os.ExpandEnv(string(build)), "Dockerfile")
			parseDockerfile(dockerfile, nil, parsedImageLines, nil)
		case verbose:
			context := path.Join(path.Dir(fileName), os.ExpandEnv(build.Context))
			dockerfile := os.ExpandEnv(build.Dockerfile)
			if dockerfile == "" {
				dockerfile = path.Join(context, "Dockerfile")
			} else {
				dockerfile = path.Join(context, dockerfile)
			}
			buildArgs := make(map[string]string)
			for _, arg := range build.Args {
				kv := strings.Split(os.ExpandEnv(arg), "=")
				buildArgs[kv[0]] = kv[1]
			}
			parseDockerfile(dockerfile, buildArgs, parsedImageLines, nil)
		}
	}
}

func parseDockerfile(fileName string, composeArgs map[string]string, parsedImageLines chan<- parsedImageLine, wg *sync.WaitGroup) {
	if wg != nil {
		defer wg.Done()
	}
	dockerfile, err := os.Open(fileName)
	if err != nil {
		parsedImageLines <- parsedImageLine{fileName: fileName, err: err}
		return
	}
	defer dockerfile.Close()
	stageNames := make(map[string]bool)
	globalArgs := make(map[string]string)
	scanner := bufio.NewScanner(dockerfile)
	scanner.Split(bufio.ScanLines)
	globalContext := true
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) > 0 {
			switch instruction := strings.ToLower(fields[0]); instruction {
			case "arg":
				if globalContext {
					if strings.Contains(fields[1], "=") {
						//ARG VAR1=VAL1 VAR2=VAL2
						for _, pair := range fields[1:] {
							splitPair := strings.Split(pair, "=")
							globalArgs[splitPair[0]] = splitPair[1]
						}
					} else {
						// ARG VAR1
						globalArgs[fields[1]] = ""
					}
				}
			case "from":
				globalContext = false
				line := expandField(fields[1], globalArgs, composeArgs)
				if !stageNames[line] {
					parsedImageLines <- parsedImageLine{line: line, fileName: fileName}
				}
				// FROM <image> AS <stage>
				// FROM <stage> AS <another stage>
				if len(fields) == 4 {
					stageName := expandField(fields[3], globalArgs, composeArgs)
					stageNames[stageName] = true
				}
			}
		}
	}
}

func expandField(field string, globalArgs map[string]string, composeArgs map[string]string) string {
	mapper := func(arg string) string {
		var val string
		globalVal, ok := globalArgs[arg]
		if !ok {
			return ""
		}
		composeVal, ok := composeArgs[arg]
		if ok {
			val = composeVal
		} else {
			val = globalVal
		}
		// Remove excess quotes, for instance ARG="val" should be equivalent to ARG=val
		if len(val) > 0 && val[0] == '"' {
			val = val[1:]
		}
		if len(val) > 0 && val[len(val)-1] == '"' {
			val = val[:len(val)-1]
		}
		return val
	}
	return os.Expand(field, mapper)
}
