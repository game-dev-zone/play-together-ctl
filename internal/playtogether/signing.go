// Package playtogether 是玩聚 SDK 的 Go 伺服器端版本，給遊戲伺服器使用。
// 規格見 docs/sdk.md 與 docs/game-integration.md「伺服器回報」。
// 簽章規則必須和 Node 版（packages/sdk-server）完全一致，以 test-vectors/signing.json 驗證。
package playtogether

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

// SignatureScheme 是簽章字串的版本標記。
const SignatureScheme = "PT-SIG-1"

// 請求標頭名稱。
const (
	HeaderKeyID          = "X-PT-Key-Id"
	HeaderTimestamp      = "X-PT-Timestamp"
	HeaderNonce          = "X-PT-Nonce"
	HeaderSignature      = "X-PT-Signature"
	HeaderIdempotencyKey = "Idempotency-Key"
)

// SignableRequest 是要簽章的請求內容。
type SignableRequest struct {
	Method string
	Path   string
	// Query 的值依編碼後的結果排序，順序不影響簽章。
	Query map[string][]string
	Body  []byte
}

// encodeURIComponent 和 JavaScript 的 encodeURIComponent 結果相同：
// 只保留 A-Z a-z 0-9 - _ . ! ~ * ' ( )，其餘以 UTF-8 位元組編碼成大寫 %XX。
func encodeURIComponent(s string) string {
	const hex = "0123456789ABCDEF"
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if ('A' <= c && c <= 'Z') || ('a' <= c && c <= 'z') || ('0' <= c && c <= '9') ||
			strings.IndexByte("-_.!~*'()", c) >= 0 {
			b.WriteByte(c)
			continue
		}
		b.WriteByte('%')
		b.WriteByte(hex[c>>4])
		b.WriteByte(hex[c&15])
	}
	return b.String()
}

// 先編碼再排序，編碼後只剩 ASCII，和 Node 版排序結果一致。
func canonicalQuery(query map[string][]string) string {
	type pair struct{ k, v string }
	var pairs []pair
	for k, vs := range query {
		for _, v := range vs {
			pairs = append(pairs, pair{encodeURIComponent(k), encodeURIComponent(v)})
		}
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].k == pairs[j].k {
			return pairs[i].v < pairs[j].v
		}
		return pairs[i].k < pairs[j].k
	})
	parts := make([]string, len(pairs))
	for i, p := range pairs {
		parts[i] = p.k + "=" + p.v
	}
	return strings.Join(parts, "&")
}

// CanonicalString 產生簽章字串：方法、路徑、查詢、時間戳記、nonce、內容雜湊，每項一行。
func CanonicalString(req SignableRequest, timestamp, nonce string) string {
	sum := sha256.Sum256(req.Body)
	return strings.Join([]string{
		SignatureScheme,
		strings.ToUpper(req.Method),
		req.Path,
		canonicalQuery(req.Query),
		timestamp,
		nonce,
		base64.RawURLEncoding.EncodeToString(sum[:]),
	}, "\n")
}

// NewNonce 產生隨機 nonce。
func NewNonce() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

// SignRequest 回傳要加在請求上的簽章標頭。timestampMs 為 Unix 毫秒。
func SignRequest(req SignableRequest, keyID string, key ed25519.PrivateKey, timestampMs int64, nonce string) http.Header {
	ts := strconv.FormatInt(timestampMs, 10)
	sig := ed25519.Sign(key, []byte(CanonicalString(req, ts, nonce)))
	h := http.Header{}
	h.Set(HeaderKeyID, keyID)
	h.Set(HeaderTimestamp, ts)
	h.Set(HeaderNonce, nonce)
	h.Set(HeaderSignature, base64.RawURLEncoding.EncodeToString(sig))
	return h
}
