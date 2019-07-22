package registry

type Wrapper interface {
	GetDigest(name string, tag string) (string, error)
	Prefix() string
}
