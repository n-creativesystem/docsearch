package fsm

type ApplyResponse struct {
	Error error
	Data  interface{}
}

func NewApplyErr(err error) *ApplyResponse {
	return &ApplyResponse{Error: err}
}
