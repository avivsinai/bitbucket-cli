//go:build !darwin

package secret

func withDarwinKeychainLock(fn func() error) error {
	return fn()
}
