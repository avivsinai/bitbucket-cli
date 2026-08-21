package secret

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/99designs/keyring"
)

type recordedWrite struct {
	key  string
	size int
}

// memoryKeyring is an in-memory keyring that rejects blobs larger than limit,
// standing in for WinCred's CRED_MAX_CREDENTIAL_BLOB_SIZE check.
type memoryKeyring struct {
	items map[string][]byte
	limit int
	sets  []recordedWrite
}

func newMemoryKeyring(limit int) *memoryKeyring {
	return &memoryKeyring{items: make(map[string][]byte), limit: limit}
}

func (m *memoryKeyring) Get(key string) (keyring.Item, error) {
	data, ok := m.items[key]
	if !ok {
		return keyring.Item{}, keyring.ErrKeyNotFound
	}
	return keyring.Item{Key: key, Data: append([]byte(nil), data...)}, nil
}

func (m *memoryKeyring) GetMetadata(string) (keyring.Metadata, error) {
	return keyring.Metadata{}, keyring.ErrMetadataNotSupported
}

func (m *memoryKeyring) Set(item keyring.Item) error {
	m.sets = append(m.sets, recordedWrite{key: item.Key, size: len(item.Data)})
	if m.limit > 0 && len(item.Data) > m.limit {
		return errors.New("The stub received bad data")
	}
	m.items[item.Key] = append([]byte(nil), item.Data...)
	return nil
}

func (m *memoryKeyring) Remove(key string) error {
	if _, ok := m.items[key]; !ok {
		return keyring.ErrKeyNotFound
	}
	delete(m.items, key)
	return nil
}

func (m *memoryKeyring) Keys() ([]string, error) {
	keys := make([]string, 0, len(m.items))
	for k := range m.items {
		keys = append(keys, k)
	}
	return keys, nil
}

func winCredStore(kr *memoryKeyring) *Store {
	return &Store{kr: kr, maxChunkSize: winCredMaxBlobSize}
}

func TestSplitChunks(t *testing.T) {
	t.Parallel()

	got := splitChunks(bytes.Repeat([]byte("a"), 5558), winCredMaxBlobSize)
	if len(got) != 3 {
		t.Fatalf("chunks = %d, want 3", len(got))
	}
	if len(got[0]) != winCredMaxBlobSize || len(got[1]) != winCredMaxBlobSize {
		t.Fatalf("full chunks = %d, %d, want %d", len(got[0]), len(got[1]), winCredMaxBlobSize)
	}
	if len(got[2]) != 5558-2*winCredMaxBlobSize {
		t.Fatalf("tail chunk = %d, want %d", len(got[2]), 5558-2*winCredMaxBlobSize)
	}
	for i, chunk := range got {
		if len(chunk) > winCredMaxBlobSize {
			t.Fatalf("chunk %d is %d bytes, exceeds %d", i, len(chunk), winCredMaxBlobSize)
		}
	}
}

