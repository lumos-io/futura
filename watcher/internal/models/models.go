package models

type Request struct {
	StartTime  int64
	Latency    uint64 // in ns
	FromIP     string
	FromType   string
	FromUID    string
	FromPort   uint16
	ToIP       string
	ToType     string
	ToUID      string
	ToPort     uint16
	Protocol   string
	Tls        bool
	Completed  bool
	StatusCode uint32
	FailReason string
	Method     string
	Path       string
	Tid        uint32
	Seq        uint32
}

type BackendResponse struct {
	Msg    string `json:"msg"`
	Errors []struct {
		EventNum int         `json:"event_num"`
		Event    interface{} `json:"event"`
		Error    string      `json:"error"`
	} `json:"errors"`
}

type ReqBackendReponse struct {
	Msg    string `json:"msg"`
	Errors []struct {
		EventNum int         `json:"request_num"`
		Event    interface{} `json:"request"`
		Error    string      `json:"errors"`
	} `json:"errors"`
}
