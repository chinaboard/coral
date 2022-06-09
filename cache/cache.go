package cache

import (
	"net"
	"strings"
	"sync"
	"time"

	"github.com/chinaboard/coral/utils"
	log "github.com/sirupsen/logrus"

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

func (c *Cache) ShouldDirect(key string) (string, bool) {
	d, notFound := c.Exist(key)
	ip := ""
	if notFound != nil {
		host, _, _ := net.SplitHostPort(key)
		if strings.TrimSpace(host) == "" {
			host = key
		}
		ips, err := net.LookupIP(host)
		if err != nil {
			log.Warningln(err, host, "force use Proxy")
			d = false
		} else {
			ip = ips[0].String()
			d = utils.ShouldDirect(ip)
		}
		c.Set(key, d)
	}
	return ip, d
}
