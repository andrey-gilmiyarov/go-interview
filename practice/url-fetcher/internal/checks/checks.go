package checks

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"github.com/andreygilmiyarov/go-interview/practice/url-fetcher/contract"
)

type fetchFunc func(context.Context, *http.Client, []string, int, time.Duration) ([]contract.Result, error)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

type trackedBody struct {
	closed atomic.Bool
}

func (b *trackedBody) Read([]byte) (int, error) { return 0, io.EOF }
func (b *trackedBody) Close() error {
	b.closed.Store(true)
	return nil
}

type fetchResult struct {
	values []contract.Result
	err    error
}

func await[T any](t *testing.T, ch <-chan T) T {
	t.Helper()
	timer := time.NewTimer(5 * time.Second)
	defer timer.Stop()
	select {
	case value := <-ch:
		return value
	case <-timer.C:
		t.Fatal("timed out waiting for HTTP operation")
		var zero T
		return zero
	}
}

func assertNoResult(t *testing.T, ch <-chan fetchResult) {
	t.Helper()
	synctest.Wait()
	select {
	case result := <-ch:
		t.Fatalf("FetchAll returned before a started request finished: %#v", result)
	default:
	}
}

func Run(t *testing.T, fetch fetchFunc) {
	t.Helper()
	t.Run("keeps URL order, stores per-URL errors, closes response bodies", func(t *testing.T) {
		dialErr := errors.New("local transport failure")
		var bodies [3]*trackedBody
		client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
			switch request.URL.Path {
			case "/one":
				bodies[0] = &trackedBody{}
				return &http.Response{StatusCode: http.StatusCreated, Header: make(http.Header), Body: bodies[0], Request: request}, nil
			case "/broken":
				return nil, dialErr
			case "/three":
				bodies[2] = &trackedBody{}
				return &http.Response{StatusCode: http.StatusNoContent, Header: make(http.Header), Body: bodies[2], Request: request}, nil
			default:
				return nil, fmt.Errorf("unexpected path %s", request.URL.Path)
			}
		})}
		urls := []string{"http://local/one", "http://local/broken", "http://local/three"}
		got, err := fetch(context.Background(), client, urls, 2, time.Second)
		if err != nil {
			t.Fatalf("FetchAll returned top-level error: %v", err)
		}
		if len(got) != len(urls) {
			t.Fatalf("FetchAll returned %d results, want %d", len(got), len(urls))
		}
		for index, url := range urls {
			if got[index].URL != url {
				t.Fatalf("result[%d].URL = %q, want %q", index, got[index].URL, url)
			}
		}
		if got[0].Status != http.StatusCreated || got[0].Err != nil ||
			got[1].Status != 0 || !errors.Is(got[1].Err, dialErr) ||
			got[2].Status != http.StatusNoContent || got[2].Err != nil {
			t.Fatalf("FetchAll results = %#v", got)
		}
		if bodies[0] == nil || !bodies[0].closed.Load() || bodies[2] == nil || !bodies[2].closed.Load() {
			t.Fatal("FetchAll left an HTTP response body open")
		}
	})

	t.Run("bounds active HTTP requests", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			var active atomic.Int32
			var maximum atomic.Int32
			started := make(chan string, 4)
			release := make(chan struct{})
			client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
				current := active.Add(1)
				for {
					previous := maximum.Load()
					if current <= previous || maximum.CompareAndSwap(previous, current) {
						break
					}
				}
				started <- request.URL.Path
				<-release
				active.Add(-1)
				return &http.Response{
					StatusCode: http.StatusNoContent,
					Header:     make(http.Header),
					Body:       io.NopCloser(strings.NewReader("")),
					Request:    request,
				}, nil
			})}
			urls := []string{"http://local/1", "http://local/2", "http://local/3", "http://local/4"}
			done := make(chan fetchResult, 1)
			go func() {
				values, err := fetch(context.Background(), client, urls, 2, time.Hour)
				done <- fetchResult{values: values, err: err}
			}()
			await(t, started)
			await(t, started)
			synctest.Wait()
			select {
			case path := <-started:
				close(release)
				t.Fatalf("FetchAll started more than two requests while both workers were blocked; extra path %s", path)
			default:
			}
			close(release)
			result := await(t, done)
			if result.err != nil || len(result.values) != len(urls) {
				t.Fatalf("FetchAll = %d results, %v; want four results", len(result.values), result.err)
			}
			if got := maximum.Load(); got != 2 {
				t.Fatalf("maximum active requests = %d, want exactly 2", got)
			}
		})
	})

	t.Run("applies a real per-request deadline", func(t *testing.T) {
		// The timeout is valid even if it expires before the server sees the request.
		server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
			<-request.Context().Done()
		}))
		defer func() {
			// Close active connections before Close waits for server handlers.
			server.CloseClientConnections()
			server.Close()
		}()
		parentCtx, parentCancel := context.WithCancel(context.Background())
		defer parentCancel()
		client := server.Client()
		done := make(chan fetchResult, 1)
		go func() {
			values, err := fetch(parentCtx, client, []string{server.URL}, 1, 30*time.Millisecond)
			done <- fetchResult{values: values, err: err}
		}()
		result := await(t, done)
		if result.err != nil || len(result.values) != 1 || !errors.Is(result.values[0].Err, context.DeadlineExceeded) {
			t.Fatalf("FetchAll deadline result = %#v, top-level error %v", result.values, result.err)
		}
		if parentCtx.Err() != nil {
			t.Fatalf("per-request timeout canceled parent context: %v", parentCtx.Err())
		}
	})

	t.Run("parent cancellation joins active request and returns no partial results", func(t *testing.T) {
		synctest.Test(t, func(t *testing.T) {
			started := make(chan struct{})
			observedCancel := make(chan struct{})
			release := make(chan struct{})
			roundTripReturned := make(chan struct{})
			client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
				close(started)
				<-request.Context().Done()
				close(observedCancel)
				<-release
				close(roundTripReturned)
				return nil, request.Context().Err()
			})}
			ctx, cancel := context.WithCancel(context.Background())
			done := make(chan fetchResult, 1)
			go func() {
				values, err := fetch(ctx, client, []string{"http://local/wait"}, 1, time.Second)
				done <- fetchResult{values: values, err: err}
			}()
			await(t, started)
			cancel()
			await(t, observedCancel)
			assertNoResult(t, done)
			close(release)
			result := await(t, done)
			await(t, roundTripReturned)
			if result.values != nil || !errors.Is(result.err, context.Canceled) {
				t.Fatalf("FetchAll after cancellation = %#v, %v; want nil, context.Canceled", result.values, result.err)
			}
		})
	})

	t.Run("rejects invalid configuration and honors pre-cancellation", func(t *testing.T) {
		for _, testCase := range []struct {
			name    string
			client  *http.Client
			workers int
			timeout time.Duration
		}{
			{name: "nil client", workers: 1, timeout: time.Second},
			{name: "zero workers", client: http.DefaultClient, workers: 0, timeout: time.Second},
			{name: "zero timeout", client: http.DefaultClient, workers: 1},
		} {
			t.Run(testCase.name, func(t *testing.T) {
				if _, err := fetch(context.Background(), testCase.client, []string{"http://local/"}, testCase.workers, testCase.timeout); err == nil {
					t.Fatal("FetchAll accepted invalid configuration")
				}
			})
		}
		empty, err := fetch(context.Background(), http.DefaultClient, nil, 1, time.Second)
		if err != nil || len(empty) != 0 {
			t.Fatalf("FetchAll(empty) = %#v, %v; want empty result", empty, err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		values, err := fetch(ctx, http.DefaultClient, nil, 1, time.Second)
		if !errors.Is(err, context.Canceled) || values != nil {
			t.Fatalf("FetchAll(pre-canceled, empty) = %#v, %v; want nil, context.Canceled", values, err)
		}
	})
}
