package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/gate/gate-cli/internal/config"
)

// TestUpdateFile_PreservesComments 覆盖本次最核心的诉求：
// 带注释的 config.yaml 经过 UpdateFile 修改一个子节点后，注释原样保留。
func TestUpdateFile_PreservesComments(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	orig := `default_profile: sample
default_settle: usdt

# base_url 可选，省略时默认 https://api.gateio.ws
# 测试环境: http://10.0.0.1:18080
# 预发环境: http://192.0.2.10:18081/
# 正式环境: https://api.gateio.ws/
base_url: http://192.0.2.10:18081/

oauth2:
    # discovery_url (RFC 8414 authorization-server metadata)
    # 测试环境: http://dev.example.test/.well-known/oauth-authorization-server
    # 预发环境: https://staging.example.test/.well-known/oauth-authorization-server
    # 正式环境: https://www.gate.com/.well-known/oauth-authorization-server
    discovery_url: https://staging.example.test/.well-known/oauth-authorization-server
`
	require.NoError(t, os.WriteFile(path, []byte(orig), 0600))

	err := config.UpdateFile(path, func(root *yaml.Node) error {
		return config.SetScalarAtPath(root, []string{"base_url"}, "https://api.gateio.ws/")
	}, 0600)
	require.NoError(t, err)

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	out := string(got)

	// 注释区块全部保留
	assert.Contains(t, out, "# base_url 可选")
	assert.Contains(t, out, "# 测试环境: http://10.0.0.1:18080")
	assert.Contains(t, out, "# 预发环境: http://192.0.2.10:18081/")
	assert.Contains(t, out, "# discovery_url (RFC 8414 authorization-server metadata)")
	assert.Contains(t, out, "# 正式环境: https://www.gate.com/.well-known/oauth-authorization-server")

	// 目标值被改写
	assert.Contains(t, out, "base_url: https://api.gateio.ws/")

	// 未触及字段原样
	assert.Contains(t, out, "default_profile: sample")
	assert.Contains(t, out, "discovery_url: https://staging.example.test/")
}

// TestUpdateFile_CreatesMissingFile 空目录，直接写一个带嵌套 key 的新文件。
func TestUpdateFile_CreatesMissingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "config.yaml")

	err := config.UpdateFile(path, func(root *yaml.Node) error {
		return config.SetScalarAtPath(root, []string{"oauth2", "discovery_url"}, "https://www.gate.com/.well-known/oauth-authorization-server")
	}, 0600)
	require.NoError(t, err)

	info, err := os.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0600), info.Mode().Perm())

	data, _ := os.ReadFile(path)
	assert.Contains(t, string(data), "oauth2:")
	assert.Contains(t, string(data), "discovery_url: https://www.gate.com/")
}

// TestUpdateFile_Atomic 简单验证：mutate 返回错误时目标文件保持原样。
func TestUpdateFile_MutateErrorPreservesOriginal(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte("k: v\n"), 0600))

	err := config.UpdateFile(path, func(root *yaml.Node) error {
		return assertHelperErr()
	}, 0600)
	require.Error(t, err)

	got, _ := os.ReadFile(path)
	assert.Equal(t, "k: v\n", string(got))
}

// TestEncodeInto_PreservesHeadComment EncodeInto 把一个新 struct 塞进已有节点，
// 节点上的 HeadComment 应保留。
func TestEncodeInto_PreservesHeadComment(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	orig := `oauth2:
    # 这是 tokens 区块的说明
    tokens: []
`
	require.NoError(t, os.WriteFile(path, []byte(orig), 0600))

	type entry struct {
		DiscoveryURL string `yaml:"discovery_url"`
		ClientID     string `yaml:"client_id"`
	}

	err := config.UpdateFile(path, func(root *yaml.Node) error {
		oauth2Node, err := config.EnsureMapChild(root, "oauth2")
		if err != nil {
			return err
		}
		_, tokensNode := config.FindChild(oauth2Node, "tokens")
		require.NotNil(t, tokensNode)
		return config.EncodeInto(tokensNode, []entry{{DiscoveryURL: "https://x/", ClientID: "c1"}})
	}, 0600)
	require.NoError(t, err)

	got, _ := os.ReadFile(path)
	out := string(got)
	assert.Contains(t, out, "# 这是 tokens 区块的说明")
	assert.Contains(t, out, "discovery_url: https://x/")
	assert.Contains(t, out, "client_id: c1")
}

func TestDeleteKey(t *testing.T) {
	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("a: 1\nb: 2\n"), &root))
	m := root.Content[0]
	assert.True(t, config.DeleteKey(m, "a"))
	assert.False(t, config.DeleteKey(m, "missing"))

	out, err := yaml.Marshal(&root)
	require.NoError(t, err)
	assert.Equal(t, "b: 2\n", strings.TrimSpace(string(out))+"\n")
}

// assertHelperErr — pure error sentinel used by the mutate-failure test.
func assertHelperErr() error { return errHelper }

var errHelper = &helperErr{}

type helperErr struct{}

func (e *helperErr) Error() string { return "forced" }
