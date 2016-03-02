package cache
import (
	"github.com/hellofresh/resizer/Godeps/_workspace/src/github.com/peterbourgon/diskv"
	"io"
)

type CacheProvider struct {
	CacheAdapter *diskv.Diskv
}

// Check if item exists on the cache
func (self *CacheProvider) Contains(key string) bool {
	return self.CacheAdapter.Has(key)
}

// Get the content from a cache key
func (self *CacheProvider ) Get(key string) (io.ReadCloser, error) {
	return self.CacheAdapter.ReadStream(key, true)
}

// Set a new item on the cache
func (self *CacheProvider) Set(key string, r io.Reader) error {
	return self.CacheAdapter.WriteStream(key, r, true)
}

// Delete item from cache
func (self *CacheProvider) Delete(key string) error {
	return self.CacheAdapter.Erase(key)
}

// Delete whole cache
func (self *CacheProvider) DeleteAll() error {
	return self.CacheAdapter.EraseAll()
}