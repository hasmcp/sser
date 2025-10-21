package err

import (
	"encoding/json"

	"github.com/mustafaturan/sser/internal/_data/entity"
	"github.com/mustafaturan/sser/internal/_data/view"
)

func FromErrorEntityToErrorView(e entity.Err) view.Err {
	return view.Err{
		Code:    int(e.Code),
		Message: e.Message,
		Details: e.Details,
	}
}

func FromErrorEntityToHttpResponse(e entity.Err) []byte {
	err := FromErrorEntityToErrorView(e)
	data := map[string]view.Err{
		"error": err,
	}
	payload, _ := json.Marshal(data)
	return payload
}

func FromErrorToHttpResponse(err error) ([]byte, int) {
	e, ok := err.(entity.Err)
	if !ok {
		e = entity.Err{
			Code:    entity.ErrorCodeInternalServerError,
			Message: "internal server error",
			Details: map[string]interface{}{
				"error": err.Error(),
			},
		}
	}
	return FromErrorEntityToHttpResponse(e), int(e.Code)
}
