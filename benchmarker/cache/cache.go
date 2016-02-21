package cache

import (
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/catatsuy/private-isu/benchmarker/util"
	"github.com/marcw/cachecontrol"
)

type cacheStore struct {
	sync.RWMutex
	items map[string]*URLCache
}

func NewCacheStore() *cacheStore {
	m := make(map[string]*URLCache)
	c := &cacheStore{
		items: m,
	}
	return c
}

func (c *cacheStore) Get(key string) (*URLCache, bool) {
	c.RLock()
	v, found := c.items[key]
	c.RUnlock()
	return v, found
}

func (c *cacheStore) Set(key string, value *URLCache) {
	c.Lock()
	c.items[key] = value
	c.Unlock()
}

var instance *cacheStore
var once sync.Once

func GetInstance() *cacheStore {
	once.Do(func() {
		instance = NewCacheStore()
	})

	return instance
}

type URLCache struct {
	LastModified string
	Etag         string
	ExpiresAt    time.Time
	CacheControl *cachecontrol.CacheControl
	MD5          string
}

func NewURLCache(res *http.Response) *URLCache {
	directive := res.Header.Get("Cache-Control")
	cc := cachecontrol.Parse(directive)
	noCache, _ := cc.NoCache()

	if len(directive) == 0 || noCache || cc.NoStore() {
		return nil
	}

	now := time.Now()
	lm := res.Header.Get("Last-Modified")
	etag := res.Header.Get("ETag")

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil
	}
	res.Body.Close()
	md5 := util.GetMD5(body)

	return &URLCache{
		LastModified: lm,
		Etag:         etag,
		ExpiresAt:    now.Add(cc.MaxAge()),
		CacheControl: &cc,
		MD5:          md5,
	}
}

func (c *URLCache) Available() bool {
	return time.Now().Before(c.ExpiresAt)
}

func (c *URLCache) Apply(req *http.Request) {
	if c.Available() {
		if c.LastModified != "" {
			req.Header.Add("If-Modified-Since", c.LastModified)
		}

		if c.Etag != "" {
			req.Header.Add("If-None-Match", c.Etag)
		}
	}
}
