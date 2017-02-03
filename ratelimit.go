package ratelimit

import (
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	bucket "github.com/DavidCai1993/token-bucket"
	"github.com/go-http-utils/headers"
)

// Version is this package's version number.
const Version = "0.3.0"

// GetIDFunc represents a function that return an ID for each request.
// All requests which have the same ID will be regarded from one source and
// be ratelimited.
type GetIDFunc func(*http.Request) string

func defaultGetIDFunc(req *http.Request) string {
	ra := req.RemoteAddr

	if ip := req.Header.Get(headers.XForwardedFor); ip != "" {
		ra = ip
	} else if ip := req.Header.Get(headers.XRealIP); ip != "" {
		ra = ip
	} else {
		ra, _, _ = net.SplitHostPort(ra)
	}

	return net.ParseIP(ra).String()
}

type expireMap map[string]time.Time

func (em expireMap) checkIfExpired(id string) bool {
	if e, ok := em[id]; ok {
		if time.Now().After(e) {
			return true
		}
	}

	return false
}

func (em expireMap) getOneExpiredID() (string, bool) {
	now := time.Now()

	for id, e := range em {
		if now.After(e) {
			return id, true
		}
	}

	return "", false
}

// Options is the ratelimit middleware options.
type Options struct {
	// GetIDFunc represents a function that return an ID for each request.
	// All requests which have the same ID will be regarded from one source and
	// be ratelimited.
	GetID GetIDFunc
	// Ratelimit factor: only Count requests can pass through in Duration.
	// By default is 1 minute.
	Duration time.Duration
	// Ratelimit factor: only Count requests can pass through in Duration.
	// By default is 1000.
	Count int64
}

// Handler wraps the http.Handler with reatelimit support (only count requests
// can pass through in duration).
func Handler(h http.Handler, opts Options) http.Handler {
	if opts.GetID == nil {
		opts.GetID = defaultGetIDFunc
	}
	if opts.Duration == 0 {
		opts.Duration = time.Minute
	}
	if opts.Count == 0 {
		opts.Count = 1000
	}

	mutex := sync.Mutex{}
	bucketsMap := map[string]*bucket.TokenBucket{}
	expireMap := expireMap{}
	interval := opts.Count / int64(opts.Duration/1e9) * 1e9

	// Start a deamon gorouinue to check expiring.
	// To take the performance into consideration, the deamon will
	// only check at most 1000 buckets or cost at most one second at
	// one tick.
	go func() {
		for now := range time.Tick(opts.Duration) {
			mutex.Lock()
			hasExpired := true
			numExpired := 0
			timeLimit := now.Add(time.Second)

			for hasExpired && (numExpired < 1000 || now.After(timeLimit)) {
				if id, ok := expireMap.getOneExpiredID(); ok {
					delete(expireMap, id)
					bucketsMap[id].Destory()
					delete(bucketsMap, id)
					numExpired++
				} else {
					hasExpired = false
				}
			}

			mutex.Unlock()
		}
	}()

	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		id := opts.GetID(req)

		mutex.Lock()

		b, ok := bucketsMap[id]

		if !ok || expireMap.checkIfExpired(id) {
			b = bucket.New(time.Duration(interval), opts.Count)
			bucketsMap[id] = b
		}

		ok = b.TryTake(1)
		avail := b.Availible()
		cap := b.Capability()
		expireMap[id] = time.Now().Add(opts.Duration)

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
