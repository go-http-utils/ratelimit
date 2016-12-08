package ratelimit_test

import (
	"net/http"
	"time"

	"github.com/go-http-utils/ratelimit"
)

func Example() {
	getIDByReq := func(req *http.Request) string {
		return req.RemoteAddr
	}

	m := http.NewServeMux()

	m.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
		res.WriteHeader(http.StatusOK)

		res.Write([]byte("Hello Worlkd"))
	})

	http.ListenAndServe(":8080", ratelimit.Handler(m, getIDByReq, time.Second, 1000))
}
