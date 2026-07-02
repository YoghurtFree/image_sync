package worker

import (
	"encoding/json"
	"testing"
)

func TestImageSyncPayloadMarshal(t *testing.T) {
	payload := ImageSyncPayload{
		Src:   "harbor-prod",
		Dst:   "acr-cn",
		Image: "library/nginx",
		Tag:   "1.25",
	}

	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var got ImageSyncPayload
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if got.Src != payload.Src {
		t.Errorf("Src = %s, want %s", got.Src, payload.Src)
	}
	if got.Dst != payload.Dst {
		t.Errorf("Dst = %s, want %s", got.Dst, payload.Dst)
	}
	if got.Image != payload.Image {
		t.Errorf("Image = %s, want %s", got.Image, payload.Image)
	}
	if got.Tag != payload.Tag {
		t.Errorf("Tag = %s, want %s", got.Tag, payload.Tag)
	}
}

func TestQueueNameMapping(t *testing.T) {
	tests := []struct {
		priority string
		want     string
	}{
		{"high", "critical"},
		{"normal", "default"},
		{"low", "low"},
		{"", "default"},
	}

	for _, tt := range tests {
		got := QueueName(tt.priority)
		if got != tt.want {
			t.Errorf("QueueName(%q) = %q, want %q", tt.priority, got, tt.want)
		}
	}
}
