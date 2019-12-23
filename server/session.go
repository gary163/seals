package server

import (
	"errors"
	"log"
	"sync"
	"sync/atomic"

	"github.com/gary163/seals/protocol"
)

var sessionID int64

type SessionManager struct {
	sessions map[int64]*Session
	mu sync.RWMutex
	wg sync.WaitGroup
	destroyOnce sync.Once
}

type Session struct {
	id        int64
	sendChan  chan interface{}
	codec     protocol.Codec
	closeFlag int32
	sendMu    sync.Mutex
	recvMu    sync.Mutex
	sm        *SessionManager
}

func (sm *SessionManager) NewSession(codec protocol.Codec, sendChanSize int) *Session {
	session := newSession(codec ,sendChanSize ,sm)
	sm.Set(session)
	sm.wg.Add(1)
	return session
}

func (sm *SessionManager) Set(session *Session) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.sessions[session.id] = session
}

func (sm *SessionManager) Len() int64 {
	return int64(len(sm.sessions))
}

func (sm *SessionManager) Get(sid int64)*Session {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	if session,ok := sm.sessions[sid]; ok {
		return session
	}
	return nil
}

func (sm *SessionManager) Del(sid int64) {
	log.Printf("SM session[%d] try to get a log",sid)
	sm.mu.Lock()
	log.Printf("SM session[%d] had got a log",sid)
	defer sm.mu.Unlock()
	if _,ok := sm.sessions[sid]; ok {
		delete(sm.sessions,sid)
		sm.wg.Done()
	}

	log.Println("SessionManager Del a record")
}

func (sm *SessionManager) Destroy() {
	log.Println("Session destroying......")
	sm.destroyOnce.Do(func(){
		log.Println("Session Destroy: try to get a lock")
		log.Println("Session Destroy: got a lock")
		sm.mu.Lock()
		for _,session := range sm.sessions {
			err := session.Close()
			log.Printf("Session Destroy Close err:%v\n",err)
		}
		sm.mu.Unlock()
		log.Println("Session Destroy: release lock")
		log.Println("Session Destroy:waiting")
		sm.wg.Wait()
		log.Println("Session Destroy: Done")
	})
}

func NewSessionManager() *SessionManager {
	sm := &SessionManager{}
	sm.sessions = make(map[int64]*Session)
	return sm
}

func NewSession(codec protocol.Codec, sendChanSize int) *Session {
	return newSession(codec,sendChanSize,nil)
}

func newSession(codec protocol.Codec, sendChanSize int, sm *SessionManager) *Session {
	session := &Session{}
	session.id = atomic.AddInt64(&sessionID,1)
	if sendChanSize > 0 {
		session.sendChan = make(chan interface{},sendChanSize)
		go session.sendLoop()
	}
	session.closeFlag = 0
	session.codec = codec
	session.sm = sm

	//fmt.Printf("newSession sm :%v\n",session.sm)

	return session
}

var SessionClosedError = errors.New("Session Closed")
var SessionBlockedError = errors.New("Session Blocked")

func (s *Session) ID() int64 {
	return s.id
}

func (s *Session) Receive() (interface{},error) {
	//log.Printf("[Receive] session[%+v] try to get lock\n",s)
	s.recvMu.Lock()
	//log.Printf("[Receive] session[%+v]has  got a lock\n",s)
	defer s.recvMu.Unlock()

	msg,err := s.codec.Receive()
	if err != nil {
		return nil,err
	}

	//log.Printf("[Receive] session[%+v]has  release a lock\n",s)
	return msg,nil
}

func (s *Session) Send(msg interface{}) error {
	s.sendMu.Lock()
	defer s.sendMu.Unlock()
	if s.sendChan == nil {
		err := s.codec.Send(msg)
		//log.Printf("[Send sync] Session:[%+v] msg:%v\n",s,msg)
		return err
	}
	//log.Printf("[Send] Session:[%+v] msg:%v\n",s,msg)
	select {
	case s.sendChan <- msg:
		return nil
	default:
		return SessionBlockedError
	}
}

func (s *Session) Close() error {
	log.Printf("Session Close session:[%+v]\n",s)
	if atomic.CompareAndSwapInt32(&s.closeFlag, 0, 1) {
		if s.sendChan != nil {
			s.sendMu.Lock()
			s.clearSendChanBuff()
			close(s.sendChan)
			s.sendMu.Unlock()
		}

		log.Printf("Session[%+v] try to close protocol.....",s)

		err := s.codec.Close()
		log.Printf("Session[%+v] close protocol.....",s)
		if s.sm != nil {
			go func(){
				s.sm.Del(s.id)
			}()
		}

		if err != nil {
			return err
		}
	}
	return SessionClosedError
}

func (s *Session) clearSendChanBuff() error {
	l := len(s.sendChan)
	for i:=0; i<l;i++ {
		msg := <-s.sendChan
		if err := s.codec.Send(msg); err != nil {
			return err
		}
	}
	return nil
}

func (s *Session) sendLoop() {
	defer s.Close()
	for{
		select {
		case msg,ok := <-s.sendChan:
			if !ok {//通道关闭
				log.Println("sendLoop return 1")
				return
			}
			if err := s.codec.Send(msg); err != nil {
				log.Printf("sendLoop return 2,error:%v\n",err)
				return
			}
		}
	}
}

