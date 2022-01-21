package hindsight

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"time"
)

// the salt is deterministic with the same random data
// in the configuration file.
// NB config initialisation should write back the config file
// with random randomness if non present, so we use the same
// value between restarts
func dailySalt(cfg *Config, t time.Time) []byte {
	// day since unix epoch
	u := t.Unix()
	// u -= u % 86400 // truncate to a day
	u /= 86400 // this should round down to a day without having to truncate
	// we won't make the hash a KDF or anything hard to calculate, as we
	// don't cache it yet
	h := sha256.New()
	fmt.Fprintf(h, "%s:%d", cfg.RandomSaltSeed, u)
	return h.Sum(nil)
}

func UniqueKey(cfg *Config, in *InboundEvent) string {
	salt := dailySalt(cfg, in.Time)
	key := sha256.New()
	fmt.Fprintf(key, "%s\n%s\n%s\n", in.Host, in.IP, in.UserAgent)
	key.Write(salt)
	unique := key.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(unique)
}
