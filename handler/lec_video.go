package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"mime"
	"mime/multipart"
	"net/url"
	"strings"
)

// The 900 endpoint accepts JSON, not multipart, even when every part is a URL.
// https://api.paipu.net/docs/videos/lec-seed-2-0-900
func normalizeLEC900VideoBody(body []byte, contentType string) ([]byte, string, error) {
	payload := map[string]any{}
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, "", errors.New("视频请求格式无效")
	}
	if mediaType == "multipart/form-data" {
		form, err := multipart.NewReader(bytes.NewReader(body), params["boundary"]).ReadForm(32 << 20)
		if err != nil {
			return nil, "", errors.New("视频请求格式无效")
		}
		defer form.RemoveAll()
		if len(form.File) > 0 {
			return nil, "", errors.New("参考素材尚未转换为公网 HTTPS 链接，请重新上传图片")
		}
		for key, values := range form.Value {
			if len(values) == 1 {
				payload[key] = parseAPIMartFormValue(values[0])
			} else {
				payload[key] = values
			}
		}
	} else if mediaType != "application/json" || json.Unmarshal(body, &payload) != nil {
		return nil, "", errors.New("视频请求格式无效")
	}
	for _, key := range []string{"video_reference[]", "audio_reference[]", "videos", "audios", "last_frame_url"} {
		if len(collectAPIMartReferenceStrings(payload[key], 0)) > 0 {
			return nil, "", errors.New("当前视频档位只支持图片参考，不支持尾帧、视频或音频参考")
		}
	}
	images := []string{}
	for _, key := range []string{"first_frame_url", "images", "input_reference[]", "input_reference", "image_urls", "image_url"} {
		images = append(images, collectAPIMartReferenceStrings(payload[key], 0)...)
	}
	if len(images) > 9 {
		return nil, "", errors.New("当前视频档位最多支持 9 张参考图片")
	}
	for _, image := range images {
		u, err := url.Parse(image)
		if err != nil || u.Scheme != "https" || u.Hostname() == "" || u.User != nil {
			return nil, "", errors.New("参考图片必须是可访问的公网 HTTPS 链接，请重新上传图片")
		}
	}
	ratio := firstNonEmpty(toStringSafe(payload["aspect_ratio"]), toStringSafe(payload["size"]), "16:9")
	switch ratio {
	case "1280x720", "1920x1080", "16:9":
		ratio = "16:9"
	case "720x1280", "1080x1920", "9:16":
		ratio = "9:16"
	default:
		return nil, "", errors.New("当前视频档位只支持横屏 16:9 或竖屏 9:16，请调整画幅")
	}
	if seconds := toStringSafe(payload["seconds"]); seconds != "" && seconds != "15" {
		return nil, "", errors.New("当前视频档位固定生成 15 秒，请调整时长")
	}
	prompt := strings.TrimSpace(toStringSafe(payload["prompt"]))
	if prompt == "" {
		return nil, "", errors.New("请填写视频提示词")
	}
	result, err := json.Marshal(map[string]any{"model": "lec-seed-2-0-900", "prompt": prompt, "aspect_ratio": ratio, "images": images})
	return result, "application/json", err
}
