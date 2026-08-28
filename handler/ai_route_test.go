package handler

import "testing"

func TestIsArkSeedanceVideoRequiresArkBaseURL(t *testing.T) {
	tests := []struct {
		name      string
		baseURL   string
		modelName string
		want      bool
	}{
		{name: "seedance nz stays on openai video endpoint", baseURL: "https://api.seedance.nz", modelName: "seedance-2.0-mini-t2v", want: false},
		{name: "lec stays on openai video endpoint", baseURL: "https://api.paipu.net", modelName: "lec-ac-seedance-900-720p", want: false},
		{name: "ark agent plan uses contents tasks", baseURL: "https://ark.cn-beijing.volces.com/api/plan/v3", modelName: "doubao-seedance-2.0", want: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := isArkSeedanceVideo(test.baseURL, test.modelName); got != test.want {
				t.Fatalf("isArkSeedanceVideo(%q, %q) = %v, want %v", test.baseURL, test.modelName, got, test.want)
			}
		})
	}
}
