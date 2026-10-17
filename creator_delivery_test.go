package mediaconsent

import (
	"context"
	"testing"
)

type stubConsent struct{ granted bool }

func (s stubConsent) Check(context.Context, string, string) (bool, error)  { return s.granted, nil }
func (s stubConsent) Grant(context.Context, string, string, string) error  { return nil }
func (s stubConsent) Revoke(context.Context, string, string, string) error { return nil }

func TestDecideDelivery(t *testing.T) {
	tests := []struct {
		name    string
		stage   AssetStage
		granted bool
		want    string
	}{
		{name: "processing job waits", stage: StageProcessing, granted: true, want: "wait_for_processing"},
		{name: "ready asset waits for user", stage: StageReady, granted: false, want: "request_consent"},
		{name: "ready consented asset is delivered", stage: StageReady, granted: true, want: "deliver_to_creator"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DecideDelivery(context.Background(), stubConsent{granted: tt.granted}, DeliveryRequest{
				UserID: "listener-42", AssetID: "asset-8", JobID: "transcode-9", Stage: tt.stage, CreatorID: "creator-3",
			})
			if err != nil {
				t.Fatal(err)
			}
			if got.Next != tt.want {
				t.Fatalf("Next = %q, want %q", got.Next, tt.want)
			}
		})
	}
}
