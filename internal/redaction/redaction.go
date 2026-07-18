package redaction

import (
	"bytes"
	"io"
	"strings"
	"sync"
)

const marker = "[REDACTED]"

func Redact(text string, values []string) string {
	if text == "" {
		return text
	}

	for _, value := range values {
		if len(value) < 3 {
			continue
		}
		text = strings.ReplaceAll(text, value, marker)
	}

	return text
}

// Writer redacts configured byte sequences even when a process splits a
// secret across multiple writes. Call Close to flush the protected tail.
type Writer struct {
	mu      sync.Mutex
	dst     io.Writer
	values  [][]byte
	pending []byte
}

func NewWriter(dst io.Writer, values [][]byte) *Writer {
	w := &Writer{dst: dst}
	for _, value := range values {
		if len(value) < 3 {
			continue
		}
		w.values = append(w.values, append([]byte(nil), value...))
	}
	return w
}

func (w *Writer) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.pending = append(w.pending, p...)
	flush := len(w.pending) - w.partialSuffixLength()
	if flush <= 0 {
		return len(p), nil
	}
	if _, err := w.dst.Write(w.redact(w.pending[:flush])); err != nil {
		return 0, err
	}
	w.pending = append(w.pending[:0], w.pending[flush:]...)
	return len(p), nil
}

func (w *Writer) partialSuffixLength() int {
	longest := 0
	for _, value := range w.values {
		max := len(value) - 1
		if max > len(w.pending) {
			max = len(w.pending)
		}
		for length := max; length > longest; length-- {
			if bytes.Equal(w.pending[len(w.pending)-length:], value[:length]) {
				longest = length
				break
			}
		}
	}
	return longest
}

func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.pending) > 0 {
		if _, err := w.dst.Write(w.redact(w.pending)); err != nil {
			return err
		}
		w.pending = nil
	}
	for _, value := range w.values {
		for i := range value {
			value[i] = 0
		}
	}
	return nil
}

func (w *Writer) redact(data []byte) []byte {
	result := append([]byte(nil), data...)
	for _, value := range w.values {
		result = bytes.ReplaceAll(result, value, []byte(marker))
	}
	return result
}
