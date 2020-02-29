package huobi

import (
	"errors"
	"log"
	"sync"

	"github.com/gorilla/websocket"
)

type Lisenter = func([]byte)

var DestroyError = errors.New("Destroy websocket")

type SafeWebSocket struct {
	ws           *websocket.Conn
	lisenter     Lisenter
	sendMsgQueue chan []byte
	run          bool
	mutex        sync.Mutex
	wg           *sync.WaitGroup
	endpoint     string
}

func NewSafeWebSocket(endpoint string) (*SafeWebSocket, error) {
	ws, _, er := websocket.DefaultDialer.Dial(endpoint, nil)
	if er != nil {
		return nil, er
	}
	var wg sync.WaitGroup
	wg.Add(2)
	s := &SafeWebSocket{ws: ws, sendMsgQueue: make(chan []byte, 1000),
		wg: &wg, run: true, endpoint: endpoint}
	return s, nil
}
func (sws *SafeWebSocket) Reconnect() error {
	sws.mutex.Lock()
	sws.run = false
	sws.mutex.Unlock()
	sws.wg.Wait()
	ws, _, er := websocket.DefaultDialer.Dial(sws.endpoint, nil)
	if er != nil {
		return er
	}
	sws.ws = ws
	sws.wg.Add(2)
	sws.run = true
	sws.sendMsgQueue = make(chan []byte, 1000)
	sws.Startup()
	return nil

}
func (s *SafeWebSocket) WriteDump() {
	for s.run {
		senddata := <-s.sendMsgQueue
		log.Println("handle write message")
		if wer := s.ws.WriteMessage(websocket.TextMessage, senddata); wer != nil {
			log.Println(wer)
			s.mutex.Lock()
			s.run = false
			s.mutex.Unlock()
		}
	}
	defer s.wg.Done()
}
func (s *SafeWebSocket) ReadDump() {
	for s.run {
		if _, data, rerr := s.ws.ReadMessage(); rerr != nil {
			log.Println(rerr)
			s.mutex.Lock()
			s.run = false
			s.mutex.Unlock()

		} else {
			go s.lisenter(data)
		}

	}
	defer s.wg.Done()

}

func (sws *SafeWebSocket) Lisenter(lisenter Lisenter) {
	sws.lisenter = lisenter
}
func (sws *SafeWebSocket) Wait() {
	sws.wg.Wait()
}
func (sws *SafeWebSocket) Startup() {
	go sws.ReadDump()
	go sws.WriteDump()
}
func (sws *SafeWebSocket) Destroy() error {
	var err error
	sws.mutex.Lock()
	sws.run = false
	sws.mutex.Unlock()
	sws.wg.Wait()
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
