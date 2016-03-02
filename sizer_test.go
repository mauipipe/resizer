package main

import "testing"

func giveMeSize(width uint, height uint) *Size {
	requestedSize := new(Size)
	requestedSize.Height = width
	requestedSize.Width = height

	return requestedSize
}

func Test_CheckAspectRatioWithHeightAndWidth(t *testing.T) {
	requestedSize := giveMeSize(290, 500)

	sizer := Sizer{requestedSize}
	aspectRatioSize := sizer.calculateAspectRatio(3180, 2120)

	if (aspectRatioSize.Height != 290 && aspectRatioSize.Width != 193) {
		t.Errorf("Expect width %d and height %d", aspectRatioSize.Width, aspectRatioSize.Height)
	}
}
