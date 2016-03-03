package main

import (
	"strings"
	"path"
	"github.com/hellofresh/resizer/cache"
	"os"
	"net/http"
	"time"
	"fmt"
	"strconv"
	"github.com/hellofresh/resizer/Godeps/_workspace/src/github.com/peterbourgon/diskv"
	"path/filepath"
	"net/url"
	"log"
)

const (
	transformBlockSize = 5 // grouping of chars per directory depth
	requestTimeout = 5
)

// Get id from a given url
func ExtractIdFromUrl(url string) string {
	i, j := strings.LastIndex(url, "/"), strings.LastIndex(url, path.Ext(url))
	name := url[i+1 : j]

	return name
}

// Creates the cache object with the cache provider
func SetCacheProvider() cache.CacheProvider {
	cacheAdapter := diskv.New(diskv.Options{
		BasePath:     os.Getenv("RESIZER_CACHE_PATH"),
		Transform:    blockTransform,
		CacheSizeMax: 1024 * 1024 * 1024,
	})

	return cache.CacheProvider{
		CacheAdapter: cacheAdapter,
		LruCache: cache.LruCacheConfiguration{
			Enabled: true,
			Size: 128,
		},
	}
}

// Used in the cache provider to build the folder structure
func blockTransform(s string) []string {
	var (
		sliceSize = len(s) / transformBlockSize
		pathSlice = make([]string, sliceSize)
	)

	for i := 0; i < sliceSize; i++ {
		from, to := i*transformBlockSize, (i*transformBlockSize)+transformBlockSize
		pathSlice[i] = s[from:to]
	}

	return pathSlice
}

// Return image
func GetExtension(givenUrl string) string {
	urlParsed, _ := url.Parse(givenUrl)
	parts := strings.Split(urlParsed.Path, ".")

	if parts[1] != "" {
		return parts[1]
	}

	return "jpeg"
}

// Given an image calculates the size
// This is a good place to put more middlewares
func GetImageSize(imageSize string, config *Configuration) *Size {
	size := new(Size)

	// Check if we have a placeholder for this
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

// Generates a basic a common client with default timeout
func GetClient() *http.Client {
	client := &http.Client{
		Timeout: time.Duration(requestTimeout) * time.Second,
	}

	return client
}

// Return a given error in JSON format to the ResponseWriter
func FormatError(err error, w http.ResponseWriter) {
	http.Error(w, fmt.Sprintf("{ \"error\": \"%s\"}", err), 400)
}

// Parse a given string into a uint value
func parseInteger(value string) (uint, error) {
	integer, err := strconv.ParseFloat(value, 64)
	log.Printf("%d", uint(integer))
	return uint(integer), err
}

// Returns size of given path
func DirSize(path string) (int64, error) {
	var size int64

	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
	}

	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}