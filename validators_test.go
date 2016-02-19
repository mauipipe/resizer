package main

import (
	"testing"
)

func Test_CheckHostInWhiteListWithEmptyConfiguration(t *testing.T) {
	config := new(Configuration)
	validator := Validator{config}

	if err := validator.CheckHostInWhiteList("doesnt exists"); err == nil {
		t.Errorf("Missing error returning!")
	}
}

func Test_CheckHostInWhiteListWithSomeHostsInWhieList(t *testing.T) {
	config := new(Configuration)
	config.HostWhiteList = []string{"http://www.google.com", "two hosts"}
	validator := Validator{config}

	// Check for one that doesn't exists
	err := validator.CheckHostInWhiteList("http://www.sergiosola.com/")
	if err == nil {
		t.Errorf("Should return an error!!!")
	}
}

func Test_CheckHostInWhiteListWithValidHost(t *testing.T) {
	config := new(Configuration)
	config.HostWhiteList = []string{"one host", "sergiosola.com"}
	validator := Validator{config}

	err := validator.CheckHostInWhiteList("https://sergiosola.com/images?withParams=dsada")
	if err != nil {
		t.Errorf("Should not to return an error!!!")
	}
}

func Test_CheckHostInWhiteListWithValidPattern(t *testing.T) {
	config := new(Configuration)
	config.HostWhiteList = []string{"www.google.com", "([a-z]+).cdn.google.com"}
	validator := Validator{config}

	err := validator.CheckHostInWhiteList("https://dsadsaasds.cdn.google.com/images?withParams=dsada")
	if err != nil {
		t.Errorf("Should not to return an error!!!")
	}
}

func Test_CheckSizeIsAllowed(t *testing.T) {
	config := new(Configuration)
	config.SizeLimits = Size{1000, 1000}
	validator := Validator{config}

	givenSize := new(Size)
	givenSize.Width = 993
	givenSize.Height = 399

	err := validator.CheckRequestNewSize(givenSize)
	if err != nil {
		t.Errorf("Should not to return an error!!")
	}
}

func Test_CHeckSizeIsNotAllows(t *testing.T) {
	config := new(Configuration)
	config.SizeLimits = Size{1000, 1000}
	validator := Validator{config}

	givenSize := new(Size)
	givenSize.Width = 9999
	givenSize.Height = 399

	err := validator.CheckRequestNewSize(givenSize)
	if err == nil {
		t.Errorf("Should to return an error!!")
	}
}
