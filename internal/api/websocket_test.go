package api

import (
	"testing"
	"time"
)

func TestHub(t *testing.T) {
	h := newHub()
	go h.run()

	ch := make(chan []byte, 10)
	client := &Client{hub: h, send: ch}

	h.register <- client
	time.Sleep(10 * time.Millisecond)

	msg := []byte(`{"test": true}`)
	h.broadcast <- msg

	select {
	case received := <-ch:
		if string(received) != string(msg) {
			t.Errorf("received %q, want %q", received, msg)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for broadcast")
	}

	h.unregister <- client
	time.Sleep(10 * time.Millisecond)
}
