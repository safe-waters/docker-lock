package parse

// Image contains information extracted from 'FROM' instructions in Dockerfiles
// or 'image:' keys in docker-compose files. For instance,
// FROM busybox:latest@sha256:dd97a3f...
// could be represented as:
// Image{Name: busybox, Tag: latest, Digest: dd97a3f...}.
type Image struct {
	Name   string `json:"name"`
	Tag    string `json:"tag"`
	Digest string `json:"digest"`
}
