package main

type Sizer struct {
	requestedSize *Size
}

// Given the requested
func (s *Sizer) calculateAspectRatio(imageWidth int, imageHeight int) *Size {
	size := new(Size)

	ratio := float32(imageWidth) / float32(imageHeight)
	width := uint(s.requestedSize.Width)
	height := float32(width) * ratio

	if uint(height) > size.Height {
		height = float32(s.requestedSize.Height)
		width = uint(float32(height) / ratio)
	}

	size.Height = uint(height)
	size.Width = width

	return size
}
