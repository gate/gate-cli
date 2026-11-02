// Package configcmd owns the shared interactive config commands; persistence uses internal/config only.
package configcmd

import (
	"bufio"
	"context"
	"fmt"
	"golang.org/x/term"
	"io"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gate/gate-cli/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// update 统一短锁与AST原子合并，交互输入必须在持锁前完成。
func update(cmd *cobra.Command, mutate func(*yaml.Node) error) error {
	parent := cmd.Context()
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, 10*time.Second)
	defer cancel()
	path := config.DefaultConfigPath()
	return config.WithFileLock(ctx, path, func() error { return config.UpdateFile(path, mutate, 0600) })
}
func RunInit(cmd *cobra.Command) error {
	// Only the explicitly requested real-TTY mode takes the new path. Missing
	// flags in existing direct calls and all pipes retain the original reader.
	var hidden [3]string
	useHidden := false
	if flag := cmd.Flags().Lookup("hide-input"); flag != nil {
		enabled, _ := cmd.Flags().GetBool("hide-input")
		if file, ok := cmd.InOrStdin().(*os.File); enabled && ok && term.IsTerminal(int(file.Fd())) {
			var err error
			ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt, syscall.SIGTERM)
			defer stop() // Retain the subscription through RunInit; stop also cancels ctx.
			hidden, err = readHiddenInput(ctx, file, cmd.OutOrStdout())
			if err != nil {
				return err
			}
			useHidden = true
		}
	}
	next := 0
	reader := bufio.NewReader(cmd.InOrStdin())
	read := func(prompt string) (string, error) {
		if useHidden {
			value := hidden[next]
			next++
			return value, nil
		}
		if _, err := fmt.Fprint(cmd.OutOrStdout(), prompt); err != nil {
			return "", err
		}
		value, err := reader.ReadString('\n')
		if err != nil && (err != io.EOF || value == "") {
			return "", fmt.Errorf("read configuration input: %w", err)
		}
		return strings.TrimSpace(value), nil
	}
	profile, err := read("Profile name [default]: ")
	if err != nil {
		return err
	}
	if profile == "" {
		profile = "default"
	}
	key, err := read("API Key: ")
	if err != nil {
		return err
	}
	secret, err := read("API Secret: ")
	if err != nil {
		return err
	}
	if len(profile) > 128 {
		return fmt.Errorf("profile name is too long")
	}
	err = update(cmd, func(root *yaml.Node) error {
		// 首次创建和已有配置使用同一锁内重读，不存在stat缺失后覆盖并发Token的分支。
		if err := config.SetScalarAtPath(root, []string{"default_profile"}, profile); err != nil {
			return err
		}
		if _, value := config.FindChild(root, "default_settle"); value == nil {
			if err := config.SetScalarAtPath(root, []string{"default_settle"}, "usdt"); err != nil {
				return err
			}
		}
		for _, item := range []struct{ key, value string }{{"info_mcp_url", config.DefaultIntelInfoMCPURL}, {"news_mcp_url", config.DefaultIntelNewsMCPURL}} {
			intel, err := config.EnsureMapChild(root, "intel")
			if err != nil {
				return err
			}
			if _, value := config.FindChild(intel, item.key); value == nil {
				if err := config.SetScalarAtPath(intel, []string{item.key}, item.value); err != nil {
					return err
				}
			}
		}
		if err := config.SetScalarAtPath(root, []string{"profiles", profile, "api_key"}, key); err != nil {
			return err
		}
		return config.SetScalarAtPath(root, []string{"profiles", profile, "api_secret"}, secret)
	})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "Config written to %s\n", config.DefaultConfigPath())
	return err
}
func RunSet(cmd *cobra.Command, args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("set requires key and value")
	}
	if _, err := os.Stat(config.DefaultConfigPath()); err != nil {
		return fmt.Errorf("no config file found — run: gate-cli config init")
	}
	key := map[string]string{"api-key": "api_key", "api-secret": "api_secret", "base-url": "base_url"}[args[0]]
	if key == "" {
		return fmt.Errorf("unknown key %q — valid keys: api-key, api-secret, base-url", args[0])
	}
	profile, _ := cmd.Flags().GetString("profile")
	err := update(cmd, func(root *yaml.Node) error {
		// 在锁内读取default_profile，避免并发初始化/切换profile导致陈旧整文件覆盖。
		if !cmd.Flags().Changed("profile") {
			if _, value := config.FindChild(root, "default_profile"); value != nil && value.Kind == yaml.ScalarNode && value.Value != "" {
				profile = value.Value
			}
		}
		if profile == "" {
			profile = "default"
		}
		return config.SetScalarAtPath(root, []string{"profiles", profile, key}, args[1])
	})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "Set %s for profile %q\n", args[0], profile)
	return err
}

