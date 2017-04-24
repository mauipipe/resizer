package main

import (
	"testing"
)

func Test_GetImageSize(t *testing.T) {
	config := new(Configuration)
	config.Placeholders = make([]Placeholder, 1)
	config.Placeholders[0] = Placeholder{"test", &Size{100, 102}}

	size := GetImageSize("1000,23", config)

	if size.Width != 1000 || size.Height != 23 {
		t.Errorf("With or height is wrong")
	}

	newSize := GetImageSize("web", config)
	if newSize.Width != 0 || newSize.Height != 0 {
		t.Errorf("With or Height are wrong with a missing placeholder")
	}

	placeholderSize := GetImageSize("test", config)
	if placeholderSize.Width != 100 || placeholderSize.Height != 102 {
		t.Errorf("With or Height are wrong with a missing placeholder")
	}
}

func Test_ParseInteger(t *testing.T) {
	value, _ := parseInteger("4")

	if value != 4 {
		t.Errorf("Value isn't 4")
	}

	newValue, err := parseInteger("this isnt a number")

	if err == nil {
		t.Errorf("We were  expecting an error")
	}

	if newValue != 0 {
		t.Errorf("Given a string value should be 0")
	}
}
