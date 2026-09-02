package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// normalizeWaveSpeedH3VideoBody converts the canvas' common video payload into
// WaveSpeed's MiniMax H3 schema. H3 has separate text-to-video,
// image-to-video, and reference-to-video endpoints; sending the generic
// OpenAI fields through unchanged is rejected by WaveSpeed before a task is
// created.
func normalizeWaveSpeedH3VideoBody(body []byte, contentType string, upstreamPath string) ([]byte, string, error) {
	if !strings.Contains(strings.ToLower(contentType), "application/json") || !strings.Contains(upstreamPath, "/minimax-h3/") {
		return body, contentType, nil
	}
	var source map[string]any
	if err := json.Unmarshal(body, &source); err != nil {
		return nil, contentType, err
	}
	applyH3ContentReferences(source)
	result := map[string]any{
		"prompt":     firstString(source, "prompt", "text"),
		"resolution": normalizeWaveSpeedH3Resolution(firstString(source, "resolution", "quality", "vquality")),
		"duration":   normalizeWaveSpeedH3Duration(source),
	}
	if ratio := normalizeWaveSpeedH3Ratio(firstString(source, "aspect_ratio", "ratio", "size")); ratio != "" {
		result["aspect_ratio"] = ratio
	}
	if seed, ok := source["seed"]; ok {
		result["seed"] = seed
	}
	lowerPath := strings.ToLower(upstreamPath)
	switch {
	case strings.Contains(lowerPath, "/image-to-video"):
		image := firstString(source, "image", "image_url", "start_image_url")
		if image == "" {
			if values, ok := source["image_urls"].([]any); ok && len(values) > 0 {
				image = fmt.Sprint(values[0])
			}
			if values, ok := source["image_urls"].([]string); ok && len(values) > 0 {
				image = values[0]
			}
		}
		if image != "" {
			result["image"] = image
		}
		if last := firstString(source, "last_image", "end_image_url"); last != "" {
			result["last_image"] = last
		}
	case strings.Contains(lowerPath, "/reference-to-video"):
		copyStringArray(result, "reference_images", source, "reference_images", "image_urls")
		copyStringArray(result, "reference_videos", source, "reference_videos", "video_urls")
		copyStringArray(result, "reference_audios", source, "reference_audios", "audio_urls")
	}
	encoded, err := json.Marshal(result)
	return encoded, "application/json", err
}

func applyH3ContentReferences(source map[string]any) {
	items, ok := source["content"].([]any)
	if !ok {
		return
	}
	images, videos, audios := []string{}, []string{}, []string{}
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		typ, _ := entry["type"].(string)
		var value string
		switch typ {
		case "image_url":
			if nested, ok := entry["image_url"].(map[string]any); ok {
				value = strings.TrimSpace(fmt.Sprint(nested["url"]))
			}
			if value != "" {
				images = append(images, value)
			}
		case "video_url":
			if nested, ok := entry["video_url"].(map[string]any); ok {
				value = strings.TrimSpace(fmt.Sprint(nested["url"]))
			}
			if value != "" {
				videos = append(videos, value)
			}
		case "audio_url":
			if nested, ok := entry["audio_url"].(map[string]any); ok {
				value = strings.TrimSpace(fmt.Sprint(nested["url"]))
			}
			if value != "" {
				audios = append(audios, value)
			}
		}
	}
	if len(images) > 0 {
		source["image_urls"] = images
		source["reference_images"] = images
	}
	if len(videos) > 0 {
		source["video_urls"] = videos
		source["reference_videos"] = videos
	}
	if len(audios) > 0 {
		source["audio_urls"] = audios
		source["reference_audios"] = audios
	}
}

func firstString(source map[string]any, names ...string) string {
	for _, name := range names {
		if value, ok := source[name]; ok {
			if text := strings.TrimSpace(fmt.Sprint(value)); text != "" && text != "<nil>" {
				return text
			}
		}
	}
	return ""
}

func copyStringArray(target map[string]any, targetName string, source map[string]any, names ...string) {
	for _, name := range names {
		if values, ok := source[name].([]any); ok && len(values) > 0 {
			cleaned := make([]string, 0, len(values))
			for _, value := range values {
				if text := strings.TrimSpace(fmt.Sprint(value)); text != "" && text != "<nil>" {
					cleaned = append(cleaned, text)
				}
			}
			if len(cleaned) > 0 {
				target[targetName] = cleaned
			}
			return
		}
		if values, ok := source[name].([]string); ok && len(values) > 0 {
			target[targetName] = values
			return
		}
	}
}

func normalizeWaveSpeedH3Resolution(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "2k", "2k/4k", "4k":
		// The currently enabled WaveSpeed open-weights H3 routes document
		// only 480p and 768p.  Do not forward the canvas' 2K/4K presets,
		// otherwise the upstream rejects the request before creating a task.
		return "768p"
	case "768p", "768", "1080p", "1080":
		return "768p"
	default:
		return "480p"
	}
}

func normalizeWaveSpeedH3Duration(source map[string]any) int {
	value := firstString(source, "duration", "seconds", "video_seconds")
	var seconds int
	if _, err := fmt.Sscan(value, &seconds); err != nil || seconds == 0 {
		seconds = 5
	}
	if seconds < 3 {
		seconds = 3
	}
	if seconds > 15 {
		seconds = 15
	}
	return seconds
}

func normalizeWaveSpeedH3Ratio(value string) string {
	value = strings.TrimSpace(strings.ToLower(value))
	if strings.Contains(value, "x") {
		parts := strings.SplitN(value, "x", 2)
		if len(parts) == 2 {
			value = parts[0] + ":" + parts[1]
		}
	}
	switch value {
	case "16:9", "9:16", "1:1", "4:3", "3:4", "21:9", "9:21":
		return value
	default:
		return "16:9"
	}
}

func normalizeWaveSpeedInfiniteTalkBody(body []byte, contentType string) ([]byte, string, error) {
	if !strings.Contains(strings.ToLower(contentType), "application/json") {
		return nil, contentType, errors.New("InfiniteTalk 仅支持 JSON 请求")
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, contentType, err
	}
	delete(payload, "model")
	delete(payload, "seconds")
	if strings.TrimSpace(waveSpeedString(payload["image"])) == "" || strings.TrimSpace(waveSpeedString(payload["audio"])) == "" {
		return nil, contentType, errors.New("InfiniteTalk 需要人物图片和口播音频")
	}
	normalized, err := json.Marshal(payload)
	return normalized, "application/json", err
}

func normalizeWaveSpeedSeed3DBody(body []byte, contentType string) ([]byte, string, error) {
	if !strings.Contains(strings.ToLower(contentType), "application/json") {
		return nil, contentType, errors.New("Seed3D 2.0 仅支持 JSON 请求")
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, contentType, err
	}
	delete(payload, "model")
	if strings.TrimSpace(waveSpeedString(payload["image"])) == "" {
		return nil, contentType, errors.New("Seed3D 2.0 需要物体参考图")
	}
	normalized, err := json.Marshal(payload)
	return normalized, "application/json", err
}

func waveSpeedString(value any) string {
	text, _ := value.(string)
	return text
}
