// Package config — yamledit.go 提供基于 gopkg.in/yaml.v3 Node AST 的
// YAML 文件原地编辑能力。
//
// 设计目标：
//   - 只改动目标子树，保留所有节点的 head/line/foot comment
//   - 原子写：tmp → fsync → rename → 父目录 fsync（同 oauth2.SaveToken 策略）
//   - 新建文件时初始化为 MappingNode
//
// 典型用法：
//
//	err := config.UpdateFile(path, func(root *yaml.Node) error {
//	    return config.SetScalarAtPath(root, []string{"oauth2", "discovery_url"}, "https://issuer.example.test")
//	}, 0600)
package config

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/flock"

	"gopkg.in/yaml.v3"
)

// UpdateFile 读取 path，应用 mutate 到根节点，再原子写回。
//
// 行为：
//   - 文件不存在 → 以空 MappingNode 作为根；mutate 后写入（目录自动创建）
//   - mutate 返回错误 → 不落盘
//   - 写入前执行 fsync（tmp 文件 + 父目录），与 oauth2.SaveToken 一致
//   - mode 仅在文件首次创建时用于设置权限；后续写入继承原权限但会在 tmp 上再 chmod 一次保险
func UpdateFile(path string, mutate func(root *yaml.Node) error, mode os.FileMode) error {
	doc, err := readDocumentOrEmpty(path)
	if err != nil {
		return err
	}
	root := documentRoot(doc)
	if err := mutate(root); err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	var buf []byte
	buf, err = yaml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshal yaml: %w", err)
	}

	return atomicWrite(dir, path, buf, mode)
}

// readDocumentOrEmpty 返回 path 的 DocumentNode；文件不存在时返回一个内容为
// 空 MappingNode 的新 DocumentNode。
func readDocumentOrEmpty(path string) (*yaml.Node, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return newEmptyDocument(), nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	if len(data) == 0 {
		return newEmptyDocument(), nil
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	// Edge case: empty or comment-only file parses to Kind=0 / no content.
	if doc.Kind == 0 || len(doc.Content) == 0 {
		return newEmptyDocument(), nil
	}
	// Ensure root is a mapping; if top-level scalar/sequence was used (unexpected),
	// we refuse to clobber it silently.
	root := documentRoot(&doc)
	if root.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("%s: top-level YAML must be a mapping (got kind=%d)", path, root.Kind)
	}
	return &doc, nil
}

func newEmptyDocument() *yaml.Node {
	return &yaml.Node{
		Kind: yaml.DocumentNode,
		Content: []*yaml.Node{
			{Kind: yaml.MappingNode, Tag: "!!map"},
		},
	}
}

// documentRoot 返回 DocumentNode 的唯一子节点（通常是 MappingNode）。
func documentRoot(doc *yaml.Node) *yaml.Node {
	if doc.Kind == yaml.DocumentNode && len(doc.Content) > 0 {
		return doc.Content[0]
	}
	return doc
}

// atomicWrite 把 data 原子落盘到 path：CreateTemp → Write → fsync → Close
// → Rename → parent dir fsync。
//
// 安全：若 path 已存在且是 symlink，则拒绝写入（review）。防止攻击者预先把
// ~/.gate-cli/config.yaml 替换为指向系统文件的软链接，通过 gate-cli 的写入
// 间接修改链接目标。os.Lstat 不跟随 symlink。
func atomicWrite(dir, path string, data []byte, mode os.FileMode) error {
	if mode == 0 {
		mode = 0600
	}
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("refusing to write through symlink: %s", path)
		}
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("lstat target: %w", err)
	}
	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".*.tmp")
	if err != nil {
		return fmt.Errorf("create tmp: %w", err)
	}
	tmpPath := tmp.Name()
	cleanup := func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}
	if err := tmp.Chmod(mode); err != nil {
		cleanup()
		return fmt.Errorf("chmod tmp: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		cleanup()
		return fmt.Errorf("write tmp: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		cleanup()
		return fmt.Errorf("fsync tmp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("close tmp: %w", err)
	}
	if err := replaceFile(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("rename tmp→target: %w", err)
	}
	if err := SyncDirectory(dir); err != nil {
		return err
	}
	return nil
}

// --- Node 辅助函数 ---------------------------------------------------------

// FindChild 在一个 MappingNode 里查找 key；未找到返回 nil。
func FindChild(m *yaml.Node, key string) (keyNode, valNode *yaml.Node) {
	if m == nil || m.Kind != yaml.MappingNode {
		return nil, nil
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		k := m.Content[i]
		if k.Kind == yaml.ScalarNode && k.Value == key {
			return k, m.Content[i+1]
		}
	}
	return nil, nil
}

// EnsureMapChild 返回 m[key] 的值节点；不存在则新建一个空 MappingNode 并插入。
func EnsureMapChild(m *yaml.Node, key string) (*yaml.Node, error) {
	if m.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("parent is not a mapping")
	}
	if _, v := FindChild(m, key); v != nil {
		if v.Kind != yaml.MappingNode {
			return nil, fmt.Errorf("key %q exists but is not a mapping", key)
		}
		return v, nil
	}
	kn := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}
	vn := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	m.Content = append(m.Content, kn, vn)
	return vn, nil
}

