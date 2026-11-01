package playtogether

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

// MaxClockSkew 是請求時間戳記與平台時間允許的最大差距。
const MaxClockSkew = 5 * time.Minute

// 驗證失敗的原因，和 Node 版 verifyRequest 的錯誤碼對應。
var (
	ErrMissingHeaders = errors.New("missing_headers")
	ErrBadTimestamp   = errors.New("bad_timestamp")
	ErrClockSkew      = errors.New("clock_skew")
	ErrUnknownKey     = errors.New("unknown_key")
	ErrBadSignature   = errors.New("bad_signature")
	ErrReplay         = errors.New("replay")
)

// NonceStore 記錄用過的 nonce。實作必須是原子操作（例如資料庫唯一索引）。
type NonceStore interface {
	// UseOnce 第一次使用回傳 true；已用過回傳 false。
	UseOnce(ctx context.Context, keyID, nonce string, ttl time.Duration) (bool, error)
}

// PublicKeyLookup 依金鑰 ID 取得遊戲伺服器登記的公鑰；找不到回傳 nil。
type PublicKeyLookup func(ctx context.Context, keyID string) (ed25519.PublicKey, error)

var timestampPattern = regexp.MustCompile(`^\d{1,16}$`)

// VerifyRequest 給平台 API 用：驗證遊戲伺服器的請求簽章，成功時回傳金鑰 ID。
// body 必須是收到的原始位元組，不能先解析再重新序列化。
func VerifyRequest(ctx context.Context, req SignableRequest, headers http.Header,
	lookup PublicKeyLookup, nonces NonceStore, now time.Time) (string, error) {
	keyID := headers.Get(HeaderKeyID)
	ts := headers.Get(HeaderTimestamp)
	nonce := headers.Get(HeaderNonce)
	sigText := headers.Get(HeaderSignature)
	if keyID == "" || ts == "" || nonce == "" || sigText == "" {
		return "", ErrMissingHeaders
	}
	if !timestampPattern.MatchString(ts) {
		return "", ErrBadTimestamp
	}
	ms, err := strconv.ParseInt(ts, 10, 64)
	if err != nil {
		return "", ErrBadTimestamp
	}
	diff := now.Sub(time.UnixMilli(ms))
	if diff > MaxClockSkew || diff < -MaxClockSkew {
		return "", ErrClockSkew
	}
	key, err := lookup(ctx, keyID)
	if err != nil {
		return "", err
	}
	if len(key) != ed25519.PublicKeySize {
		return "", ErrUnknownKey
	}
	sig, err := base64.RawURLEncoding.DecodeString(sigText)
	if err != nil || !ed25519.Verify(key, []byte(CanonicalString(req, ts, nonce)), sig) {
		return "", ErrBadSignature
	}
	// 簽章正確後才記錄 nonce，避免別人用假請求把 nonce 佔掉
	fresh, err := nonces.UseOnce(ctx, keyID, nonce, 2*MaxClockSkew)
	if err != nil {
		return "", err
	}
	if !fresh {
		return "", ErrReplay
	}
	return keyID, nil
}
