package view

type Err struct {
	Code    int                    `json:"code,omitempty"`
	Message string                 `json:"message,omitempty"`
	Details map[string]interface{} `json:"details,omitempty"`
}

func (e *Err) Error() string {
	return e.Message
}
