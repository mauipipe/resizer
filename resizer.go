package main

import (
	"bytes"
	"fmt"
	"github.com/hellofresh/resizer/Godeps/_workspace/src/github.com/gorilla/mux"
	"github.com/hellofresh/resizer/Godeps/_workspace/src/github.com/nfnt/resize"
	"github.com/hellofresh/resizer/Godeps/_workspace/src/github.com/spf13/viper"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"net/http"
	"runtime"
	"time"
	"runtime/debug"
	"os"
)

var (
	httpClient 		*http.Client
	config     		*Configuration
    cacheProvider 	= SetCacheProvider()
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

func init() {
	httpClient = GetClient()
}

// Resizing endpoint.
func resizing(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	start := time.Now()

	// Get parameters
	imageUrl := fmt.Sprintf("%s%s", config.ImageHost, params["path"])
	size := GetImageSize(params["size"], config)
	timestamp := r.URL.Query().Get("t")
	extension := GetExtension(imageUrl)

	validator := Validator{config}

	if err := validator.CheckRequestNewSize(size); err != nil {
		FormatError(err, w)
		return
	}

	// Build caching key
	imageId := ExtractIdFromUrl(imageUrl)
	key := fmt.Sprintf("%s_%s_%d_%d", imageId, timestamp, size.Height, size.Width)

	if config.Cachethumbnails && cacheProvider.Contains(key) {
		finalImage, _ := cacheProvider.Get(key, extension)
		jpeg.Encode(w, finalImage, nil)
		return
	}

	// Download the image
	originalImageKey := fmt.Sprintf("%s_%s_original", imageId, timestamp)

	imageBuffer := new(http.Response)
	var cachedHit bool

	if cacheProvider.Contains(originalImageKey) {
		cachedHit = true
	} else {
		cachedHit = false
		var err error

		imageBuffer, err = httpClient.Get(imageUrl)

		if err != nil {
			FormatError(err, w)
			return
		}

		defer imageBuffer.Body.Close()
	}

	defer r.Body.Close()

	if imageBuffer.StatusCode != 200 && cachedHit == false {
		http.NotFound(w, r)
		return
	}

	var finalImage image.Image
	var err error

	if cachedHit == false {
		if extension == "png" {
			finalImage, err = png.Decode(imageBuffer.Body)
		}

		if extension == "jpg" {
			finalImage, err = jpeg.Decode(imageBuffer.Body)
		}

		if err != nil {
			_ = cacheProvider.Delete(originalImageKey)
			_ = cacheProvider.Delete(key)
			log.Printf("Error jpeg.decode")

			FormatError(err, w)
			return
		}
	} else {
		var err error
		finalImage, err = cacheProvider.Get(originalImageKey, extension)

		if err != nil {
			log.Printf("Error reading stream %s", err)
		}
	}

	// calculate aspect ratio
	if size.Width > 0 && size.Height > 0 {
		b := finalImage.Bounds()
		sizer := Sizer{size}
		aspectedRatioSize := sizer.calculateAspectRatio(b.Max.Y, b.Max.X)
		size.Width = aspectedRatioSize.Width
		size.Height = aspectedRatioSize.Height
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
		if err := cacheProvider.Set(key, buf); err != nil {
			FormatError(err, w)
			return
		}
	}

	if cachedHit == false {
		originalBuf := new(bytes.Buffer)

		if extension == "png" {
			if err = png.Encode(originalBuf, finalImage); err != nil {
				log.Printf("Error encoding")
			}
		}

		if extension == "jpg" {
			if err = jpeg.Encode(originalBuf, finalImage, nil); err != nil {
				log.Printf("Error encoding")
			}
		}

		if err := cacheProvider.Set(originalImageKey, originalBuf); err != nil {
			FormatError(err, w)
			return
		}
	}

	var fromWhere string
	if cachedHit {
		fromWhere = "cache"
	} else {
		fromWhere = "network"
	}

	switch extension {
	case "png":
		png.Encode(w, imageResized)
		log.Printf("[%s] Successfully handled content type '%s Delivered in %f s'\n", fromWhere, "image/png", time.Since(start).Seconds())
	case "jpg":
		jpeg.Encode(w, imageResized, nil)
		log.Printf("[%s] Successfully handled content type '%s'  Delivered in %f s\n", fromWhere, "image/jpeg", time.Since(start).Seconds())
	default:
		log.Printf("[%s ]Cannot handle content type '%s'  Delivered in %f s\n", fromWhere, contentType, time.Since(start).Seconds())
	}

	// free memory
	debug.FreeOSMemory()

}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	var totalSize float32

	w.Header().Add("Content-Type", "application/json")

	dirSize, err := DirSize(os.Getenv("RESIZER_CACHE_PATH"))

	if dirSize > 0 {
		totalSize = float32(dirSize) / 1048576
	}

	if err != nil {
		totalSize = 0.0
	}

	stats, lruSize := cacheProvider.GetStats()

	response := fmt.Sprintf("{\"status\": \"ok\",\"cache\": [{\"file_cache\": {\"hits\": %d,\"misses\": %d}}, {\"lru_cache\": {\"hits\": %d,\"misses\": %d, \"size\": %d}}], \"used_space\": \"%f Mb\"}", stats.FileCacheHits, stats.FileCacheMisses, stats.LruCacheHits, stats.LruCacheMisses, lruSize, totalSize)
	fmt.Fprint(w, response)
}

func purgeCache(w http.ResponseWriter, r *http.Request) {
	err := cacheProvider.DeleteAll()

	if err != nil {
		FormatError(err, w)
		return
	}

	fmt.Fprint(w, fmt.Sprintf("OK"))
}

func main() {
	runtime.GOMAXPROCS(3)
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
//		ReadTimeout: 3 * time.Second,
//		WriteTimeout: 10 * time.Second,
	}

	server.ListenAndServe()
}