// EnsureSeqChild 返回 m[key] 的序列节点；不存在则新建空 SequenceNode 并插入。
func EnsureSeqChild(m *yaml.Node, key string) (*yaml.Node, error) {
	if m.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("parent is not a mapping")
	}
	if _, v := FindChild(m, key); v != nil {
		if v.Kind != yaml.SequenceNode {
			return nil, fmt.Errorf("key %q exists but is not a sequence", key)
		}
		return v, nil
	}
	kn := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}
	vn := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	m.Content = append(m.Content, kn, vn)
	return vn, nil
}

// SetScalarAtPath 把 root.<path...> 设成 string scalar；中间缺失的层级自动以
// MappingNode 新建。空路径返回错误。
func SetScalarAtPath(root *yaml.Node, path []string, value string) error {
	if len(path) == 0 {
		return fmt.Errorf("empty path")
	}
	cur := root
	for i, seg := range path[:len(path)-1] {
		next, err := EnsureMapChild(cur, seg)
		if err != nil {
			return fmt.Errorf("at path[%d]=%q: %w", i, seg, err)
		}
		cur = next
	}
	last := path[len(path)-1]
	if _, v := FindChild(cur, last); v != nil {
		// 保留原节点的 Style / Comment，仅改 Value 与 Tag。
		v.Kind = yaml.ScalarNode
		v.Tag = "!!str"
		v.Value = value
		v.Content = nil
		return nil
	}
	kn := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: last}
	vn := &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
	cur.Content = append(cur.Content, kn, vn)
	return nil
}

// DeleteKey 从 MappingNode m 里移除 key；不存在返回 false。
func DeleteKey(m *yaml.Node, key string) bool {
	if m == nil || m.Kind != yaml.MappingNode {
		return false
	}
	for i := 0; i+1 < len(m.Content); i += 2 {
		if m.Content[i].Kind == yaml.ScalarNode && m.Content[i].Value == key {
			m.Content = append(m.Content[:i], m.Content[i+2:]...)
			return true
		}
	}
	return false
}

// EncodeInto 把任意 Go 值编码为 yaml Node 并**原地替换** dst 的 Kind/Content/Tag/Value。
// 保留 dst 的 HeadComment/LineComment/FootComment。调用者负责保证 dst 已在树里。
func EncodeInto(dst *yaml.Node, v any) error {
	if dst == nil {
		return fmt.Errorf("nil dst node")
	}
	var buf yaml.Node
	if err := buf.Encode(v); err != nil {
		return err
	}
	dst.Kind = buf.Kind
	dst.Tag = buf.Tag
	dst.Value = buf.Value
	dst.Style = buf.Style
	dst.Content = buf.Content
	// 不覆写 dst 的 HeadComment/LineComment/FootComment —— 保留注释。
	return nil
}

// debugDump 只在单元测试里用的排错工具。
func debugDump(w io.Writer, n *yaml.Node) {
	enc := yaml.NewEncoder(w)
	enc.SetIndent(2)
	_ = enc.Encode(n)
	_ = enc.Close()
}

var _ = debugDump // 避免 unused 告警（仅测试期引用）

// WriteAtomic 复用配置的唯一文件替换边界，供OAuth短操作记录等小文件使用。
func WriteAtomic(path string, data []byte, mode os.FileMode) error {
	return atomicWrite(filepath.Dir(path), path, data, mode)
}

// WithFileLock 只串行化短文件事务；所有配置写者共用同一锁，禁止在fn内等待网络。
func WithFileLock(ctx context.Context, path string, fn func() error) (resultErr error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	lock := flock.New(path + ".lock")
	locked, err := lock.TryLockContext(ctx, 25*time.Millisecond)
	if err != nil {
		return fmt.Errorf("acquire config lock: %w", err)
	}
	if !locked {
		return fmt.Errorf("configuration is busy")
	}
	defer func() {
		if err := lock.Unlock(); err != nil && resultErr == nil {
			resultErr = fmt.Errorf("unlock configuration: %w", err)
		}
	}()
	return fn()
}
