package huobi

import (
	"errors"
	"time"

	"github.com/gorilla/websocket"
)

type Lisenter = func([]byte)

var DestroyError = errors.New("Destroy websocket")

type SafeWebSocket struct {
	ws           *websocket.Conn
	lisenter     Lisenter
	sendMsgQueue chan []byte
	lastError    error
	sendFlag     bool
	recFlag      bool
}

func NewSafeWebSocket(endpoint string) (*SafeWebSocket, error) {
	ws, _, er := websocket.DefaultDialer.Dial(endpoint, nil)
	if er != nil {
		return nil, er
	}
	s := &SafeWebSocket{ws: ws, sendMsgQueue: make(chan []byte, 1000)}
	go func() {
		s.sendFlag = true
		for s.lastError == nil {
			senddata := <-s.sendMsgQueue
			if wer := s.ws.WriteMessage(websocket.TextMessage, senddata); wer != nil {
				s.lastError = wer
				break
			}

		}
		s.sendFlag = false
	}()
	go func() {
		s.recFlag = true
		for s.lastError == nil {
			if _, data, rerr := s.ws.ReadMessage(); rerr != nil {
				s.lastError = rerr
				break

			} else {
				s.lisenter(data)
			}

		}
		s.recFlag = false
	}()
	return s, nil
}

func (sws *SafeWebSocket) Lisenter(lisenter Lisenter) {
	sws.lisenter = lisenter
}
func (sws *SafeWebSocket) Destroy() error {
	var err error
	err = nil
	sws.lastError = DestroyError
	if !sws.sendFlag && !sws.recFlag {
		time.Sleep(100 * time.Millisecond)

	}
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
