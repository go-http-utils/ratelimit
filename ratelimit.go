package ratelimit

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	bucket "github.com/DavidCai1993/token-bucket"
	"github.com/go-http-utils/headers"
)

// Version is this package's version number.
const Version = "0.2.0"

// GetIDFunc represents a function that return an ID for each request.
// All requests which have the same ID will be regarded from one source and
// be ratelimited.
type GetIDFunc func(*http.Request) string

type expireBucket struct {
	bucket  *bucket.TokenBucket
	start   time.Time
	expired time.Time
}

// Handler wraps the http.Handler with reatelimit support (only count requests
// can pass through in duration).
func Handler(h http.Handler, getID GetIDFunc, duration time.Duration, count int64) http.Handler {
	mutex := sync.Mutex{}
	bucketsMap := map[string]*expireBucket{}
	interval := count / int64(duration/1e9) * 1e9

	go func() {
		for now := range time.Tick(duration) {
			mutex.Lock()

			for id, eb := range bucketsMap {
				if eb.expired.After(now) {
					delete(bucketsMap, id)
					eb.bucket.Destory()
				}
			}

			mutex.Unlock()
		}
	}()

	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		id := getID(req)

		mutex.Lock()

		b, ok := bucketsMap[id]

		if !ok {
			b = &expireBucket{
				bucket:  bucket.New(time.Duration(interval), count),
				expired: time.Now().Add(duration),
			}

			bucketsMap[id] = b
		}

		ok = b.bucket.TryTake(1)
		avail := b.bucket.Availible()
		cap := b.bucket.Capability()
		b.expired = time.Now().Add(duration)

		mutex.Unlock()

		resHeader := res.Header()

		if ok {
			resHeader.Set(headers.XRatelimitLimit, strconv.FormatInt(cap, 10))
			resHeader.Set(headers.XRatelimitRemaining, strconv.FormatInt(avail, 10))

			h.ServeHTTP(res, req)
		} else {
			resHeader.Set(headers.RetryAfter, strconv.FormatInt(interval/1e9, 10))
			res.WriteHeader(http.StatusTooManyRequests)
			res.Write([]byte(http.StatusText(http.StatusTooManyRequests)))
		}
	})
}
