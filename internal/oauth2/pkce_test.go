package oauth2

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateVerifier(t *testing.T) {
	v, err := GenerateVerifier()
	require.NoError(t, err)
	assert.Len(t, v, 43, "base64url(32 bytes) = 43 chars")

	// RFC 7636: verifier chars must be [A-Z][a-z][0-9]-._~
	for _, c := range v {
		assert.True(t, isUnreserved(c), "invalid char in verifier: %c", c)
	}

	// Each call should produce different output
	v2, _ := GenerateVerifier()
	assert.NotEqual(t, v, v2)
}

func TestGenerateChallenge(t *testing.T) {
	// RFC 7636 Appendix B test vector
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	challenge := GenerateChallenge(verifier)
	assert.Equal(t, "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM", challenge)

	// Challenge must be valid base64url
	_, err := base64.RawURLEncoding.DecodeString(challenge)
	assert.NoError(t, err)
}

func isUnreserved(c rune) bool {
	return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') ||
		(c >= '0' && c <= '9') || c == '-' || c == '.' || c == '_' || c == '~'
}
