package huobi

import (
	"encoding/json"
	"log"
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
	lastPing         int64
	sendPing         int64
	timeout          time.Duration
}

func GenerateConnect(endpoint string, timeout time.Duration) (*HuoBi, error) {
	sws, err := NewSafeWebSocket(endpoint)
	if err != nil {
		return nil, err
	}
	s := &HuoBi{sws: sws, subscribedList: make(map[string]Listener),
		subscribedResult: make(map[string]bool), timeout: timeout}
	s.SetHandleMessage()
	s.sws.Startup()
	return s, nil
}

func DefaultConnect(timeout time.Duration) (*HuoBi, error) {
	sws, err := NewSafeWebSocket(Endpoint)
	if err != nil {
		return nil, err
	}
	s := &HuoBi{sws: sws, subscribedList: make(map[string]Listener),
		subscribedResult: make(map[string]bool), timeout: timeout}
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
	hb.sws.Startup()

	return nil
}
func (hb *HuoBi) RecoverResource() {
	hb.ReConnect()
	hb.mutex.Lock()
	for key := range hb.subscribedList {
		hb.SendMessage(subData{Id: key, Sub: key})
	}
	hb.mutex.Unlock()

}
func (hb *HuoBi) SendMessage(data interface{}) error {
	bdata, er := json.Marshal(data)
	if er != nil {
		return er
	}
	hb.sws.SendMessage(bdata)
	return nil
}

func (hb *HuoBi) SendPingMessage() {
	hb.mutex.Lock()
	hb.sendPing = GetUinxMillisecond()
	hb.mutex.Unlock()
	hb.SendMessage(pingData{Ping: hb.sendPing})
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
			log.Println("handle ping message")
			if er := hb.HandlePing(pingData{Ping: ping}); er != nil {
				log.Println(er)
				hb.HandlePing(pingData{Ping: ping})

			}

		}
		if pong := jsondata.Get("pong").MustInt64(); pong > 0 {
			log.Println("handle pong message")
			hb.mutex.Lock()
			hb.lastPing = pong
			hb.mutex.Unlock()
		}

		if ch := jsondata.Get("ch").MustString(); ch != "" {
			hb.mutex.Lock()
			listerHandler, ok := hb.subscribedList[ch]
			hb.mutex.Unlock()
			if ok {
				log.Println("handle read message")
				go listerHandler(jsondata)
			}
		}
		if subed := jsondata.Get("subbed").MustString(); subed != "" {
			hb.subscribedResult[subed] = true

		}

	})

}
func (hb *HuoBi) HandlePing(ping pingData) error {
	data := pongData{Pong: ping.Ping}
	return hb.SendMessage(data)
}
func (hb *HuoBi) Subcribe(topic string, lisenter Listener) {
	if _, ok := hb.subscribedList[topic]; !ok {
		hb.mutex.Lock()
		hb.subscribedList[topic] = lisenter
		hb.mutex.Unlock()
		hb.SendMessage(subData{Id: topic, Sub: topic})
	}
	return
}
func (hb *HuoBi) KeepAlived() {
	ticker := time.NewTicker(hb.timeout)
	hb.SendPingMessage()
	for {
		select {
		case <-ticker.C:
			if hb.sendPing != hb.lastPing {
				hb.RecoverResource()
				ticker.Stop()
				ticker = time.NewTicker(hb.timeout)

			}
			hb.SendPingMessage()

		}
	}

}
func (hb *HuoBi) Close() {
	hb.sws.Destroy()
}
