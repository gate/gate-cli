package configcmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- maskSecrets (P2-1) ---

func TestMaskSecrets_BasicYAML(t *testing.T) {
	input := `profiles:
  default:
    api_key: mykey
    api_secret: mysecret`
	out := maskSecrets(input)
	assert.Contains(t, out, "api_secret: ****")
	assert.NotContains(t, out, "mysecret")
	assert.Contains(t, out, "api_key: ****")
	assert.NotContains(t, out, "mykey")
}

func TestMaskSecrets_PreservesIndentation(t *testing.T) {
	input := "    api_secret: supersecret"
	out := maskSecrets(input)
	assert.Equal(t, "    api_secret: ****", out)
}

func TestMaskSecrets_NoCredentialFields(t *testing.T) {
	input := "base_url: https://example.com\ndefault_settle: usdt"
	out := maskSecrets(input)
	assert.Equal(t, input, out)
}

func TestMaskSecrets_MultipleProfiles(t *testing.T) {
	input := `profiles:
  prod:
    api_key: prodkey
    api_secret: prodsecret
  test:
    api_key: testkey
    api_secret: testsecret`
	out := maskSecrets(input)
	assert.NotContains(t, out, "prodsecret")
	assert.NotContains(t, out, "testsecret")
	assert.NotContains(t, out, "prodkey")
	assert.NotContains(t, out, "testkey")
}

func TestMaskSecrets_APIKeyMasked(t *testing.T) {
	input := "    api_key: abc123"
	out := maskSecrets(input)
	assert.Equal(t, "    api_key: ****", out)
	assert.NotContains(t, out, "abc123")
}

func TestMaskSecrets_EmptySecret(t *testing.T) {
	// Empty value should still be masked (no accidental leaks of empty-string marker).
	input := "    api_secret: \"\""
	out := maskSecrets(input)
	assert.Equal(t, "    api_secret: ****", out)
}
