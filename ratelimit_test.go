package ratelimit

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/go-http-utils/headers"
	"github.com/stretchr/testify/suite"
)

type RatelimitSuite struct {
	suite.Suite

	mux http.Handler
}

func (s *RatelimitSuite) SetupSuite() {
	mux := http.NewServeMux()
	mux.Handle("/", http.HandlerFunc(helloHandler))
	s.mux = mux
}

func (s *RatelimitSuite) TestHasRemaining() {
	server := httptest.NewServer(Handler(s.mux, Options{
		GetID:    getID,
		Duration: time.Second * 2,
		Count:    2,
	}))

	req, err := http.NewRequest(http.MethodGet, server.URL+"/", nil)
	s.Nil(err)

	res, err := sendRequest(req)
	s.Nil(err)
	s.Equal(http.StatusOK, res.StatusCode)
}

func (s *RatelimitSuite) TestHasNoMoreRemaining() {
	server := httptest.NewServer(Handler(s.mux, Options{
		Duration: time.Second * 2,
		Count:    2,
	}))

	for i := 0; i < 3; i++ {
		req, err := http.NewRequest(http.MethodGet, server.URL+"/", nil)
		s.Nil(err)

		res, err := sendRequest(req)
		s.Nil(err)

		if i < 2 {
			s.Equal(http.StatusOK, res.StatusCode)
			s.Equal("2", res.Header.Get(headers.XRatelimitLimit))
			s.Equal(strconv.Itoa(1-i), res.Header.Get(headers.XRatelimitRemaining))
		} else {
			s.Equal(http.StatusTooManyRequests, res.StatusCode)
			s.NotEmpty(res.Header.Get(headers.RetryAfter))
		}
	}
}

func (s *RatelimitSuite) TestTwoTicks() {
	server := httptest.NewServer(Handler(s.mux, Options{
		GetID:    getID,
		Duration: time.Second * 2,
		Count:    2,
	}))

	for i := 0; i < 3; i++ {
		req, err := http.NewRequest(http.MethodGet, server.URL+"/", nil)
		s.Nil(err)

		res, err := sendRequest(req)
		s.Nil(err)

		if i < 2 {
			s.Equal(http.StatusOK, res.StatusCode)
			s.Equal("2", res.Header.Get(headers.XRatelimitLimit))
			s.Equal(strconv.Itoa(1-i), res.Header.Get(headers.XRatelimitRemaining))
		} else {
			s.Equal(http.StatusTooManyRequests, res.StatusCode)
			s.NotEmpty(res.Header.Get(headers.RetryAfter))
		}
	}

	<-time.After(2 * time.Second)

	for i := 0; i < 3; i++ {
		req, err := http.NewRequest(http.MethodGet, server.URL+"/", nil)
		s.Nil(err)

		res, err := sendRequest(req)
		s.Nil(err)

		if i < 2 {
			s.Equal(http.StatusOK, res.StatusCode)
			s.Equal("2", res.Header.Get(headers.XRatelimitLimit))
			s.Equal(strconv.Itoa(1-i), res.Header.Get(headers.XRatelimitRemaining))
		} else {
			s.Equal(http.StatusTooManyRequests, res.StatusCode)
			s.NotEmpty(res.Header.Get(headers.RetryAfter))
		}
	}
}

func TestRatelimit(t *testing.T) {
	suite.Run(t, new(RatelimitSuite))
}

func helloHandler(res http.ResponseWriter, req *http.Request) {
	res.WriteHeader(http.StatusOK)

	res.Write([]byte("Hello World"))
}

func getID(req *http.Request) string {
	return "test"
}

func sendRequest(req *http.Request) (*http.Response, error) {
	cli := &http.Client{}
	return cli.Do(req)
}

func getResRawBody(res *http.Response) []byte {
	bytes, err := ioutil.ReadAll(res.Body)

	if err != nil {
		panic(err)
	}

	return bytes
}
