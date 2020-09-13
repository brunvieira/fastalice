# Fast Alice

[![GoDoc](https://godoc.org/github.com/golang/gddo?status.svg)](http://godoc.org/github.com/brunvieira/fastalice)
[![Build Status](https://travis-ci.org/brunvieira/fastalice.svg?branch=master)](https://travis-ci.org/brunvieira/fastalice)
[![Coverage](http://gocover.io/_badge/github.com/brunvieira/fastalice)](http://gocover.io/github.com/brunvieira/fastalice)

Fast Alice is a port of [alice](https://github.com/justinas/alice) for [fasthttp]("github.com/valyala/fasthttp").
Fast Alice provides a convenient way to chain
your Fast HTTP middleware functions and the app handler.

In short, it transforms

```go
Middleware1(Middleware2(Middleware3(App)))
```

to

```go
fastalice.New(Middleware1, Middleware2, Middleware3).Then(App)
```

### Why?

None of the other middleware chaining solutions
behaves exactly like Alice.
Alice is as minimal as it gets:
in essence, it's just a for loop that does the wrapping for you.

Check out [this blog post](http://justinas.org/alice-painless-middleware-chaining-for-go/)
for explanation how Alice is different from other chaining solutions.

### Usage

Your middleware constructors should have the form of

```go
package main 

func (fasthttp.RequestHandler) fasthttp.RequestHandler
```

Some middleware provide this out of the box.
For ones that don't, it's trivial to write one yourself.

```go
package main 

func myLog(req fasthttp.RequestHandler) fasthttp.RequestHandler {
    return fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) {
    		log.Printf("%s %s - %v",
    			ctx.Method(),
    			ctx.RequestURI(),
    			ctx.Response.Header.StatusCode(),
    		)
    	})
}
```

This complete example shows the full power of Fast Alice.

```go
package main

import (
    "fmt"

    "github.com/AubSs/fasthttplogger"
    "github.com/brunvieira/fast-alice"
    "github.com/brunvieira/fastcsrf"
    "github.com/valyala/fasthttp"
)


func fastHTTPHandler(ctx *fasthttp.RequestCtx) {
	fmt.Fprintf(ctx, "Hi there! RequestURI is %q", ctx.RequestURI())
}

func main() {
    chained := fastalice.New(fastcsrf.CSRF, fasthttplogger.CombinedColored).Then(fastHTTPHandler)
    fasthttp.ListenAndServe(":8080", chained)
}
```

Here, the request will pass [fastcsrf](github.com/brunvieira/fastcsrf) first,
then [fasthttplogger](github.com/AubSs/fasthttplogger)
and will finally reach our handler.

Note that Fast Alice, as Alice, makes **no guarantees** for
how one or another piece of  middleware will behave.
Once it passes the execution to the outer layer of middleware,
it has no saying in whether middleware will execute the inner handlers.
This is intentional behavior.

Fast Alice works with Go 1.0 and higher.

### Contributing

0. Find an issue that bugs you / open a new one.
1. Discuss.
2. Branch off, commit, test.
3. Make a pull request / attach the commits to the issue.
