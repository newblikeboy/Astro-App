package platform

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// id creates a cryptographically random, prefixed record identifier and fails closed if randomness is unavailable.
func id(prefix string) string {
	b := make([]byte, 18)
	if _, e := rand.Read(b); e != nil {
		panic("secure random unavailable")
	}
	return prefix + "_" + hex.EncodeToString(b)
}

// hash returns a SHA-256 digest for stored token identifiers and deterministic internal keys.
func hash(s string) string { v := sha256.Sum256([]byte(s)); return hex.EncodeToString(v[:]) }

// signature creates the hexadecimal HMAC-SHA256 used for authenticated provider messages.
func signature(secret, message string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

// verifySignature decodes and compares a supplied HMAC using constant-time authentication.
func verifySignature(secret, message, provided string) bool {
	expected, e := hex.DecodeString(signature(secret, message))
	if e != nil {
		return false
	}
	actual, e := hex.DecodeString(provided)
	return e == nil && hmac.Equal(expected, actual)
}

// same compares equal-length secret strings using a constant-time primitive.
func same(a, b string) bool { return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1 }

var phonePattern = regexp.MustCompile(`^\+[1-9][0-9]{7,14}$`)

// normalizePhone normalizes customer phone input and validates its international-number format.
func normalizePhone(s string) (string, error) {
	s = strings.ReplaceAll(strings.TrimSpace(s), " ", "")
	if len(s) == 10 {
		s = "+91" + s
	}
	if !phonePattern.MatchString(s) {
		return "", fmt.Errorf("देश कोड सहित सही मोबाइल नंबर लिखें")
	}
	return s, nil
}

// totp checks an administrator's time-based one-time password against the configured secret.
func totp(secret, code string, now time.Time) bool {
	key, e := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(strings.TrimSpace(secret)))
	if e != nil || len(key) < 16 {
		return false
	}
	for delta := -1; delta <= 1; delta++ {
		var b [8]byte
		binary.BigEndian.PutUint64(b[:], uint64(now.Unix()/30+int64(delta)))
		mac := hmac.New(sha1.New, key)
		mac.Write(b[:])
		v := mac.Sum(nil)
		off := v[len(v)-1] & 15
		n := (binary.BigEndian.Uint32(v[off:off+4]) & 0x7fffffff) % 1000000
		if same(fmt.Sprintf("%06d", n), code) {
			return true
		}
	}
	return false
}

// audit appends an actor, action and target entry to the transactional audit log.
func audit(s *State, actor, action, ref string) {
	s.Audits = append(s.Audits, Audit{time.Now().UTC(), actor, action, ref})
}

// enqueue adds an idempotent background job for the requested entity version and payload.
func enqueue(s *State, kind, ref string, version int, payload string) {
	key := fmt.Sprintf("%s:%s:%d", kind, ref, version)
	if s.Jobs[key] == nil {
		s.Jobs[key] = &Job{ID: key, Kind: kind, Ref: ref, Version: version, Status: "pending", Payload: payload}
	}
}
