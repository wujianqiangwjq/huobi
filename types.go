package huobi

import "github.com/bitly/go-simplejson"

type pingData struct {
	Ping int64 `json:"ping"`
}

type pongData struct {
	Pong int64 `json:"pong"`
}
type reqData struct {
	Req string `json:"req"`
	Id  string `json:"id"`
}
type subData struct {
	Sub  string `json:"sub"`
	Id   string `json:"id"`
	Freq string `json: "freq-ms"`
}
type unSubData struct {
	UnSub string `json:"unsub"`
	Id    string `json:"id"`
}
type JSON = simplejson.Json
