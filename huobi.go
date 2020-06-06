package huobi

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/bitly/go-simplejson"
)

var Endpoint = "wss://api.huobi.pro/ws"

type Listener = func(data *simplejson.Json)

type HuoBi struct {
	sws              *SafeWebSocket
	subscribedList   map[string]Listener
	mutex            sync.Mutex
	subscribedResult map[string]bool
}

func GenerateConnect(endpoint string, timeout time.Duration) (*HuoBi, error) {
	sws, err := NewSafeWebSocket(endpoint)
	if err != nil {
		return nil, err
	}
	s := &HuoBi{sws: sws, subscribedList: make(map[string]Listener),
		subscribedResult: make(map[string]bool)}
	s.SetHandleMessage()
	s.sws.Startup()
	return s, nil
}

func DefaultConnect() (*HuoBi, error) {
	sws, err := NewSafeWebSocket(Endpoint)
	if err != nil {
		return nil, err
	}
	s := &HuoBi{sws: sws, subscribedList: make(map[string]Listener),
		subscribedResult: make(map[string]bool)}
	s.SetHandleMessage()
	s.sws.Startup()
	return s, nil
}
func (hb *HuoBi) ReConnect() error {
	err := hb.sws.Reconnect()
	if err != nil {
		return err
	}
	hb.SetHandleMessage()
	return nil
}
func (hb *HuoBi) RecoverResource() {
	hb.ReConnect()
	hb.mutex.Lock()
	for key := range hb.subscribedList {
		hb.SendMessage(subData{Id: key, Sub: key, Freq: "5000"})
	}
	hb.mutex.Unlock()

}

func (hb *HuoBi) HandlePing(ping int64) {
	data := pongData{Pong: ping}
	hb.SendMessage(data)
}

func (hb *HuoBi) SendMessage(data interface{}) error {
	bdata, er := json.Marshal(data)
	if er != nil {
		return er
	}
	hb.sws.SendMessage(bdata)
	return nil
}

func (hb *HuoBi) SetHandleMessage() {
	hb.sws.Lisenter(func(buf []byte) {
		data, er := UnGzip(buf)
		if er != nil {
			hb.sws.Logger.Println(er)
			return
		}
		jsondata, er := simplejson.NewJson(data)
		if er != nil {
			hb.sws.Logger.Println(er)
			return
		}
                if ping := jsondata.Get("ping").MustInt64(); ping > 0 {
                       hb.HandlePing(ping)
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

func (hb *HuoBi) Subcribe(topic string, lisenter Listener) {
	if _, ok := hb.subscribedList[topic]; !ok {
		hb.mutex.Lock()
		hb.subscribedList[topic] = lisenter
		hb.mutex.Unlock()
		hb.SendMessage(subData{Id: topic, Sub: topic, Freq: "5000"})
	}
	return
}
func (hb *HuoBi) KeepAlived() {
	for {
		hb.sws.Wait()
		hb.RecoverResource()
	}

}
func (hb *HuoBi) Close() {
	hb.sws.Destroy()
}
