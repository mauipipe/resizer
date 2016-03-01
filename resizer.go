package main

import (
	"bytes"
	"fmt"
	"github.com/hellofresh/resizer/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/hellofresh/resizer/Godeps/_workspace/src/github.com/nfnt/resize"
	"github.com/hellofresh/resizer/Godeps/_workspace/src/github.com/peterbourgon/diskv"
	"github.com/hellofresh/resizer/Godeps/_workspace/src/github.com/spf13/viper"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"net/http"
	"path"
	"runtime"
	"strconv"
	"strings"
	"time"
	"os"
)

const (
	MaxIdleConnections int = 50
	RequestTimeout     int = 5
)

var (
	httpClient *http.Client
	config     *Configuration
	cache      *diskv.Diskv
	cacheStats *CacheStats
)

type Configuration struct {
	Port            uint
	ImageHost       string
	HostWhiteList   []string
	SizeLimits      Size
	Placeholders    []Placeholder
	Warmupsizes     []Size
	Cachethumbnails bool
}

type Placeholder struct {
	Name string
	Size *Size
}

type Size struct {
	Width  uint
	Height uint
}

type CacheStats struct {
	Hits   uint64
	Misses uint64
}

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

func getClient() *http.Client {
	client := &http.Client{
		Timeout: time.Duration(RequestTimeout) * time.Second,
	}

	return client
}

func init() {
	var cachePath string
	cachePath = os.Getenv("RESIZER_CACHE_PATH")
	if cachePath == "" {
		cachePath = "/tmp"
	}

	httpClient = getClient()
	cacheStats = new(CacheStats)
	flatTransform := func(s string) []string { return []string{} }
	cache = diskv.New(diskv.Options{
		BasePath:     cachePath,
		Transform:    flatTransform,
		CacheSizeMax: 1024 * 1024 * 1024,
	})
}

func (self *CacheStats) hit() {
	self.Hits++
}

func (self *CacheStats) miss() {
	self.Misses++
}

func extractIdFromUrl(url string) string {
	i, j := strings.LastIndex(url, "/"), strings.LastIndex(url, path.Ext(url))
	name := url[i+1 : j]

	return name
}

// Resizing endpoint.
func resizing(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	// Get parameters
	imageUrl := fmt.Sprintf("%s%s", config.ImageHost, params["path"])
	size := GetImageSize(params["size"], config)
	validator := Validator{config}

	if err := validator.CheckRequestNewSize(size); err != nil {
		formatError(err, w)
		return
	}

	// Build caching key
	imageId := extractIdFromUrl(imageUrl)
	key := fmt.Sprintf("%d_%d_%s", size.Height, size.Width, imageId)
	log.Printf("Caching key %s", key)

	if config.Cachethumbnails && cache.Has(key) {
		log.Printf("Cached hit!")
		cacheStats.hit()
		cachedImage, _ := cache.ReadStream(key, true)
		finalImage, _, _ := image.Decode(cachedImage)
		jpeg.Encode(w, finalImage, nil)
		return
	} else {
		if config.Cachethumbnails {
			cacheStats.miss()
		}
	}

	// Download the image
	originalImageKey := fmt.Sprintf("original_%s", imageId)

	imageBuffer := new(http.Response)
	var cachedHit bool

	if cache.Has(originalImageKey) {
		cacheStats.hit()
		cachedHit = true
	} else {
		cachedHit = false
		cacheStats.miss()
		log.Printf("Downloading image")
		var err error
		imageBuffer, err = httpClient.Get(imageUrl)

		if err != nil {
			formatError(err, w)
			return
		}

		defer imageBuffer.Body.Close()
	}

	defer r.Body.Close()

	if imageBuffer.StatusCode != 200 && cachedHit == false {
		http.NotFound(w, r)
		return
	}

	log.Printf("Status: %d", imageBuffer.StatusCode)

	var finalImage image.Image
	var err error

	if cachedHit == false {
		finalImage, _, err = image.Decode(imageBuffer.Body)
		if err != nil {
			_ = cache.Erase(originalImageKey)
			_ = cache.Erase(key)
			log.Printf("Error jpeg.decode")
			formatError(err, w)
			return
		}
	} else {
		log.Printf("Get image from cache")
		cachedImage, _ := cache.ReadStream(originalImageKey, true)
		finalImage, _, _ = image.Decode(cachedImage)
	}

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

	imageResized := resize.Resize(size.Width, size.Height, finalImage, resize.NearestNeighbor)
	var contentType string
	if cachedHit {
		contentType = "image/jpeg"
	} else {
		contentType = imageBuffer.Header.Get("Content-Type")
	}

	// store image to cache
	if config.Cachethumbnails {
		buf := new(bytes.Buffer)
		_ = jpeg.Encode(buf, imageResized, nil)
		if err := cache.WriteStream(key, buf, true); err != nil {
			formatError(err, w)
			return
		}
	}

	originalBuf := new(bytes.Buffer)
	_ = jpeg.Encode(originalBuf, finalImage, nil)
	if err := cache.WriteStream(originalImageKey, originalBuf, true); err != nil {
		formatError(err, w)
		return
	}

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

func healthCheck(w http.ResponseWriter, r *http.Request) {
	response := fmt.Sprintf("{\"status\": \"ok\",\"cache\": {\"hits\": %d,\"misses\": %d}}", cacheStats.Hits, cacheStats.Misses)
	fmt.Fprint(w, response)
}

func purgeCache(w http.ResponseWriter, r *http.Request) {
	err := cache.EraseAll()

	if err != nil {
		formatError(err, w)
		return
	}

	fmt.Fprint(w, fmt.Sprintf("OK"))
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
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
	rtr.HandleFunc("/health-check", healthCheck).Methods("GET")
	rtr.HandleFunc("/purge", purgeCache).Methods("GET")
	rtr.HandleFunc("/warmup", warmUp).Methods("GET")

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.Port),
		Handler: rtr,
	}

	server.ListenAndServe()
}
