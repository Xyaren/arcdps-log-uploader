package main

import (
	"context"
	"golang.org/x/time/rate"
	"net/http"
)

//RLHTTPClient Rate Limited HTTP Client
type RLHTTPClient struct {
	client      *http.Client
	RateLimiter *rate.Limiter
}

//Do dispatches the HTTP request to the network
func (c *RLHTTPClient) Do(req *http.Request, activationCallback func()) (*http.Response, error) {
	// Comment out the below 5 lines to turn off ratelimiting
	ctx := context.Background()
	err := c.RateLimiter.Wait(ctx) // This is a blocking call. Honors the rate limit
	if err != nil {
		return nil, err
	}
	activationCallback()
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

//NewRateLimitedClient return http client with a rateLimiter
func NewRateLimitedClient(rl *rate.Limiter) *RLHTTPClient {
	c := &RLHTTPClient{
		client:      http.DefaultClient,
		RateLimiter: rl,
	}
	return c
}
