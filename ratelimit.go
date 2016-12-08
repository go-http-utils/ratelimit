package ratelimit

import (
	"net/http"
	"strconv"
	"time"

	bucket "github.com/DavidCai1993/token-bucket"
	"github.com/go-http-utils/headers"
)

// Version is this package's version number.
const Version = "0.0.1"

// GetIDFunc represents a function that return an ID for each request.
// All requests which have the same ID will be regarded from one source and
// be ratelimited.
type GetIDFunc func(*http.Request) string

// Handle wraps the http.Handler with reatelimit support (only count requests
// can pass through in duration).
func Handle(h http.Handler, getID GetIDFunc, duration time.Duration, count int64) http.Handler {
	bucketsMap := make(map[string]*bucket.TokenBucket)
	interval := count / int64(duration)

	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		id := getID(req)

		b, ok := bucketsMap[id]

		if !ok {
			bucketsMap[id] = bucket.New(time.Duration(interval), count)
		}

		ok = b.TryTake(1)
		avail := b.Availible()
		resHeader := res.Header()

		if ok {
			resHeader.Set(headers.XRatelimitLimit, strconv.FormatInt(b.Capability(), 10))
			resHeader.Set(headers.XRatelimitRemaining, strconv.FormatInt(avail, 10))
		} else {
			resHeader.Set(headers.RetryAfter, strconv.FormatInt(interval, 10))
			res.WriteHeader(http.StatusTooManyRequests)
			res.Write([]byte(http.StatusText(http.StatusTooManyRequests)))
		}
	})
}
