package auth

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/qpubio/qpub-server/internal/api/response"
	"github.com/qpubio/qpub-server/internal/application/dto"
	projectAPIKey "github.com/qpubio/qpub-server/internal/domain/apikey"
	"github.com/qpubio/qpub-server/internal/shared/apikey"

	"github.com/gin-gonic/gin"
)

const defaultTimestampTolerance = 5 * time.Minute
const tokenRequestTimestampToleranceEnv = "TOKEN_REQUEST_TIMESTAMP_TOLERANCE"

// TokenRequestAuth middleware authenticates requests using a canonical-string HMAC signature.
// The trusted server builds a deterministic canonical string from the payload fields and signs it
// with the API secret key using HMAC-SHA256 (Base64 encoded). The untrusted client forwards the
// payload unchanged; the server reproduces the canonical string from parsed fields, so signing is
// independent of JSON key ordering or formatting.
func TokenRequestAuth(apiKeyService projectAPIKey.Service, apiKeyParser *apikey.Parser) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req dto.TokenRequestBody
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, "Invalid token request body")
			c.Abort()
			return
		}

		apiKeyID, err := apiKeyParser.ParseComponents(req.AKI)
		if err != nil {
			response.BadRequest(c, "Invalid API key identifier (aki)")
			c.Abort()
			return
		}

		storedKey, err := apiKeyService.Get(apiKeyID)
		if err != nil {
			response.Unauthorized(c, "API key not found")
			c.Abort()
			return
		}

		if storedKey.Status != projectAPIKey.StatusActive {
			response.Unauthorized(c, "API key is not active")
			c.Abort()
			return
		}

		keyIDParam := c.Param("keyID")
		// Route keyID must be API key public ID (not full aki).
		if keyIDParam == "" || keyIDParam != storedKey.PublicID {
			response.Unauthorized(c, "Invalid API key")
			c.Abort()
			return
		}

		tolerance := timestampTolerance()
		now := time.Now().Unix()
		diff := now - req.Timestamp
		if diff < 0 {
			diff = -diff
		}
		if diff > int64(tolerance.Seconds()) {
			response.BadRequest(c, "Token request has expired or has an invalid timestamp")
			c.Abort()
			return
		}

		canonical, err := buildCanonicalTokenRequestString(req)
		if err != nil {
			response.BadRequest(c, "Invalid token request payload")
			c.Abort()
			return
		}

		if !verifyCanonicalSignature(canonical, []byte(storedKey.SecretKey), req.Signature) {
			response.Unauthorized(c, "Invalid token request signature")
			c.Abort()
			return
		}

		c.Set("apiKeyID", &storedKey.ID)
		c.Set("apiKeyPublicID", &storedKey.PublicID)
		c.Set("apiPublicKey", &storedKey.PublicID)
		c.Set("apiSecretKey", &storedKey.SecretKey)
		c.Set("projectID", &storedKey.ProjectID)

		if len(req.Permission) > 0 {
			c.Set("permission", &req.Permission)
		} else {
			c.Set("permission", &storedKey.Permission)
		}

		c.Set("alias", &req.Alias)
		c.Next()
	}
}

// buildCanonicalTokenRequestString assembles the deterministic signable string from the parsed
// request fields. Fields are emitted in a fixed order, one per line, separated by LF. Optional
// fields (alias, permission) are omitted entirely when empty.
func buildCanonicalTokenRequestString(req dto.TokenRequestBody) (string, error) {
	lines := make([]string, 0, 4)
	lines = append(lines, "aki="+req.AKI)
	lines = append(lines, "timestamp="+strconv.FormatInt(req.Timestamp, 10))

	if req.Alias != "" {
		lines = append(lines, "alias="+req.Alias)
	}

	if len(req.Permission) > 0 {
		permission, err := canonicalJSON(req.Permission)
		if err != nil {
			return "", err
		}
		lines = append(lines, "permission="+permission)
	}

	return strings.Join(lines, "\n"), nil
}

// canonicalJSON returns a deterministic JSON encoding with object keys sorted recursively and no
// insignificant whitespace. Arrays keep their original order.
func canonicalJSON(raw json.RawMessage) (string, error) {
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := writeCanonicalJSON(&buf, value); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func writeCanonicalJSON(buf *bytes.Buffer, value any) error {
	switch typed := value.(type) {
	case map[string]any:
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)

		buf.WriteByte('{')
		for i, key := range keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			encodedKey, err := json.Marshal(key)
			if err != nil {
				return err
			}
			buf.Write(encodedKey)
			buf.WriteByte(':')
			if err := writeCanonicalJSON(buf, typed[key]); err != nil {
				return err
			}
		}
		buf.WriteByte('}')
		return nil
	case []any:
		buf.WriteByte('[')
		for i, item := range typed {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := writeCanonicalJSON(buf, item); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
		return nil
	default:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return err
		}
		buf.Write(encoded)
		return nil
	}
}

// verifyCanonicalSignature compares an HMAC-SHA256 signature over the canonical string against
// the Base64-encoded signature supplied by the client.
func verifyCanonicalSignature(canonical string, secret []byte, signature string) bool {
	normalized := strings.TrimSpace(signature)
	if normalized == "" {
		return false
	}

	decoded, err := base64.StdEncoding.DecodeString(normalized)
	if err != nil || len(decoded) != sha256.Size {
		return false
	}

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(canonical))
	expected := mac.Sum(nil)

	return hmac.Equal(expected, decoded)
}

func timestampTolerance() time.Duration {
	raw := os.Getenv(tokenRequestTimestampToleranceEnv)
	if raw == "" {
		return defaultTimestampTolerance
	}

	if duration, err := time.ParseDuration(raw); err == nil && duration > 0 {
		return duration
	}

	return defaultTimestampTolerance
}
