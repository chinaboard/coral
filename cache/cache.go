package cache

import (
	"sync"
	"time"

	"github.com/juju/errors"
)

type Cache struct {
	data sync.Map
	ttl  time.Duration
}
type kv struct {
	value bool
	ttl   time.Time
}

func NewCache(ttl time.Duration) *Cache {
	cache := &Cache{ttl: ttl}
	cache.init()
	return cache
}

func (c *Cache) init() {
	go func(ch <-chan time.Time) {
		for range ch {
			c.data.Range(func(key, value interface{}) bool {
				if time.Since(value.(kv).ttl) > c.ttl {
					c.data.Delete(key)
				}
				return true
			})
		}
	}(time.Tick(time.Second))
}

func (c *Cache) Set(key string, value bool) {
	c.data.Store(key, kv{value: value, ttl: time.Now()})
}
func (c *Cache) Exist(key string) (bool, error) {
	v, ok := c.data.Load(key)
	if ok {
		c.Set(key, v.(kv).value)
		return v.(kv).value, nil
	}
	return false, errors.New("not found")
}
