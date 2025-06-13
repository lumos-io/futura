package models

type BackendResponse struct {
	Msg    string `json:"msg"`
	Errors []struct {
		EventNum int    `json:"event_num"`
		Event    any    `json:"event"`
		Error    string `json:"error"`
	} `json:"errors"`
}

type ReqBackendReponse struct {
	Msg    string `json:"msg"`
	Errors []struct {
		EventNum int    `json:"request_num"`
		Event    any    `json:"request"`
		Error    string `json:"errors"`
	} `json:"errors"`
}