func TestStoreRoundTripOversizedBlob(t *testing.T) {
	kr := newMemoryKeyring(winCredMaxBlobSize)
	store := winCredStore(kr)
	key := TokenKey("bitbucket.org")
	want := strings.Repeat("A", 5558)

	if err := store.Set(key, want); err != nil {
		t.Fatalf("Set: %v", err)
	}
	for _, w := range kr.sets {
		if w.size > winCredMaxBlobSize {
			t.Fatalf("wrote %q at %d bytes; CredWriteW limit is %d", w.key, w.size, winCredMaxBlobSize)
		}
	}
	if len(kr.items) < 2 {
		t.Fatalf("expected chunked write, got %d items", len(kr.items))
	}

	got, err := store.Get(key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != want {
		t.Fatalf("Get reconstructed %d bytes, want %d", len(got), len(want))
	}
}

func TestStoreExactLimitIsSingleItem(t *testing.T) {
	kr := newMemoryKeyring(winCredMaxBlobSize)
	store := winCredStore(kr)
	key := TokenKey("bitbucket.org")
	want := strings.Repeat("B", winCredMaxBlobSize)

	if err := store.Set(key, want); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if _, ok := kr.items[key]; !ok {
		t.Fatal("expected primary key")
	}
	if _, ok := kr.items[chunkKey(key, 0)]; ok {
		t.Fatal("value at the blob limit must not be chunked")
	}

	got, err := store.Get(key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != want {
		t.Fatalf("Get = %d bytes, want %d", len(got), len(want))
	}
}

func TestStoreSmallValueUnchunked(t *testing.T) {
	kr := newMemoryKeyring(winCredMaxBlobSize)
	store := winCredStore(kr)
	key := TokenKey("bitbucket.org")
	const want = "scoped-api-token"

	if err := store.Set(key, want); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if len(kr.items) != 1 {
		t.Fatalf("small value should write one item, got %d", len(kr.items))
	}

	got, err := store.Get(key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != want {
		t.Fatalf("Get = %q, want %q", got, want)
	}
}

func TestStoreDeleteRemovesAllChunks(t *testing.T) {
	kr := newMemoryKeyring(winCredMaxBlobSize)
	store := winCredStore(kr)
	key := TokenKey("bitbucket.org")

	if err := store.Set(key, strings.Repeat("C", 5558)); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if err := store.Delete(key); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if len(kr.items) != 0 {
		t.Fatalf("Delete left %d items: %v", len(kr.items), keysOf(kr))
	}
	if _, err := store.Get(key); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Get after Delete: %v, want os.ErrNotExist", err)
	}
}

func keysOf(kr *memoryKeyring) []string {
	keys, _ := kr.Keys()
	return keys
}

func TestStoreDeleteMissingIsOK(t *testing.T) {
	kr := newMemoryKeyring(winCredMaxBlobSize)
	store := winCredStore(kr)
	if err := store.Delete(TokenKey("bitbucket.org")); err != nil {
		t.Fatalf("Delete missing: %v", err)
	}
}

func TestStoreOverwriteLargeWithSmallClearsChunks(t *testing.T) {
	kr := newMemoryKeyring(winCredMaxBlobSize)
	store := winCredStore(kr)
	key := TokenKey("bitbucket.org")

	if err := store.Set(key, strings.Repeat("D", 4000)); err != nil {
		t.Fatalf("Set large: %v", err)
	}
	if err := store.Set(key, "small-token"); err != nil {
		t.Fatalf("Set small: %v", err)
	}
	if len(kr.items) != 1 {
		t.Fatalf("overwrite should leave one item, got %d: %v", len(kr.items), keysOf(kr))
	}
	got, err := store.Get(key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "small-token" {
		t.Fatalf("Get = %q, want small-token", got)
	}
}

func TestStoreLegacyUnchunkedGet(t *testing.T) {
	kr := newMemoryKeyring(winCredMaxBlobSize)
	store := winCredStore(kr)
	key := TokenKey("bitbucket.org")
	if err := kr.Set(keyring.Item{Key: key, Data: []byte("legacy-api-token")}); err != nil {
		t.Fatal(err)
	}

	got, err := store.Get(key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "legacy-api-token" {
		t.Fatalf("Get = %q, want legacy-api-token", got)
	}
}

func TestWriteItemNeverCallsSetWithOversizedBlob(t *testing.T) {
	kr := newMemoryKeyring(winCredMaxBlobSize)
	store := winCredStore(kr)

	err := store.writeItem("host/bitbucket.org/token", bytes.Repeat([]byte("x"), winCredMaxBlobSize+1))
	if !errors.Is(err, errSecretChunkTooLarge) {
		t.Fatalf("error = %v, want %v", err, errSecretChunkTooLarge)
	}
	if len(kr.sets) != 0 {
		t.Fatalf("Set was called %d times; oversized blob must not reach CredWriteW", len(kr.sets))
	}
}

func TestSetDoesNotFallBackToFileOnChunkError(t *testing.T) {
	kr := newMemoryKeyring(winCredMaxBlobSize)
	store := winCredStore(kr)

	// Force the size-aware error path with no second backend to retry.
	err := store.writeItem("k", bytes.Repeat([]byte("y"), 3000))
	if !errors.Is(err, errSecretChunkTooLarge) {
		t.Fatalf("error = %v, want %v", err, errSecretChunkTooLarge)
	}
	if usesFileBackend([]keyring.BackendType{keyring.WinCredBackend}) {
		t.Fatal("WinCred-only path must not include the file backend")
	}
	if _, ok := kr.items["k"]; ok {
		t.Fatal("failed write must not leave a file-backend fallback item")
	}
}

func TestStoreRefreshWriteBackReplacesChunks(t *testing.T) {
	kr := newMemoryKeyring(winCredMaxBlobSize)
	store := winCredStore(kr)
	key := TokenKey("api.bitbucket.org")

	first := strings.Repeat("F", 5558)
	if err := store.Set(key, first); err != nil {
		t.Fatalf("Set first: %v", err)
	}
	second := strings.Repeat("G", 3000)
	if err := store.Set(key, second); err != nil {
		t.Fatalf("Set refresh: %v", err)
	}
	for _, w := range kr.sets {
		if w.size > winCredMaxBlobSize {
			t.Fatalf("refresh write %q was %d bytes", w.key, w.size)
		}
	}
	got, err := store.Get(key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != second {
		t.Fatalf("Get = %d bytes, want %d", len(got), len(second))
	}
	if _, ok := kr.items[chunkKey(key, 2)]; ok {
		t.Fatal("refresh to a smaller blob left an extra chunk")
	}
}

func TestUnlimitedStoreKeepsSingleBlob(t *testing.T) {
	kr := newMemoryKeyring(0)
	store := &Store{kr: kr, maxChunkSize: 0}
	key := TokenKey("bitbucket.org")
	want := strings.Repeat("E", 5558)

	if err := store.Set(key, want); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if len(kr.items) != 1 {
		t.Fatalf("unlimited store should write one item, got %d", len(kr.items))
	}
	got, err := store.Get(key)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != want {
		t.Fatalf("Get = %d bytes, want %d", len(got), len(want))
	}
}
