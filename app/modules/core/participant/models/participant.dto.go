package models

import (
	entity "auth_service/infra/entities"
)

// Only the profile: moving a participation to another organization or another user
// is a different row, which is why ParticipantUpdateDao does not carry either.
type UpdateParticipant struct {
	ProfileId string `json:"profile_id" validate:"required,uuid4"`
}

type GetParticipantsQuery struct {
	Skip  int `query:"skip"`
	Limit int `query:"limit"`
}

type GetParticipantsResponse struct {
	Total  int64                `json:"total"`
	Amount int                  `json:"amount"`
	Skip   int                  `json:"skip"`
	Limit  int                  `json:"limit"`
	Data   []entity.Participant `json:"data"`
}
