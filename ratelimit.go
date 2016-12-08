package ratelimit

import (
	"net/http"
	"strconv"
	"time"

	bucket "github.com/DavidCai1993/token-bucket"
	"github.com/go-http-utils/headers"
)

// Version is this package's version number.
const Version = "0.1.0"

// GetIDFunc represents a function that return an ID for each request.
// All requests which have the same ID will be regarded from one source and
// be ratelimited.
type GetIDFunc func(*http.Request) string

// Handler wraps the http.Handler with reatelimit support (only count requests
// can pass through in duration).
func Handler(h http.Handler, getID GetIDFunc, duration time.Duration, count int64) http.Handler {
	bucketsMap := map[string]*bucket.TokenBucket{}
	interval := count / int64(duration/1e9) * 1e9

	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		id := getID(req)

		b, ok := bucketsMap[id]

		if !ok {
			b = bucket.New(time.Duration(interval), count)

			bucketsMap[id] = b
		}

		ok = b.TryTake(1)
		avail := b.Availible()
		resHeader := res.Header()

		if ok {
			resHeader.Set(headers.XRatelimitLimit, strconv.FormatInt(b.Capability(), 10))
			resHeader.Set(headers.XRatelimitRemaining, strconv.FormatInt(avail, 10))

			h.ServeHTTP(res, req)
		} else {
			resHeader.Set(headers.RetryAfter, strconv.FormatInt(interval/1e9, 10))
			res.WriteHeader(http.StatusTooManyRequests)
			res.Write([]byte(http.StatusText(http.StatusTooManyRequests)))
		}
	})
}
