package purdue_api

import (
	"time"

	"github.com/valyala/fasthttp"
)

// fastHeaders attaches the provided request headers (in map[string]string format) to the request.
// Can be used for application-type and cookies since I'm too lazy to create separate functional options for those
func fastHeaders(headers map[string]string) func(*fasthttp.Request, *fasthttp.Response, *time.Duration) {
	return func(request *fasthttp.Request, response *fasthttp.Response, timeout *time.Duration) {
		for key, value := range headers {
			request.Header.Set(key, value)
		}
	}
}

// fastTimeout performs the request using a timeout (DoTimeout versus Do)
// Does not perform the request using timeout if time duration is zero (not sure about the effects)
func fastTimeout(duration time.Duration) func(*fasthttp.Request, *fasthttp.Response, *time.Duration) {
	return func(request *fasthttp.Request, response *fasthttp.Response, timeout *time.Duration) {
		if duration != time.Duration(0) {
			*timeout = duration
		}
	}
}

// compactGET provides an interface for performing GET requests through the fasthttp package.
// Returns the response pointer (which must be externally released) and error, if any.
func compactGET(
	httpClient *fasthttp.Client,
	uri string,
	options ...func(*fasthttp.Request, *fasthttp.Response, *time.Duration),
) (response *fasthttp.Response, finalErr error) {
	var timeout time.Duration
	request := fasthttp.AcquireRequest()
	response = fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(request)

	// prepare request, functional options and defaults
	for _, option := range options {
		option(request, response, &timeout)
	}
	request.Header.SetMethod("GET")
	request.SetRequestURI(uri)

	// perform the actual request
	var err error
	if timeout != time.Duration(0) { // timeout set
		err = httpClient.DoTimeout(request, response, timeout)
	} else {
		err = httpClient.Do(request, response)
	}

	if err != nil {
		fasthttp.ReleaseResponse(response) // needed?
		return nil, err
	}

	return response, finalErr
}
