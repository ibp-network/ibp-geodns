package nats

import "github.com/google/uuid"

func generateProposalID() string {
	return uuid.New().String()
}
