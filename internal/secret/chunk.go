package secret

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/99designs/keyring"
)

// winCredMaxBlobSize is CRED_MAX_CREDENTIAL_BLOB_SIZE (5 * 512).
// A single CredWriteW payload larger than this fails with RPC_X_BAD_STUB_DATA
// ("The stub received bad data.").
const winCredMaxBlobSize = 2560

const (
	chunkedHeaderPrefix = "bkt.chunked.v1:"
	chunkKeySep         = "#c"
	maxSecretChunks     = 256
)

// errSecretChunkTooLarge is returned when a single keyring write would exceed
// the backend blob limit. Callers must not fall back to the file backend.
var errSecretChunkTooLarge = errors.New("secret chunk exceeds backend blob size limit")

func chunkSizeForBackends(backends []keyring.BackendType) int {
	for _, backend := range backends {
		if backend == keyring.WinCredBackend {
			return winCredMaxBlobSize
		}
	}
	return 0
}

func chunkKey(key string, i int) string {
	return key + chunkKeySep + strconv.Itoa(i)
}

func encodeChunkHeader(n int) []byte {
	return []byte(chunkedHeaderPrefix + strconv.Itoa(n))
}

// parseChunkHeader returns the chunk count when data is a chunked-secret
// header. A count of 0 means data is a regular (unchunked) value.
func parseChunkHeader(data []byte) (int, error) {
	s := string(data)
	if !strings.HasPrefix(s, chunkedHeaderPrefix) {
		return 0, nil
	}
	n, err := strconv.Atoi(s[len(chunkedHeaderPrefix):])
	if err != nil || n < 1 || n > maxSecretChunks {
		return 0, fmt.Errorf("invalid chunked secret header")
	}
	return n, nil
}

func splitChunks(data []byte, size int) [][]byte {
	if len(data) == 0 {
		return [][]byte{data}
	}
	if size <= 0 {
		return [][]byte{data}
	}
	n := (len(data) + size - 1) / size
	out := make([][]byte, 0, n)
	for len(data) > 0 {
		if size > len(data) {
			size = len(data)
		}
		out = append(out, data[:size])
		data = data[size:]
	}
	return out
}

func (s *Store) writeItem(key string, data []byte) error {
	if s.maxChunkSize > 0 && len(data) > s.maxChunkSize {
		return fmt.Errorf("%w: %d bytes (limit %d)", errSecretChunkTooLarge, len(data), s.maxChunkSize)
	}
	return s.kr.Set(keyring.Item{
		Key:   key,
		Data:  data,
		Label: fmt.Sprintf("bkt %s", key),
	})
}

func (s *Store) setValue(key, value string) error {
	data := []byte(value)
	limit := s.maxChunkSize
	if limit <= 0 || len(data) <= limit {
		if err := s.writeItem(key, data); err != nil {
			return err
		}
		if s.maxChunkSize > 0 {
			return s.removeChunksFrom(key, 0)
		}
		return nil
	}

	chunks := splitChunks(data, limit)
	if len(chunks) > maxSecretChunks {
		return fmt.Errorf("%w: value needs %d chunks (limit %d)", errSecretChunkTooLarge, len(chunks), maxSecretChunks)
	}

	for i, chunk := range chunks {
		if err := s.writeItem(chunkKey(key, i), chunk); err != nil {
			return err
		}
	}
	if err := s.writeItem(key, encodeChunkHeader(len(chunks))); err != nil {
		return err
	}
	return s.removeChunksFrom(key, len(chunks))
}

func (s *Store) getValue(key string) (string, error) {
	item, err := s.kr.Get(key)
	if err != nil {
		return "", err
	}

	n, err := parseChunkHeader(item.Data)
	if err != nil {
		return "", err
	}
	if n == 0 {
		return string(item.Data), nil
	}

	var buf []byte
	for i := 0; i < n; i++ {
		part, getErr := s.kr.Get(chunkKey(key, i))
		if getErr != nil {
			return "", fmt.Errorf("read secret chunk %d: %w", i, getErr)
		}
		buf = append(buf, part.Data...)
	}
	return string(buf), nil
}

func (s *Store) deleteValue(key string) error {
	n := 0
	if item, err := s.kr.Get(key); err == nil {
		count, parseErr := parseChunkHeader(item.Data)
		if parseErr == nil {
			n = count
		}
	}

	if n > 0 {
		if err := s.removeKnownChunks(key, n); err != nil {
			return err
		}
	} else if s.maxChunkSize > 0 {
		if err := s.removeChunksFrom(key, 0); err != nil {
			return err
		}
	}

	err := s.kr.Remove(key)
	if err == nil || errors.Is(err, keyring.ErrKeyNotFound) || errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

func (s *Store) removeKnownChunks(key string, n int) error {
	var first error
	for i := 0; i < n; i++ {
		if err := ignoreMissing(s.kr.Remove(chunkKey(key, i))); err != nil && first == nil {
			first = err
		}
	}
	return first
}

func (s *Store) removeChunksFrom(key string, start int) error {
	for i := start; i < maxSecretChunks; i++ {
		err := s.kr.Remove(chunkKey(key, i))
		if err == nil {
			continue
		}
		if errors.Is(err, keyring.ErrKeyNotFound) || errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	return nil
}

func ignoreMissing(err error) error {
	if err == nil || errors.Is(err, keyring.ErrKeyNotFound) || errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
