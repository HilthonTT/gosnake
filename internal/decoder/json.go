package decoder

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func DoJSON[T any](client *http.Client, req *http.Request) (T, error) {
	var zero T
	resp, err := client.Do(req)
	if err != nil {
		return zero, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return zero, fmt.Errorf("unexpected status %s", resp.Status)
	}

	return DecodeJSON[T](resp.Body)
}

func DecodeJSON[T any](r io.Reader) (T, error) {
	var v T
	if err := json.NewDecoder(r).Decode(&v); err != nil {
		return v, fmt.Errorf("decode response: %w", err)
	}
	return v, nil
}
