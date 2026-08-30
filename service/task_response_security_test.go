package service

import (
	"testing"

	"github.com/tigerowo/infinite-canvas/model"
	"github.com/tigerowo/infinite-canvas/repository"
)

func TestUserVideoTaskResponseDoesNotExposeUpstreamOrInternalFields(t *testing.T) {
	response := VideoTaskResponse(model.VideoTask{
		ID: "local-task", UpstreamTaskID: "provider-task", UpstreamVideoID: "provider-video",
		UpstreamModel: "secret-upstream-model", ChannelID: "provider_lec", ChannelName: "LEC",
		RequestBody: `{"api_key":"secret"}`, Status: "failed", ErrorDetail: "HTTP 502 upstream timeout",
	})
	for _, key := range []string{"video_id", "channelId", "userChannelId", "channelName", "request_body", "error_detail"} {
		if _, ok := response[key]; ok {
			t.Fatalf("user response leaked %s", key)
		}
	}
	if response["task_id"] != "local-task" {
		t.Fatalf("task_id = %v, want local task id", response["task_id"])
	}
	errorValue := response["error"].(map[string]any)["message"]
	if errorValue != "当前模型生成失败，请稍后重试" {
		t.Fatalf("unexpected user error: %v", errorValue)
	}
}

func TestUserImageAndAudioResponsesHideInternalErrors(t *testing.T) {
	image := CanvasImageTaskResponse(model.CanvasImageTask{ID: "image", Status: "failed", ErrorDetail: "provider API key invalid"})
	audio := CanvasAudioTaskResponse(model.CanvasAudioTask{ID: "audio", Status: "failed", ErrorDetail: "HTTP status code 503"})
	for name, response := range map[string]map[string]any{"image": image, "audio": audio} {
		if _, ok := response["error_detail"]; ok {
			t.Fatalf("%s response leaked error_detail", name)
		}
	}
}

func TestUnknownVideoStatusRemainsPollable(t *testing.T) {
	if got := NormalizeVideoTaskStatus("provider_new_state"); got != "reconciling" {
		t.Fatalf("NormalizeVideoTaskStatus unknown = %q, want reconciling", got)
	}
}

func TestReconcilingVideoTaskKeepsDiagnosticWithoutRefunding(t *testing.T) {
	task := model.VideoTask{ID: "local", Status: "queued", BillingStatus: "frozen"}
	// Exercise the state mutation without billing by leaving Credits at zero.
	if err := UpdateVideoTaskFromPoll(task, VideoTaskPollUpdate{Status: "reconciling", ErrorDetail: "transport response lost"}); err != nil {
		t.Fatal(err)
	}
	saved, found, err := repository.GetVideoTask("local")
	if err != nil || !found {
		t.Fatalf("saved task missing: found=%v err=%v", found, err)
	}
	if saved.Status != "reconciling" || saved.ErrorDetail != "transport response lost" || saved.BillingStatus != "frozen" {
		t.Fatalf("unexpected reconciliation state: %+v", saved)
	}
}
