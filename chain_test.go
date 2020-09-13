// Package alice implements a middleware chaining solution.
package alice

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

// A constructor for middleware
// that writes its own "tag" into the RW and does nothing else.
// Useful in checking if a chain is behaving in the right order.
func tagMiddleware(tag string) Constructor {
	return func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
		return fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) {
			fmt.Fprint(ctx, tag)
			next(ctx)
		})
	}
}

// Not recommended (https://golang.org/pkg/reflect/#Value.Pointer),
// but the best we can do.
func funcsEqual(f1, f2 interface{}) bool {
	val1 := reflect.ValueOf(f1)
	val2 := reflect.ValueOf(f2)
	return val1.Pointer() == val2.Pointer()
}

var testApp = fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) {
	fmt.Fprint(ctx, "app")
	ctx.SetStatusCode(fasthttp.StatusOK)
})

func testStatusOk(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(fasthttp.StatusOK)
	}
}


func TestNew(t *testing.T) {
	c1 := func(h fasthttp.RequestHandler) fasthttp.RequestHandler {
		return nil
	}

	c2 := testStatusOk

	slice := []Constructor{c1, c2}

	chain := New(slice...)
	for k := range slice {
		if !funcsEqual(chain.constructors[k], slice[k]) {
			t.Error("New does not add constructors correctly")
		}
	}
}

func TestThenWorksWithNoMiddleware(t *testing.T) {
	if !funcsEqual(New().Then(testApp), testApp) {
		t.Error("Then does not work with no middleware")
	}
}

func startServerOnPort(t *testing.T, port int, h fasthttp.RequestHandler) io.Closer {
	ln, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", port))
	if err != nil {
		t.Fatalf("cannot start tcp server on port %d: %s", port, err)
	}
	go fasthttp.Serve(ln, h)
	return ln
}

func TestThenOrdersHandlersCorrectly(t *testing.T) {
	t1 := tagMiddleware("t1\n")
	t2 := tagMiddleware("t2\n")
	t3 := tagMiddleware("t3\n")

	chained := New(t1, t2, t3).Then(testApp)

	ln := startServerOnPort(t, 8081, chained)
	defer ln.Close()

	req, err := http.NewRequest("GET", "http://localhost:8081", nil)
	assert.Nil(t, err, "Test Then Handler should be able to create a  Request")

	resp, err := http.DefaultClient.Do(req)
	assert.Nil(t, err, "Sending the request must not return an error")
	assert.NotNil(t, resp, "Request response must not be nil")
	assert.Equal(t, fasthttp.StatusOK, resp.StatusCode, "Test Then Handler Order should return an OK status")

	body, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err, "Reading the body response should not return an error")
	assert.Equal(t, "t1\nt2\nt3\napp", string(body), "Request response should return the correct middleware output order")
}

func TestAppendAddsHandlersCorrectly(t *testing.T) {
	chain := New(tagMiddleware("t1\n"), tagMiddleware("t2\n"))
	newChain := chain.Append(tagMiddleware("t3\n"), tagMiddleware("t4\n"))
	assert.Equal(t, 2, len(chain.constructors), "chain should have 2 constructors")
	assert.Equal(t, 4, len(newChain.constructors), "newChain should have 4 constructors")
	chained := newChain.Then(testApp)

	ln := startServerOnPort(t, 8082, chained)
	defer ln.Close()

	req, err := http.NewRequest("GET", "http://localhost:8082", nil)
	assert.Nil(t, err, "Should be able to create a  Request")

	resp, err := http.DefaultClient.Do(req)
	assert.Nil(t, err, "Sending the request must not return an error")
	assert.NotNil(t, resp, "Request response must not be nil")
	assert.Equal(t, fasthttp.StatusOK, resp.StatusCode, "Request response should return an OK status")
	body, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err, "Reading the body response should not return an error")
	assert.Equal(t, "t1\nt2\nt3\nt4\napp", string(body), "Request response should return the correct middleware output order")
}

func TestAppendRespectsImmutability(t *testing.T) {
	chain := New(tagMiddleware(""))
	newChain := chain.Append(tagMiddleware(""))
	assert.NotEqual(t, &chain.constructors[0], &newChain.constructors[0], "Apppend does not respect immutability")
}

func TestExtendAddsHandlersCorrectly(t *testing.T) {
	chain1 := New(tagMiddleware("t1\n"), tagMiddleware("t2\n"))
	chain2 := New(tagMiddleware("t3\n"), tagMiddleware("t4\n"))
	newChain := chain1.Extend(chain2)
	assert.Equal(t, 2, len(chain1.constructors), "chain1 should have 2 constructors")
	assert.Equal(t, 2, len(chain2.constructors), "chain2 should have 4 constructors")
	assert.Equal(t, 4, len(newChain.constructors), "chain2 should have 4 constructors")

	chained := newChain.Then(testApp)

	ln := startServerOnPort(t, 8083, chained)
	defer ln.Close()

	req, err := http.NewRequest("GET", "http://localhost:8083", nil)
	assert.Nil(t, err, "Should be able to create a  Request")

	resp, err := http.DefaultClient.Do(req)
	assert.Nil(t, err, "Sending the request must not return an error")
	assert.NotNil(t, resp, "Request response must not be nil")
	assert.Equal(t, fasthttp.StatusOK, resp.StatusCode, "Request response should return an OK status")
	body, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err, "Reading the body response should not return an error")
	assert.Equal(t, "t1\nt2\nt3\nt4\napp", string(body), "Request response should return the correct middleware output order")
}

func TestExtendRespectsImmutability(t *testing.T) {
	chain := New(tagMiddleware(""))
	newChain := chain.Extend(New(tagMiddleware("")))
	assert.NotEqual(t, &chain.constructors[0], &newChain.constructors[0], "Extend does not respect immutability")
}

func TestDefaultHandler(t *testing.T) {
	chained := New().Then(nil)

	ln := startServerOnPort(t, 8084, chained)
	defer ln.Close()

	req, err := http.NewRequest("GET", "http://localhost:8084", nil)
	assert.Nil(t, err, "Should be able to create a  Request")

	resp, err := http.DefaultClient.Do(req)
	assert.Nil(t, err, "Sending the request must not return an error")
	assert.NotNil(t, resp, "Request response must not be nil")
	assert.Equal(t, fasthttp.StatusNotFound, resp.StatusCode, "Request response should return a Not Found status")
	body, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err, "Reading the body response should not return an error")
	assert.Equal(t, Default404Message, string(body), "Request response should return the Default404Message")
}
