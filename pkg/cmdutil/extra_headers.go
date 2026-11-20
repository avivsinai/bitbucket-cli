package cmdutil

import (
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/avivsinai/bitbucket-cli/internal/secret"
)

// envExtraHeaders holds headers added to every request, one "Name: value" per
// line, for example a cookie a proxy in front of the server requires.
const envExtraHeaders = "BKT_EXTRA_HEADERS"

// ExtraHeadersFromEnv returns the BKT_EXTRA_HEADERS headers for a client of the
// server at baseURL. They are sent only to the BKT_HOST server, matched on
// scheme too so an https BKT_HOST never sends them over http. The variable is
// refused without BKT_HOST and yields nil for any other server.
func ExtraHeadersFromEnv(baseURL string) (http.Header, error) {
	raw := os.Getenv(envExtraHeaders)
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	rawHost := strings.TrimSpace(os.Getenv(secret.EnvHost))
	if rawHost == "" {
		return nil, fmt.Errorf("%s is set without BKT_HOST; set BKT_HOST to the server the headers are for", envExtraHeaders)
	}
	envURL, _, err := envBaseURL(rawHost)
	if err != nil {
		return nil, fmt.Errorf("BKT_HOST: %w", err)
	}
	if !sameServer(envURL, baseURL) {
		return nil, nil
	}
	return parseExtraHeaders(raw)
}

func sameServer(a, b string) bool {
	ua, errA := url.Parse(a)
	ub, errB := url.Parse(b)
	if errA != nil || errB != nil || !strings.EqualFold(ua.Scheme, ub.Scheme) {
		return false
	}
	ka, errA := serverKey(a)
	kb, errB := serverKey(b)
	return errA == nil && errB == nil && ka == kb
}

func parseExtraHeaders(raw string) (http.Header, error) {
	headers := http.Header{}
	for i, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		name, value, ok := strings.Cut(line, ":")
		name = strings.TrimSpace(name)
		if !ok || name == "" || strings.ContainsAny(name, " \t") {
			return nil, fmt.Errorf("%s line %d: want \"Name: value\"", envExtraHeaders, i+1)
		}
		if strings.EqualFold(name, "Authorization") {
			return nil, fmt.Errorf("%s line %d: Authorization comes from the stored token or BKT_TOKEN", envExtraHeaders, i+1)
		}
		headers.Add(name, strings.TrimSpace(value))
	}
	return headers, nil
}
