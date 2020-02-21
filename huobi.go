package huobi

import (
	"encoding/json"
	"log"
	"sync"

	"github.com/bitly/go-simplejson"
)

var Endpoint = "wss://api.huobi.pro/ws"

type Listener = func(data *simplejson.Json)

type HuoBi struct {
	sws              *SafeWebSocket
	subscribedList   map[string]Listener
	mutex            sync.Mutex
	subscribedResult map[string]bool
	flagpong         bool
}

func GenerateConnect(endpoint string) (*HuoBi, error) {
	sws, err := NewSafeWebSocket(endpoint)
	if err != nil {
		return nil, err
	}
	s := &HuoBi{sws: sws, subscribedList: make(map[string]Listener), subscribedResult: make(map[string]bool), flagpong: false}
	s.SetHandleMessage()
	return s, nil
}

func DefaultConnect() (*HuoBi, error) {
	sws, err := NewSafeWebSocket(Endpoint)
	if err != nil {
		return nil, err
	}
	s := &HuoBi{sws: sws, subscribedList: make(map[string]Listener), subscribedResult: make(map[string]bool), flagpong: false}
	s.SetHandleMessage()
	return s, nil
}

func (hb *HuoBi) SendMessage(data interface{}) error {
	bdata, er := json.Marshal(data)
	if er != nil {
		return er
	}
	hb.sws.SendMessage(bdata)
	return nil
}
func (hb *HuoBi) SendPongMessage(data interface{}) {
	if hb.flagpong {
		bdata, er := json.Marshal(data)
		if er != nil {
			return
		}
		hb.sws.DirectSendMessage(bdata)
		hb.flagpong = !hb.flagpong
	}
	return
}

func (hb *HuoBi) SetHandleMessage() {
	hb.sws.Lisenter(func(buf []byte) {
		data, er := UnGzip(buf)
		if er != nil {
			log.Println(er)
			return
		}
		jsondata, er := simplejson.NewJson(data)
		if er != nil {
			log.Println(er)
			return
		}
		if ping := jsondata.Get("ping").MustInt64(); ping > 0 {
			hb.HandlePing(pingData{Ping: ping})

		}
		if ch := jsondata.Get("ch").MustString(); ch != "" {
			hb.mutex.Lock()
			listerHandler, ok := hb.subscribedList[ch]
			hb.mutex.Unlock()
			if ok {
				go listerHandler(jsondata)
			}
		}
		if subed := jsondata.Get("subbed").MustString(); subed != "" {
			hb.subscribedResult[subed] = true

		}

	})

}
func (hb *HuoBi) HandlePing(ping pingData) {
	data := pongData{Pong: ping.Ping}
	hb.SendPongMessage(data)
}
func (hb *HuoBi) Subcribe(topic string, lisenter Listener) {
	if _, ok := hb.subscribedList[topic]; !ok {
		hb.subscribedList[topic] = lisenter
		hb.SendMessage(subData{Id: topic, Sub: topic})
	}
	return
}
func (hb *HuoBi) Loop() {
	hb.sws.Wait()
}
func (hb *HuoBi) Close() {
	hb.sws.Destroy()
}
