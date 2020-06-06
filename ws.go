package huobi

import (
	"errors"
	"log"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Lisenter = func([]byte)

var DestroyError = errors.New("Destroy websocket")
var writeWait = 2 * time.Second
var  LoggerFile *os.File
type SafeWebSocket struct {
	ws           *websocket.Conn
	lisenter     Lisenter
	sendMsgQueue chan []byte
	run          chan bool
	wg           *sync.WaitGroup
	endpoint     string
	Logger       *log.Logger
}

func NewSafeWebSocket(endpoint string) (*SafeWebSocket, error) {
	ws, _, er := websocket.DefaultDialer.Dial(endpoint, nil)
	if er != nil {
		return nil, er
        }
	var wg sync.WaitGroup
	wg.Add(2)
	LoggerFile, _ := os.OpenFile("/var/log/huobi.log", os.O_RDWR|os.O_APPEND|os.O_CREATE, os.ModeAppend)
	Logger := log.New(LoggerFile, "huobi: ", log.Ldate|log.Ltime)
	s := &SafeWebSocket{ws: ws, sendMsgQueue: make(chan []byte, 1000),
		wg: &wg, run: make(chan bool, 2), endpoint: endpoint, Logger: Logger}
	return s, nil
}

func (sws *SafeWebSocket) Reconnect() error {
	sws.run <- false
	sws.wg.Wait()
        LoggerFile.Close()        
	LoggerFile, _ = os.OpenFile("/var/log/huobi.log", os.O_RDWR|os.O_APPEND|os.O_CREATE, os.ModeAppend)
        
	Logger := log.New(LoggerFile, "huobi: ", log.Ldate|log.Ltime)
        sws.Logger = Logger
	sws.Logger.Println("now reconnect")
	ws, _, er := websocket.DefaultDialer.Dial(sws.endpoint, nil)
	if er != nil {
		return er
	}
	sws.ws = ws
	sws.wg.Add(2)
	sws.run = make(chan bool, 2)
	sws.sendMsgQueue = make(chan []byte, 1000)
	sws.Startup()
	return nil

}
func (s *SafeWebSocket) WriteDump() {
        defer s.wg.Done()
	for {
              select{
		case senddata := <-s.sendMsgQueue:
                       {
		             s.Logger.Println("handle write message",string(senddata))
		             if wer := s.ws.WriteMessage(websocket.TextMessage, senddata); wer != nil {
			       s.Logger.Println(wer)
			       s.Logger.Println("*******************************************")
			       s.run <- false
			       return
                            }
                            
	 	      }
              case flag:=<- s.run:
                  if !flag{
                         return
                    }
             }
	}
}
func (s *SafeWebSocket) ReadDump() {
        defer s.wg.Done()
	for {
             select{
                 case flag:= <-s.run:
                     { 
                              if !flag {

	                             return
                                 }

                      }
               default:
               {
		  if mestype, data, rerr := s.ws.ReadMessage(); rerr != nil {
			s.Logger.Println(rerr)
			s.Logger.Println("############################################")
			s.run <- false
                        return 

		  } else {
                        if mestype == websocket.BinaryMessage{   
			    s.lisenter(data)
                        }
		  }
               }
              }

	}
	
}

func (sws *SafeWebSocket) Lisenter(lisenter Lisenter) {
	sws.lisenter = lisenter
}
func (sws *SafeWebSocket) Wait() {
	sws.wg.Wait()
	sws.Logger.Println("read and write exit.....................")
}
func (sws *SafeWebSocket) Startup() {
	go sws.ReadDump()
	go sws.WriteDump()
}
func (sws *SafeWebSocket) Destroy() error {
	var err error
	sws.run <- false
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
