package registry

import "strings"

type WrapperManager struct {
	defaultWrapper Wrapper
	wrappers       []Wrapper
}

func NewWrapperManager(defaultWrapper Wrapper) *WrapperManager {
	return &WrapperManager{defaultWrapper: defaultWrapper}
}

func (m *WrapperManager) Add(wrappers ...Wrapper) {
	m.wrappers = append(m.wrappers, wrappers...)
}

func (m *WrapperManager) GetWrapper(imageName string) Wrapper {
	for _, wrapper := range m.wrappers {
		if strings.Contains(imageName, wrapper.Prefix()) {
			return wrapper
		}
	}
	return m.defaultWrapper
}
