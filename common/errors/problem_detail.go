package errors

import "fmt"

// Problem Detail

type ProblemDetail struct {
	Type     string `json:"type"`
	Title    string `json:"title"`
	Status   int    `json:"status"`
	Detail   string `json:"detail"`
	Instance string `json:"instance"`
	Code     string `json:"code"`
}

func (e *ProblemDetail) Error() string {
	return fmt.Sprintf("ProblemDetail: %s, Detail: %s", e.Title, e.Detail)
}

func NewProblemDetail(
	type_ string,
	title string,
	status int,
	detail string,
	instance string,
	code string,
) *ProblemDetail {
	return &ProblemDetail{
		Type:     type_,
		Title:    title,
		Status:   status,
		Detail:   detail,
		Instance: instance,
		Code:     code,
	}
}
