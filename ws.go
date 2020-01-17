package huobi

import (
	"errors"
	"sync"

	"github.com/gorilla/websocket"
)

type Lisenter = func([]byte)

var DestroyError = errors.New("Destroy websocket")

type SafeWebSocket struct {
	ws           *websocket.Conn
	lisenter     Lisenter
	sendMsgQueue chan []byte
	lastError    error
	wg           *sync.WaitGroup
}

func NewSafeWebSocket(endpoint string) (*SafeWebSocket, error) {
	ws, _, er := websocket.DefaultDialer.Dial(endpoint, nil)
	if er != nil {
		return nil, er
	}
	var wg sync.WaitGroup
	wg.Add(2)
	s := &SafeWebSocket{ws: ws, sendMsgQueue: make(chan []byte, 1000), wg: &wg}
	go func() {

		for s.lastError == nil {
			senddata := <-s.sendMsgQueue
			go func(sws *SafeWebSocket, data []byte) {
				if wer := sws.ws.WriteMessage(websocket.TextMessage, data); wer != nil {
					sws.lastError = wer
				}
			}(s, senddata)

		}
		wg.Done()
	}()
	go func() {

		for s.lastError == nil {
			if _, data, rerr := s.ws.ReadMessage(); rerr != nil {
				s.lastError = rerr
				break

			} else {
				s.lisenter(data)
			}

		}
		wg.Done()
	}()
	return s, nil
}

func (sws *SafeWebSocket) Lisenter(lisenter Lisenter) {
	sws.lisenter = lisenter
}
func (sws *SafeWebSocket) Wait() {
	sws.wg.Wait()
}
func (sws *SafeWebSocket) Destroy() error {
	var err error
	err = nil
	sws.lastError = DestroyError
	if sws.ws != nil {
		err = sws.ws.Close()
		sws.ws = nil
	}
	sws.sendMsgQueue = nil
	return err

}

func (sws *SafeWebSocket) SendMessage(data []byte) {
	sws.sendMsgQueue <- data
}
