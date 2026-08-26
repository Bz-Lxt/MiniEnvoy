package admin

import (
	"crypto/hmac"
	"net/http"
	"strings"
)

func tokenOK(r *http.Request, want string) bool {
	if want == "" {
		return true
	}
	got := r.Header.Get("X-Admin-Token")
	if got == "" {
		if a := r.Header.Get("Authorization"); strings.HasPrefix(a, "Bearer ") {
			got = strings.TrimPrefix(a, "Bearer ")
		}
	}
	return hmac.Equal([]byte(got), []byte(want))
}
