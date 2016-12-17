package congomap

import (
	"fmt"
	"sync"
	"time"
)

// RefreshingCacheConfig specifies the configuration parameters for a RefreshingCache instance.
type RefreshingCacheConfig struct {
	GoodStaleDuration  time.Duration
	GoodExpiryDuration time.Duration
	BadStaleDuration   time.Duration
	BadExpiryDuration  time.Duration
	Lookup             func(string) (interface{}, error)
}

// RefreshingCache memoizes responses from a Querier.
type RefreshingCache struct {
	Config *RefreshingCacheConfig
	db     map[string]*lockingTimedValue
	dbLock sync.RWMutex
}

// NewRefreshingCache returns RefreshingCache that attempts to respond to Query methods by
// consulting its TTL cache, then directing the call to the underlying Querier if a valid response
// is not stored. Note this function accepts a pointer so creating an instance with defaults can be
// done by passing a nil value rather than a pointer to a RefreshingCacheConfig instance.
func NewRefreshingCache(config *RefreshingCacheConfig) (*RefreshingCache, error) {
	if config == nil {
		config = &RefreshingCacheConfig{}
	}
	if config.GoodStaleDuration < 0 {
		return nil, fmt.Errorf("cannot create RefreshingCache with negative good stale duration: %v", config.GoodStaleDuration)
	}
	if config.GoodExpiryDuration < 0 {
		return nil, fmt.Errorf("cannot create RefreshingCache with negative good expiry duration: %v", config.GoodExpiryDuration)
	}

	if config.GoodStaleDuration > 0 && config.GoodExpiryDuration > 0 && config.GoodStaleDuration >= config.GoodExpiryDuration {
		return nil, fmt.Errorf("cannot create RefreshingCache with good stale duration not less than good expiry duration: %v; %v", config.GoodStaleDuration, config.GoodExpiryDuration)
	}

	if config.BadStaleDuration < 0 {
		return nil, fmt.Errorf("cannot create RefreshingCache with negative bad stale duration: %v", config.BadStaleDuration)
	}
	if config.BadExpiryDuration < 0 {
		return nil, fmt.Errorf("cannot create RefreshingCache with negative bad expiry duration: %v", config.BadExpiryDuration)
	}
	if config.BadStaleDuration > 0 && config.BadExpiryDuration > 0 && config.BadStaleDuration >= config.BadExpiryDuration {
		return nil, fmt.Errorf("cannot create RefreshingCache with bad stale duration not less than bad expiry duration: %v; %v", config.BadStaleDuration, config.BadExpiryDuration)
	}
	if config.Lookup == nil {
		config.Lookup = func(_ string) (interface{}, error) {
			return nil, ErrNoLookupDefined{}
		}
	}
	return &RefreshingCache{
		Config: config,
		db:     make(map[string]*lockingTimedValue),
	}, nil
}

// LoadStore loads the value associated with the specified key from the cache.
func (rc *RefreshingCache) LoadStore(key string) (interface{}, error) {
	return rc.ensureTopLevelThenAcquire(key, func(ltv *lockingTimedValue) (interface{}, error) {
		// NOTE: also check whether value filled while waiting for lock above
		if ltv.tv == nil {
			rc.fetch(key, ltv)
			return ltv.tv.Value, ltv.tv.Err
		}

		now := time.Now()

		// NOTE: When value is an error, we want to reverse handling of zero time checks: a zero
		// expiry time does not imply no expiry, but rather already expired. Similarly, a zero stale
		// time does not imply no stale, but rather already stale.
		if ltv.tv.Err != nil {
			if ltv.tv.Expiry.IsZero() || now.After(ltv.tv.Expiry) {
				rc.fetch(key, ltv)
			} else if ltv.tv.Stale.IsZero() || now.After(ltv.tv.Stale) {
				// NOTE: following immediately blocks until this method's deferred unlock executes
				go func(key string, ltv *lockingTimedValue) {
					ltv.lock.Lock()
					rc.fetch(key, ltv)
					ltv.lock.Unlock()
				}(key, ltv)
			}
			return ltv.tv.Value, ltv.tv.Err
		}

		// NOTE: When value is not an error, a zero expiry time means it never expires and a zero
		// stale time means the value does not get stale.
		if !ltv.tv.Expiry.IsZero() && now.After(ltv.tv.Expiry) {
			rc.fetch(key, ltv)
		} else if !ltv.tv.Stale.IsZero() && now.After(ltv.tv.Stale) {
			// NOTE: following immediately blocks until this method's deferred unlock executes
			go func(key string, ltv *lockingTimedValue) {
				ltv.lock.Lock()
				rc.fetch(key, ltv)
				ltv.lock.Unlock()
			}(key, ltv)
		}
		return ltv.tv.Value, ltv.tv.Err
	})
}

// Fetch method attempts to fetch a new value for the specified key. If the fetch is successful, it
// stores the value in the lockingTimedValue associated with the key.
func (rc *RefreshingCache) fetch(key string, lv *lockingTimedValue) {
	staleDuration := rc.Config.GoodStaleDuration
	expiryDuration := rc.Config.GoodExpiryDuration

	value, err := rc.Config.Lookup(key)
	if err != nil {
		staleDuration = rc.Config.BadStaleDuration
		expiryDuration = rc.Config.BadExpiryDuration
	}

	lv.tv = newTimedValue(value, err, staleDuration, expiryDuration)
}

// Store saves the key/value pair to the cache, overwriting whatever was previously stored.
func (rc *RefreshingCache) Store(key string, value interface{}) {
	_, _ = rc.ensureTopLevelThenAcquire(key, func(ltv *lockingTimedValue) (interface{}, error) {
		ltv.tv = newTimedValue(value, nil, rc.Config.GoodStaleDuration, rc.Config.GoodExpiryDuration)
		return nil, nil
	})
}

func (rc *RefreshingCache) ensureTopLevelThenAcquire(key string, callback func(*lockingTimedValue) (interface{}, error)) (interface{}, error) {
	rc.dbLock.RLock()
	ltv, ok := rc.db[key]
	rc.dbLock.RUnlock()
	if !ok {
		rc.dbLock.Lock()
		// check whether value filled while waiting for lock above
		ltv, ok = rc.db[key]
		if !ok {
			ltv = &lockingTimedValue{}
			rc.db[key] = ltv
		}
		rc.dbLock.Unlock()
	}

	ltv.lock.Lock()
	defer ltv.lock.Unlock()
	return callback(ltv)
}

// Close() error
// Delete(string)
// GC()
// Keys() []string
// Load(string) (interface{}, bool)
// Pairs() <-chan *Pair
// Lookup(func(string) (interface{}, error)) error
// Reaper(func(interface{})) error
// TTL(time.Duration) error
