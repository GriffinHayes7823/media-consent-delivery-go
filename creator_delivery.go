package mediaconsent

import (
	"context"
	"errors"
)

const DeliveryCategory = "third_party"

type AssetStage string

const (
	StageIngested   AssetStage = "ingested"
	StageProcessing AssetStage = "processing"
	StageReady      AssetStage = "ready"
)

type DeliveryRequest struct {
	UserID    string     `json:"user_id"`
	AssetID   string     `json:"asset_id"`
	JobID     string     `json:"job_id"`
	Stage     AssetStage `json:"stage"`
	CreatorID string     `json:"creator_id"`
}

type DeliveryDecision struct {
	AssetID string `json:"asset_id"`
	JobID   string `json:"job_id"`
	Next    string `json:"next"`
	Reason  string `json:"reason"`
}

func DecideDelivery(ctx context.Context, consent ConsentGateway, input DeliveryRequest) (DeliveryDecision, error) {
	decision := DeliveryDecision{AssetID: input.AssetID, JobID: input.JobID}
	if input.UserID == "" || input.AssetID == "" || input.JobID == "" || input.CreatorID == "" {
		return decision, errors.New("user_id, asset_id, job_id, and creator_id are required")
	}
	if input.Stage != StageReady {
		decision.Next = "wait_for_processing"
		decision.Reason = "asset_not_ready"
		return decision, nil
	}
	granted, err := consent.Check(ctx, input.UserID, DeliveryCategory)
	if err != nil {
		return decision, err
	}
	if !granted {
		decision.Next = "request_consent"
		decision.Reason = "consent_required"
		return decision, nil
	}
	decision.Next = "deliver_to_creator"
	decision.Reason = "ready_and_granted"
	return decision, nil
}
