package main

import (
	"bytes"
	"fmt"
	"log"
	"net/http"
	"strings"
	"testing"
)

type fakeHRW struct {
	data **bytes.Buffer
}

func (f fakeHRW) Header() http.Header {
	return *new(http.Header)
}
func (f fakeHRW) WriteHeader(int) {}

func (f fakeHRW) Write(xx []byte) (int, error) {
	return (*f.data).Write(xx)
}

// mockHTTP takes a function that would be used as a handlefunc
// a request type, the endpoint, a body, and returns what would
// have been written back to the client.
func MockHTTP(pretend func(w http.ResponseWriter, r *http.Request),
	body string) string {
	req, err := http.NewRequest("POST", "http://localhost:90309/test", strings.NewReader(body))
	if err != nil {
		log.Println(err)
	}
	xx := bytes.NewBuffer([]byte{})
	buf := &xx
	w := fakeHRW{buf}
	pretend(w, req)
	return (*buf).String()
}

func TestMockHTTP(t *testing.T) {
	t.Run("mock", func(t *testing.T) {

		in := "hello"
		out := MockHTTP(func(w http.ResponseWriter,
			r *http.Request) {
			bts := make([]byte, 10, 10000000)
			_, err := (*r).Body.Read(bts)
			if err != nil {
				t.Log(err)
			}
			w.Write(bts)
		},
			"hello")
		// some null characters come out...
		if in != strings.Trim(out, "\x00") {
			t.Fail()
		}
	})

}

func TestServer(t *testing.T) {
	testId := ""
	t.Run("/new", func(t *testing.T) {
		start := len(registry)
		// shouldn't work in the future. Not enough parameters.
		testId = MockHTTP(ServeNew, "{\"Resolution\":1,\"keys\":[\"b\",\"v\"], \"maxval\":1000,\"maxtime\":10000}")
		end := len(registry)
		if len(testId) < 26 || start+1 != end {
			t.Fail()
		}
	})

	t.Run("/add", func(t *testing.T) {
		// should work
		resp := MockHTTP(ServeAdd, fmt.Sprintf("{\"ID\":\"%v\",\"kvs\":{\"b\":\"b\",\"v\":\"v\"}, \"time\":1, \"value\":12, \"count\":3}", testId))
		resp = MockHTTP(ServeAdd, fmt.Sprintf("{\"ID\":\"%v\",\"kvs\":{\"b\":\"b\",\"v\":\"v\"}, \"time\":1, \"value\":38, \"count\":9}", testId))
		if "ok" != strings.Trim(resp, "\x00") {
			t.Log(resp)
			t.Fail()
		}
		// shouldn't work.
		resp = MockHTTP(ServeAdd, "{\"ID\":\"1234\",\"Resolution\":1,\"kvs\":{\"b\":\"v\"}}")
		if "ok" == strings.Trim(resp, "\x00") {
			t.Log(resp)
			t.Fail()
		}

	})

	t.Run("/quantiles", func(t *testing.T) {
		// should work
		resp := MockHTTP(ServeQuantiles, fmt.Sprintf("{\"ID\":\"%v\",\"kvs\":{\"b\":\"b\",\"v\":\"v\"}, \"time\":1, \"quants\":[0.1,0.5,0.9]}", testId))
		if "[12 38 38]" != strings.Trim(resp, "\x00") {
			t.Log(resp)
			t.Fail()
		}
		// shouldn't work.
		resp = MockHTTP(ServeQuantiles, "{\"ID\":\"1234\",\"Resolution\":1,\"keys\":[\"b\",\"v\"]}")
		if "ID not found" != strings.Trim(resp, "\x00") {
			t.Log(resp)
			t.Fail()
		}

	})

	t.Run("/delete", func(t *testing.T) {
		// should work
		resp := MockHTTP(ServeDelete, fmt.Sprintf("{\"ID\":\"%v\",\"Resolution\":1,\"keys\":[\"b\",\"v\"]}", testId))
		if "ok" != strings.Trim(resp, "\x00") {
			t.Log(resp)
			t.Fail()
		}
		// shouldn't work.
		resp = MockHTTP(ServeAdd, fmt.Sprintf("{\"ID\":\"%v\",\"Resolution\":1,\"keys\":[\"b\",\"v\"]}", testId))
		if "ok" == strings.Trim(resp, "\x00") {
			t.Log(resp)
			t.Fail()
		}

		// shouldn't work.
		resp = MockHTTP(ServeDelete, "{\"ID\":\"1234\",\"Resolution\":1,\"keys\":[\"b\",\"v\"]}")
		if "ok" == strings.Trim(resp, "\x00") {
			t.Log(resp)
			t.Fail()
		}

	})

}
