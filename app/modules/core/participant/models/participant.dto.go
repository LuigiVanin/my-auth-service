package models

import (
	entity "auth_service/infra/entities"
)

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
