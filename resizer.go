package main

import (
	"fmt"
	"github.com/hellofresh/resizer/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/daddye/vips"
	"github.com/hellofresh/resizer/Godeps/_workspace/src/github.com/spf13/viper"
	"image"
	"image/jpeg"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"bytes"
)

type Configuration struct {
	Port          uint
	ImageHost     string
	HostWhiteList []string
	SizeLimits    Size
	Placeholders  []Placeholder
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
	parts := strings.Split(imageSize, ",")
	if len(parts) == 2 {
		size.Width, _ = parseInteger(parts[0])
		size.Height, _ = parseInteger(parts[1])
	}

	return size
}

// Resizing endpoint.
func resizing(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	// Get parameters
	imageUrl := fmt.Sprintf("%s%s", config.ImageHost, params["path"])

	size := GetImageSize(params["size"], config)

	validator := Validator{config}

	//	if err := validator.CheckHostInWhiteList(imageUrl); err != nil {
	//		formatError(err, w)
	//		return
	//	}

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

	defer r.Body.Close()
	log.Printf("Status code [%d]", imageBuffer.StatusCode )
	if imageBuffer.StatusCode != 200 {
		http.NotFound(w, r)
		return
	}

	finalImage, _, _ := image.Decode(imageBuffer.Body)

	// calculate aspect ratio
	if size.Width > 0 && size.Height > 0 {
		b := finalImage.Bounds()
		ratio := float32(b.Max.Y) / float32(b.Max.X)
		width := uint(size.Width)
		height := float32(width) * ratio
		if uint(height) > size.Height {
			height = float32(size.Height)
			width = uint(float32(height) / ratio)
		}

		size.Height = uint(height)
		size.Width = width
	}

	start := time.Now()
	options := vips.Options{
		Width: int(size.Width),
		Height: int(size.Height),
		Crop: false,
		Extend: vips.EXTEND_WHITE,
		Interpolator: vips.BILINEAR,
		Gravity: vips.CENTRE,
		Quality: 95,
	}

	buf := new(bytes.Buffer)
	_ = jpeg.Encode(buf, finalImage, nil)
	imageResized, err := vips.Resize(buf.Bytes(), options)
	if err != nil {
		formatError(err, w)
		return
	}

	elapsed := time.Since(start)
	log.Printf("It took %f", elapsed.Seconds())

	contentType := imageBuffer.Header.Get("Content-Type")
	switch contentType {
	case "image/png":
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Content-Length", strconv.Itoa(len(imageResized)))
		if _, err := w.Write(imageResized); err != nil {
			formatError(err, w)
			return
		}
		log.Printf("Successfully handled content type '%s'\n", contentType)
	case "image/jpeg":
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("Content-Length", strconv.Itoa(len(imageResized)))
		if _, err := w.Write(imageResized); err != nil {
			formatError(err, w)
			return
		}
		log.Printf("Successfully handled content type '%s'\n", contentType)
	case "binary/octet-stream":
		w.Header().Set("Content-Type", "image/jpeg")
		w.Header().Set("Content-Length", strconv.Itoa(len(imageResized)))
		if _, err := w.Write(imageResized); err != nil {
			formatError(err, w)
			return
		}
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

	rtr := mux.NewRouter()
	rtr.HandleFunc("/resize/{size}/{path:(.*)}", resizing).Methods("GET")

	server := &http.Server{
		Addr: fmt.Sprintf(":%d", config.Port),
		Handler: rtr,
		ReadTimeout: 3 * time.Second,
		WriteTimeout: 5 * time.Second,
	}

	server.ListenAndServe()
}
