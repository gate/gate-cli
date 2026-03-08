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
	// api_key must be left untouched.
	assert.Contains(t, out, "api_key: mykey")
}

func TestMaskSecrets_PreservesIndentation(t *testing.T) {
	input := "    api_secret: supersecret"
	out := maskSecrets(input)
	assert.Equal(t, "    api_secret: ****", out)
}

func TestMaskSecrets_NoSecretField(t *testing.T) {
	input := "api_key: mykey\nbase_url: https://example.com"
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
	assert.Contains(t, out, "api_key: prodkey")
	assert.Contains(t, out, "api_key: testkey")
}

func TestMaskSecrets_EmptySecret(t *testing.T) {
	// Empty value should still be masked (no accidental leaks of empty-string marker).
	input := "    api_secret: \"\""
	out := maskSecrets(input)
	assert.Equal(t, "    api_secret: ****", out)
}
