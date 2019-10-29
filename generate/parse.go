package generate

import (
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/michaelperel/docker-lock/generate/internal/parse"
	"gopkg.in/yaml.v2"
)

type parsedImageLine struct {
	line            string
	dockerfileName  string
	composefileName string
	position        int
	serviceName     string
	err             error
}

func parseComposefile(fileName string, parsedImageLines chan<- parsedImageLine) {
	yamlByt, err := ioutil.ReadFile(fileName)
	if err != nil {
		extraErrorInfo := fmt.Errorf("%s From compose-file: '%s'.", err, fileName)
		parsedImageLines <- parsedImageLine{err: extraErrorInfo}
		return
	}
	var comp parse.Compose
	if err := yaml.Unmarshal(yamlByt, &comp); err != nil {
		extraErrInfo := fmt.Errorf("%s From compose-file: '%s'.", err, fileName)
		parsedImageLines <- parsedImageLine{err: extraErrInfo}
		return
	}
	var wg sync.WaitGroup
	for serviceName, service := range comp.Services {
		wg.Add(1)
		go func(serviceName string, service parse.Service) {
			defer wg.Done()
			if service.BuildWrapper == nil {
				line := os.ExpandEnv(service.ImageName)
				parsedImageLines <- parsedImageLine{line: line, composefileName: fileName, serviceName: serviceName}
				return
			}
			switch build := service.BuildWrapper.Build.(type) {
			case parse.Simple:
				var dockerfile string
				dockerfileDir := os.ExpandEnv(string(build))
				if filepath.IsAbs(dockerfileDir) {
					dockerfile = filepath.Join(dockerfileDir, "Dockerfile")
				} else {
					dockerfile = filepath.Join(filepath.Dir(fileName), dockerfileDir, "Dockerfile")
				}
				parseDockerfileFromComposefile(dockerfile, nil, fileName, serviceName, parsedImageLines)
			case parse.Verbose:
				context := os.ExpandEnv(build.Context)
				if !filepath.IsAbs(context) {
					context = filepath.Join(filepath.Dir(fileName), context)
				}
				dockerfile := os.ExpandEnv(build.Dockerfile)
				if dockerfile == "" {
					dockerfile = filepath.Join(context, "Dockerfile")
				} else {
					dockerfile = filepath.Join(context, dockerfile)
				}
				buildArgs := make(map[string]string)
				for _, arg := range build.Args {
					kv := strings.SplitN(arg, "=", 2)
					buildArgs[os.ExpandEnv(kv[0])] = os.ExpandEnv(kv[1])
				}
				parseDockerfileFromComposefile(dockerfile, buildArgs, fileName, serviceName, parsedImageLines)
			}
		}(serviceName, service)
	}
	wg.Wait()
}

func parseDockerfile(fileName string, buildArgs map[string]string, parsedImageLines chan<- parsedImageLine) {
	parseDockerfileFromComposefile(fileName, buildArgs, "", "", parsedImageLines)
}

func parseDockerfileFromComposefile(dockerfileName string, buildArgs map[string]string, composefileName string, serviceName string, parsedImageLines chan<- parsedImageLine) {
	dockerfile, err := os.Open(dockerfileName)
	if err != nil {
		extraErrInfo := fmt.Sprintf("%s From dockerfile: '%s'.", err, dockerfileName)
		if composefileName != "" {
			extraErrInfo += fmt.Sprintf(" From service: '%s' in compose-file: '%s'.", serviceName, composefileName)
		}
		parsedImageLines <- parsedImageLine{err: errors.New(extraErrInfo)}
		return
	}
	defer dockerfile.Close()
	stageNames := make(map[string]bool)
	globalContext := true
	globalArgs := make(map[string]string)
	scanner := bufio.NewScanner(dockerfile)
	scanner.Split(bufio.ScanLines)
	position := 0
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) > 0 {
			switch instruction := strings.ToLower(fields[0]); instruction {
			case "arg":
				if globalContext {
					if strings.Contains(fields[1], "=") {
						//ARG VAR=VAL
						splitPair := strings.SplitN(fields[1], "=", 2)
						globalArgs[stripQuotesFromArgInstruction(splitPair[0])] = stripQuotesFromArgInstruction(splitPair[1])
					} else {
						// ARG VAR1
						globalArgs[fields[1]] = ""
					}
				}
			case "from":
				globalContext = false
				line := expandField(fields[1], globalArgs, buildArgs)
				if !stageNames[line] {
					parsedImageLines <- parsedImageLine{line: line,
						dockerfileName:  dockerfileName,
						composefileName: composefileName,
						serviceName:     serviceName,
						position:        position}
					position++
				}
				// FROM <image> AS <stage>
				// FROM <stage> AS <another stage>
				if len(fields) == 4 {
					stageName := expandField(fields[3], globalArgs, buildArgs)
					stageNames[stageName] = true
				}
			}
		}
	}
}

func expandField(field string, globalArgs map[string]string, buildArgs map[string]string) string {
	mapper := func(arg string) string {
		var val string
		globalVal, ok := globalArgs[arg]
		if !ok {
			return ""
		}
		buildArg, ok := buildArgs[arg]
		if ok {
			val = buildArg
		} else {
			val = globalVal
		}
		return val
	}
	return os.Expand(field, mapper)
}

func stripQuotesFromArgInstruction(s string) string {
	// Valid in a Dockerfile - Any number of quotes as long as there is one on either side
	// ARG "IMAGE"="busybox"
	// ARG "IMAGE"""""="busybox"""""""""""""
	if s[0] == '"' && s[len(s)-1] == '"' {
		s = strings.TrimRight(strings.TrimLeft(s, "\""), "\"")
	}
	return s
}
