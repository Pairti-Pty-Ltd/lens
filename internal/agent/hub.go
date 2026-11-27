package agent

import (
	"context"
	"errors"
	"time"
)

const answerWait = 3 * time.Second

// Kind is a request the editor answers on its UI tick.
type Kind int

const (
	KindRead Kind = iota
	KindEdit
)

// Request is one tool call waiting for the editor.
type Request struct {
	Kind  Kind
	Read  ReadIn
	Edit  ApplyIn
	reply chan Response
}

// Response is the editor's answer.
type Response struct {
	Snap    Snapshot
	Applied ApplyOut
	Err     error
}

// Reply delivers the answer. The channel is buffered.
func (r Request) Reply(resp Response) {
	if r.reply != nil {
		r.reply <- resp
	}
}

// Hub carries tool calls onto the editor goroutine and back.
type Hub struct {
	wake func()
	asks chan Request
}

// NewHub returns a hub. wake runs after a call is queued. RequestRedraw is safe.
func NewHub(wake func()) *Hub {
	return &Hub{wake: wake, asks: make(chan Request, 8)}
}

// Take returns one queued call, if the editor has one waiting.
func (h *Hub) Take() (Request, bool) {
	if h == nil {
		return Request{}, false
	}
	select {
	case r := <-h.asks:
		return r, true
	default:
		return Request{}, false
	}
}

// Read asks the editor for the focused document.
func (h *Hub) Read(ctx context.Context, in ReadIn) (Snapshot, error) {
	resp, err := h.round(ctx, Request{Kind: KindRead, Read: in})
	if err != nil {
		return Snapshot{}, err
	}
	if resp.Err != nil {
		return Snapshot{}, resp.Err
	}
	return resp.Snap, nil
}

// Edit asks the editor to change the focused buffer.
func (h *Hub) Edit(ctx context.Context, in ApplyIn) (ApplyOut, error) {
	resp, err := h.round(ctx, Request{Kind: KindEdit, Edit: in})
	if err != nil {
		return ApplyOut{}, err
	}
	if resp.Err != nil {
		return ApplyOut{}, resp.Err
	}
	return resp.Applied, nil
}

func (h *Hub) round(ctx context.Context, req Request) (Response, error) {
	if h == nil {
		return Response{}, errors.New("mcp is not running")
	}
	ctx, cancel := context.WithTimeout(ctx, answerWait)
	defer cancel()
	req.reply = make(chan Response, 1)
	select {
	case h.asks <- req:
	case <-ctx.Done():
		return Response{}, answerErr(ctx.Err())
	}
	if h.wake != nil {
		h.wake()
	}
	select {
	case resp := <-req.reply:
		return resp, nil
	case <-ctx.Done():
		return Response{}, answerErr(ctx.Err())
	}
}

func answerErr(err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		return errors.New("editor did not answer")
	}
	return err
}
