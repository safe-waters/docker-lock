package parse

// Image contains information extracted from image lines such as
// busybox:latest@sha256:dd97a3f... which could be represented as:
// Image{Name: busybox, Tag: latest, Digest: dd97a3f...}.
type Image struct {
	Name   string `json:"name"`
	Tag    string `json:"tag"`
	Digest string `json:"digest"`
}
