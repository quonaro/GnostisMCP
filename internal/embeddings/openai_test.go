package embeddings

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestEmbed_ConcurrencySerialized(t *testing.T) {
	var callCount int
	var mu sync.Mutex
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		callCount++
		mu.Unlock()
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"embedding":[0.1,0.2],"index":0}]}`))
	}))
	defer srv.Close()

	p := newOpenAICompatible(srv.URL, "test-model", "", 32)

	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := p.Embed(context.Background(), []string{"test"})
			if err != nil {
				t.Errorf("Embed failed: %v", err)
			}
		}()
	}
	close(start)
	wg.Wait()

	if callCount != 3 {
		t.Errorf("expected 3 API calls, got %d", callCount)
	}
}

func TestEmbed_ConcurrentCallsSucceed(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"embedding":[0.1,0.2],"index":0}]}`))
	}))
	defer srv.Close()

	p := newOpenAICompatible(srv.URL, "test-model", "", 32)

	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := p.Embed(context.Background(), []string{"test"})
			if err != nil {
				t.Errorf("Embed failed: %v", err)
			}
		}()
	}
	close(start)
	wg.Wait()
}
