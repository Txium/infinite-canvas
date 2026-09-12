package service

import "testing"

func TestVideoMultipartErrorIsActionable(t *testing.T) {
	got := userFriendlyTaskError("multipart image uploads are not supported; use public HTTPS image URLs", "生成失败")
	if got != "视频渠道不接受当前参考图传输格式，请联系管理员修复；无需充值或反复重试" {
		t.Fatalf("unexpected safe error: %s", got)
	}
}

func TestMissing302KeyIsActionableWithoutLeakingUpstreamText(t *testing.T) {
	got := userFriendlyTaskError("Missing 302 Apikey: private-debug-value", "生成失败")
	if got != "当前模型渠道密钥认证失败，请联系管理员修复；无需充值或反复重试" {
		t.Fatalf("unexpected safe error: %s", got)
	}
}
