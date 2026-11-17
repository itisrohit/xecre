package engine

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/moby/moby/client"
)

// Exercise the real Docker SDK against a fake daemon: no host Docker access.
func TestRemovalCapacity(t *testing.T) {
	for _, tc := range []struct {
		name     string
		statuses []int
		retained bool
	}{
		{"removed", []int{204}, false},
		{"already absent", []int{404}, false},
		{"transient failure", []int{500, 204}, false},
		{"persistent failure", []int{500, 500, 500}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "DELETE" {
					t.Errorf("unexpected method %s", r.Method)
				}
				status := tc.statuses[calls]
				calls++
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(status)
				if status != 204 {
					w.Write([]byte(`{"message":"test failure"}`))
				}
			}))
			defer server.Close()
			cli, err := client.New(client.WithHost(server.URL), client.WithAPIVersion("1.51"))
			if err != nil {
				t.Fatal(err)
			}
			defer cli.Close()
			e := &DockerEngine{Client: cli, slots: make(chan struct{}, 1)}
			e.slots <- struct{}{}
			e.containers.Store("sandbox", true)
			e.remove("sandbox")
			_, retained := e.containers.Load("sandbox")
			if retained != tc.retained || (len(e.slots) == 1) != tc.retained {
				t.Fatalf("tracking/slot mismatch: retained=%v slots=%d", retained, len(e.slots))
			}
			if calls != len(tc.statuses) {
				t.Fatalf("calls=%d", calls)
			}
		})
	}
}

func TestOutputBudgetSharedAcrossStreams(t *testing.T) {
	remaining := 4
	var stdout, stderr bytes.Buffer
	out, errout := boundedOutput{&stdout, &remaining}, boundedOutput{&stderr, &remaining}
	if _, err := out.Write([]byte("abc")); err != nil {
		t.Fatal(err)
	}
	if _, err := errout.Write([]byte("de")); err == nil {
		t.Fatal("combined budget must reject excess")
	}
	if stderr.Len() != 0 || remaining != 1 {
		t.Fatal("excess output changed buffer or budget")
	}
}

func TestPauseCancelsImmediately(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if pause(ctx, time.Hour) {
		t.Fatal("cancelled wait continued")
	}
}
