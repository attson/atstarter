package control

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

type State struct {
	URL     string `json:"url"`
	Token   string `json:"token"`
	PID     int    `json:"pid"`
	Version string `json:"version"`
}

type Request struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Hint    string `json:"hint,omitempty"`
}

type Response struct {
	OK    bool        `json:"ok"`
	Data  interface{} `json:"data,omitempty"`
	Error *Error      `json:"error,omitempty"`
}

type Client struct {
	StatePath string
	HTTP      *http.Client
}

func ReadState(path string) (State, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return State{}, err
	}
	var state State
	if err := json.Unmarshal(b, &state); err != nil {
		return State{}, err
	}
	if state.URL == "" || state.Token == "" {
		return State{}, errors.New("control state is incomplete")
	}
	return state, nil
}

func WriteState(path string, state State) error {
	b, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func (c Client) Call(method string, params interface{}, out interface{}) error {
	state, err := ReadState(c.StatePath)
	if err != nil {
		return AppNotRunningError{}
	}
	var raw json.RawMessage
	if params != nil {
		raw, err = json.Marshal(params)
		if err != nil {
			return err
		}
	}
	reqBody, err := json.Marshal(Request{Method: method, Params: raw})
	if err != nil {
		return err
	}
	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	req, err := http.NewRequest(http.MethodPost, state.URL+"/rpc", bytes.NewReader(reqBody))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+state.Token)
	resp, err := httpClient.Do(req)
	if err != nil {
		return AppNotRunningError{}
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var cr Response
	if err := json.Unmarshal(body, &cr); err != nil {
		return fmt.Errorf("decode control response: %w", err)
	}
	if !cr.OK {
		if cr.Error == nil {
			return RemoteError{Code: "remote_error", Message: "control call failed"}
		}
		return RemoteError{Code: cr.Error.Code, Message: cr.Error.Message, Hint: cr.Error.Hint}
	}
	if out != nil {
		b, err := json.Marshal(cr.Data)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(b, out); err != nil {
			return err
		}
	}
	return nil
}

type AppNotRunningError struct{}

func (AppNotRunningError) Error() string { return "atstarter desktop app is not running" }

type RemoteError struct {
	Code    string
	Message string
	Hint    string
}

func (e RemoteError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return e.Code
}
