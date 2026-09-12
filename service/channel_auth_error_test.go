package service

import "testing"

func TestMissing302KeyIsActionableWithoutLeakingUpstreamText(t *testing.T) {
	got := userFriendlyTaskError("Missing 302 Apikey: private-debug-value", "生成失败")
	if got != "当前模型渠道密钥认证失败，请联系管理员修复；无需充值或反复重试" {
		t.Fatalf("unexpected safe error: %s", got)
	}
}
