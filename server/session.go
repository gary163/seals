package server

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"

	"github.com/gary163/seals/protocol"
)

type SessionManager struct {
	sessions map[int64]*Session
	mu sync.RWMutex
	wg sync.WaitGroup
}

type Session struct {
	id        int64
	sendChan  chan interface{}
	protocol  protocol.Protocol
	closeFlag int32
	sendMu    sync.RWMutex
	recvMu    sync.Mutex
	ctx       context.Context
	sm        *SessionManager
}

func (sm *SessionManager) NewSession(protocol protocol.Protocol, sendChanSize int, ctx context.Context) *Session {
	session := newSession(protocol , sendChanSize , ctx , sm)
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
	sm.mu.Lock()
	defer sm.mu.Unlock()
	delete(sm.sessions,sid)
	sm.wg.Done()
}

func (sm *SessionManager) Wait() {
	sm.wg.Wait()
}

func NewSessionManager() *SessionManager {
	sm := &SessionManager{}
	sm.sessions = make(map[int64]*Session)
	return sm
}

func NewSession(protocol protocol.Protocol, sendChanSize int) *Session {
	return newSession(protocol,sendChanSize,nil,nil)
}

func newSession(protocol protocol.Protocol, sendChanSize int, ctx context.Context, sm *SessionManager) *Session {
	session := &Session{}
	var sid int64
	session.id = atomic.AddInt64(&sid,1)
	if sendChanSize > 0 {
		session.sendChan = make(chan interface{},sendChanSize)
		go session.sendLoop()
	}
	session.closeFlag = 0
	session.ctx = ctx
	session.protocol = protocol
	session.sm = sm

	return session
}

var SessionClosedError = errors.New("Session Closed")
var SessionBlockedError = errors.New("Session Blocked")

func (s *Session) Receive() (interface{},error) {
	s.recvMu.Lock()
	defer s.recvMu.Unlock()

	msg,err := s.protocol.Receive()
	if err != nil {
		return nil,err
	}
	return msg,nil
}

func (s *Session) Send(msg interface{}) error {
	s.sendMu.RLock()
	defer s.sendMu.RUnlock()

	select {
	case s.sendChan <- msg:
		return nil
	default:
		return SessionBlockedError
	}
}

func (s *Session) Close() error {
	if atomic.CompareAndSwapInt32(&s.closeFlag, 0, 1) {
		if s.sendChan != nil {
			s.sendMu.Lock()
			s.clearSendChanBuff()
			close(s.sendChan)
			s.sendMu.Unlock()
			s.sm.Del(s.id)
			if err := s.protocol.Close(); err != nil {
				return err
			}
		}
	}
	return SessionClosedError
}

func (s *Session) clearSendChanBuff() error {
	l := len(s.sendChan)
	for i:=0; i<l;i++ {
		msg := <-s.sendChan
		if err := s.protocol.Send(msg); err != nil {
			return err
		}
	}
	return nil
}

func (s *Session) sendLoop() {
	defer s.Close()
	for{
		select {
		case <-s.ctx.Done()://执行了close
			return
		case msg,ok := <-s.sendChan:
			if !ok {//通道关闭
				return
			}
			if err := s.protocol.Send(msg); err != nil {
				return
			}
		}
	}
}

