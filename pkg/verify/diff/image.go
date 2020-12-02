package diff

import (
	"fmt"
)

type imageDifferentiator struct{}

func (i *imageDifferentiator) differentiateImage(
	existingImage map[string]interface{},
	newImage map[string]interface{},
	fields []string,
) error {
	var diffField string

	for _, field := range fields {
		if existingImage[field] != newImage[field] {
			diffField = field
			break
		}
	}

	if diffField != "" {
		return fmt.Errorf(
			"existing image with field '%s' "+
				"and value '%v' differs from the new "+
				"image's value '%v'",
			diffField, existingImage[diffField],
			newImage[diffField],
		)
	}

	return nil
}
