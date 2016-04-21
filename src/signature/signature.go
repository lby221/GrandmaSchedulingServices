package sellyxsign

import (
	"conf"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strconv"
	"time"
)

func VToken(method, time, path string) string {
	str_to_sign := method + "\n" + time + "\n" + path
	key := []byte(conf.GetRestSecret())
	h := hmac.New(sha256.New, key)
	h.Write([]byte(str_to_sign))
	hash := hex.EncodeToString(h.Sum(nil))
	return base64.StdEncoding.EncodeToString([]byte(hash))
}

func TimeStamp() string {
	now := time.Now().UnixNano()
	now = now / 1000000
	strnow := strconv.FormatInt(now, 10)

	return strnow
}
