package main

import (
	"fmt"
	"github.com/hellofresh/resizer/Godeps/_workspace/src/github.com/nfnt/resize"
	"github.com/hellofresh/resizer/Godeps/_workspace/src/github.com/spf13/viper"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type Configuration struct {
	Port          uint
	HostWhiteList []string
	Size          Size
	Placeholders []Placeholder
}

type Placeholder struct {
	Name string
	Size *Size
}

type Size struct {
	Width  uint
	Height uint
}

var config *Configuration

// Return a given error in JSON format to the ResponseWriter
func formatError(err error, w http.ResponseWriter) {
	http.Error(w, fmt.Sprintf("{ \"error\": \"%s\"}", err), 400)
}

// Parse a given string into a uint value
func parseInteger(value string) (uint, error) {
	integer, err := strconv.Atoi(value)
	return uint(integer), err
}

func GetImageSize(imageSize string, config *Configuration) *Size {
	size := new(Size)

	for _, placeholder := range config.Placeholders {
		if placeholder.Name == imageSize {
			return placeholder.Size
		}
	}

	// If we didn't found the placeholder then we split the size
	parts := strings.Split(imageSize, "x")
	if (len(parts) == 2) {
		size.Width, _ = parseInteger(parts[0])
		size.Height, _ = parseInteger(parts[1])
	}

	return size
}

// Resizing endpoint.
func resizing(w http.ResponseWriter, r *http.Request) {
	// Get parameters
	imageUrl := r.FormValue("image")

	size := GetImageSize(r.FormValue("size"), config)

	validator := Validator{config}

	if err := validator.CheckHostInWhiteList(imageUrl); err != nil {
		formatError(err, w)
		return
	}

	if err := validator.CheckRequestNewSize(size); err != nil {
		formatError(err, w)
		return
	}

	// Download the image
	imageBuffer, err := http.Get(imageUrl)

	if err != nil {
		formatError(err, w)
		return
	}

	finalImage, _, _ := image.Decode(imageBuffer.Body)
	r.Body.Close()

	imageResized := resize.Resize(size.Width, size.Height, finalImage, resize.Lanczos3)

	contentType := imageBuffer.Header.Get("Content-Type")
	switch contentType {
	case "image/png":
		png.Encode(w, imageResized)
		log.Printf("Successfully handled content type '%s'\n", contentType)
	case "image/jpeg":
		jpeg.Encode(w, imageResized, nil)
		log.Printf("Successfully handled content type '%s'\n", contentType)
	case "binary/octet-stream":
		jpeg.Encode(w, imageResized, nil)
		log.Printf("Successfully handled content type '%s'\n", contentType)
	default:
		log.Printf("Cannot handle content type '%s'\n", contentType)
	}
}

func main() {
	// Load configuration
	viper.SetConfigName("config")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("Fatal error loading configuration file: %s", err))
	}

	// Marshal the configuration into our Struct
	viper.Unmarshal(&config)

	http.HandleFunc("/resize", resizing)
	http.ListenAndServe(fmt.Sprintf(":%d", config.Port), nil)
}