var secretKeys = map[string]bool{"api_key": true, "api_secret": true, "bearer_token": true, "news_bearer_token": true, "info_bearer_token": true, "access_token": true, "refresh_token": true}

// redact 使用YAML AST覆盖块/数组/flow形式，不靠行前缀遗漏Token；解析失败时不回显原文。
func redact(content string) (string, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(content), &doc); err != nil {
		return "", fmt.Errorf("config cannot be safely displayed: invalid YAML")
	}
	changed := false
	// Header names are extensible, so the whole custom-header value is secret.
	// Follow aliases before replacing nodes: otherwise an anchor elsewhere in
	// the same document could still disclose the credential after masking its use.
	masked := make(map[*yaml.Node]bool)
	var mask func(*yaml.Node)
	mask = func(node *yaml.Node) {
		if node == nil || masked[node] {
			return
		}
		masked[node] = true
		mask(node.Alias)
		for _, child := range node.Content {
			mask(child)
		}
		anchor := node.Anchor
		*node = yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: "****", Style: yaml.DoubleQuotedStyle, Anchor: anchor}
		changed = true
	}
	// Endpoints can carry credentials outside token fields. Mask malformed
	// settings and remove userinfo/query/fragment from valid display values;
	// the persisted configuration and the actual request destination are untouched.
	redactEndpoint := func(value *yaml.Node) {
		target := value
		aliases := make(map[*yaml.Node]bool)
		for target != nil && target.Kind == yaml.AliasNode {
			if aliases[target] {
				mask(value)
				return
			}
			aliases[target] = true
			target = target.Alias
		}
		if target == nil || target.Kind != yaml.ScalarNode {
			mask(value)
			return
		}
		if target.Value == "" {
			return
		}
		endpoint, err := url.Parse(target.Value)
		if err != nil || endpoint.Host == "" || endpoint.Scheme == "" {
			mask(value)
			return
		}
		if endpoint.User != nil || endpoint.RawQuery != "" || endpoint.ForceQuery || endpoint.Fragment != "" {
			endpoint.User, endpoint.RawQuery, endpoint.Fragment, endpoint.RawFragment = nil, "", "", ""
			endpoint.ForceQuery = false
			target.Value = endpoint.String()
			changed = true
		}
	}
	visited := make(map[*yaml.Node]bool)
	var walk func(*yaml.Node)
	walk = func(node *yaml.Node) {
		if node == nil || visited[node] {
			return
		}
		visited[node] = true
		if node.Kind == yaml.MappingNode {
			for i := 0; i+1 < len(node.Content); i += 2 {
				key, value := node.Content[i], node.Content[i+1]
				if secretKeys[strings.ToLower(key.Value)] || strings.EqualFold(key.Value, "extra_headers") {
					mask(value)
				} else if strings.HasSuffix(strings.ToLower(key.Value), "_url") || key.Value == "resource" {
					redactEndpoint(value)
				} else {
					walk(value)
				}
			}
		} else {
			walk(node.Alias)
			for _, child := range node.Content {
				walk(child)
			}
		}
	}
	walk(&doc)
	if !changed {
		return content, nil
	}
	out, err := yaml.Marshal(&doc)
	if err != nil {
		return "", fmt.Errorf("config cannot be safely displayed")
	}
	return string(out), nil
}
func MaskSecrets(content string) string {
	result, err := redact(content)
	if err != nil {
		return "Config cannot be safely displayed.\n"
	}
	return result
}
func RunList(cmd *cobra.Command) error {
	data, err := os.ReadFile(config.DefaultConfigPath())
	if err != nil {
		return fmt.Errorf("no config file found — run: gate-cli config init")
	}
	show, _ := cmd.Flags().GetBool("show-secrets")
	content := string(data)
	if !show {
		content, err = redact(content)
		if err != nil {
			return err
		}
	}
	_, err = fmt.Fprint(cmd.OutOrStdout(), content)
	return err
}
