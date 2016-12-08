# ratelimit
[![Build Status](https://travis-ci.org/go-http-utils/ratelimit.svg?branch=master)](https://travis-ci.org/go-http-utils/ratelimit)
[![Coverage Status](https://coveralls.io/repos/github/go-http-utils/ratelimit/badge.svg?branch=master)](https://coveralls.io/github/go-http-utils/ratelimit?branch=master)

HTTP ratelimit middleware for Go.

## Installation

```
go get -u github.com/go-http-utils/ratelimit
```

## Documentation

API documentation can be found here: https://godoc.org/github.com/go-http-utils/ratelimit

## Usage

```go
import (
  "github.com/go-http-utils/ratelimit"
)
```

```go
getIDByReq := func(req *http.Request) string {
  return req.RemoteAddr
}

m := http.NewServeMux()

m.HandleFunc("/", func(res http.ResponseWriter, req *http.Request) {
  res.WriteHeader(http.StatusOK)

  res.Write([]byte("Hello Worlkd"))
})

http.ListenAndServe(":8080", ratelimit.Handler(m, getIDByReq, time.Second, 1000))
```
