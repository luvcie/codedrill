package music

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"time"
)

type Command string

const (
	GetStatus      Command = "status"
	Previous       Command = "previous"
	Next           Command = "next"
	ToggleLoop     Command = "loop"
	ToggleShuffle  Command = "shuffle"
	NextFilter     Command = "filter-next"
	PreviousFilter Command = "filter-previous"
)

var ErrNoPlayer = errors.New("no music is playing")

type controlResponse struct {
	State PlaybackState `json:"state"`
	Error string        `json:"error,omitempty"`
}

type controlRequest struct {
	command Command
	reply   chan controlResponse
}

// Control connects only to the player belonging to this drill.
func Control(socket string, command Command) (PlaybackState, error) {
	if socket == "" {
		return PlaybackState{}, ErrNoPlayer
	}
	conn, err := net.DialTimeout("unix", socket, time.Second)
	if err != nil {
		return PlaybackState{}, ErrNoPlayer
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(time.Second))
	if err := json.NewEncoder(conn).Encode(command); err != nil {
		return PlaybackState{}, err
	}
	var response controlResponse
	if err := json.NewDecoder(io.LimitReader(conn, 4096)).Decode(&response); err != nil {
		return PlaybackState{}, err
	}
	if response.Error != "" {
		return response.State, errors.New(response.Error)
	}
	return response.State, nil
}

func serveControls(parent context.Context, socket string, requests chan<- controlRequest) (func(), error) {
	if socket == "" {
		return func() {}, nil
	}
	listener, err := net.Listen("unix", socket)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(parent)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			func() {
				defer conn.Close()
				_ = conn.SetDeadline(time.Now().Add(time.Second))
				var command Command
				if json.NewDecoder(io.LimitReader(conn, 4096)).Decode(&command) != nil {
					return
				}
				request := controlRequest{command: command, reply: make(chan controlResponse, 1)}
				select {
				case requests <- request:
				case <-ctx.Done():
					return
				}
				select {
				case response := <-request.reply:
					_ = json.NewEncoder(conn).Encode(response)
				case <-ctx.Done():
				}
			}()
		}
	}()
	return func() {
		cancel()
		_ = listener.Close()
		<-done
	}, nil
}
