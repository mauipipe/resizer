package cache

import (
	"github.com/hellofresh/resizer/Godeps/_workspace/src/github.com/peterbourgon/diskv"
	"io"
	"github.com/hellofresh/resizer/Godeps/_workspace/src/github.com/hashicorp/golang-lru"
	"io/ioutil"
	"bytes"
	"image"
	"image/png"
	"image/jpeg"
)

type CacheStats struct {
	FileCacheHits 		uint64
	FileCacheMisses		uint64
	LruCacheHits		uint64
	LruCacheMisses		uint64
}

type CacheProvider struct {
	CacheAdapter 		*diskv.Diskv
	LruCache		 	LruCacheConfiguration
}

type LruCacheConfiguration struct {
	Enabled		bool
	Size 		int32
}

var lruCache 	*lru.Cache
var cacheStats 	*CacheStats

func init() {
	lruCache, _ = lru.New(3)
	cacheStats = new(CacheStats)
}

func (self *CacheStats) hitLru() {
	self.LruCacheHits++
}

func (self *CacheStats) missLru() {
	self.LruCacheMisses++
}

func (self *CacheStats) hitFileCache() {
	self.FileCacheHits++
}

func (self *CacheStats) missFileCache() {
	self.FileCacheMisses++
}

// Return the cache stats info
func (self *CacheProvider) GetStats() (*CacheStats, int) {
	return cacheStats, lruCache.Len()
}

// Check if item exists on the cache
func (self *CacheProvider) Contains(key string) bool {
	if self.LruCache.Enabled && lruCache.Contains(key) {
		return true
	}

	return self.CacheAdapter.Has(key)
}

// Get the content from a cache key
func (self *CacheProvider ) Get(key string, extension string) (image.Image, error) {
	if self.LruCache.Enabled && lruCache.Contains(key) {
		buffer, found := lruCache.Get(key)
		bufferInBytes := buffer.([]byte)

		if found {
			cacheStats.hitLru()
			reader := ioutil.NopCloser(bytes.NewReader(bufferInBytes))
			var decodedImage image.Image
			var err error

			if extension == "png" {
				decodedImage, err = png.Decode(reader)
			} else {
				decodedImage, err = jpeg.Decode(reader)
			}

			if err != nil {
				self.Delete(key)
			}

			return decodedImage, nil
		}
	}

	cacheStats.hitFileCache()

	reader, err := self.CacheAdapter.ReadStream(key, true)

	var decodedImage image.Image

	if extension == "png" {
		decodedImage, err = png.Decode(reader)
	}

	if extension == "jpg" {
		decodedImage, err = jpeg.Decode(reader)
	}

	if err != nil {
		self.Delete(key)
	}

	return decodedImage, err
}

// Set a new item on the cache
func (self *CacheProvider) Set(key string, r io.Reader) error {
	if self.LruCache.Enabled && lruCache.Contains(key) == false {
		buffer, err := ioutil.ReadAll(r)
		if err == nil {
			cacheStats.missLru()
			lruCache.Add(key, buffer)
		}
	}

	cacheStats.missFileCache()

	return self.CacheAdapter.WriteStream(key, r, true)
}

// Delete item from cache
func (self *CacheProvider) Delete(key string) error {
	if self.LruCache.Enabled && lruCache.Contains(key) {
		lruCache.Remove(key)
	}

	return self.CacheAdapter.Erase(key)
}

// Delete whole cache
func (self *CacheProvider) DeleteAll() error {
	if self.LruCache.Enabled {
		lruCache.RemoveOldest()
	}

	return self.CacheAdapter.EraseAll()
}