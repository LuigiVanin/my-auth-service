package dto

import entity "auth_service/infra/entities"

type GetUsersAppQuery struct {
	Skip  int    `query:"skip"`
	Limit int    `query:"limit"`
	Name  string `query:"name"`
	Email string `query:"email"`
}

type GetUsersAppResponse struct {
	Total  int64         `json:"total"`
	Amount int           `json:"amount"`
	Skip   int           `json:"skip"`
	Data   []entity.User `json:"data"`
}
