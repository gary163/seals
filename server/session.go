package server

import (
	"errors"
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
	sendMu    sync.RWMutex //使用读写锁，发送时可以并发写入buffchan,提高并发能力
	recvMu    sync.Mutex
	sm        *SessionManager
	closeChan chan int
	closeCallBackHead *callbackList
	closeMu  sync.Mutex
}

//以单链表记录关闭调用链
type callbackList struct {
	handler   interface{}
	key       interface{}
	callback  func()
	next      *callbackList
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
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if _,ok := sm.sessions[sid]; ok {
		delete(sm.sessions,sid)
		sm.wg.Done()
	}
}

func (sm *SessionManager) Destroy() {
	sm.destroyOnce.Do(func(){
		sm.mu.Lock()
		for _,session := range sm.sessions {
			session.Close()
		}
		sm.mu.Unlock()
		sm.wg.Wait()
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
	session.closeChan = make(chan int)
	if sendChanSize > 0 {
		session.sendChan = make(chan interface{},sendChanSize)
		go session.sendLoop()
	}
	session.closeFlag = 0
	session.codec = codec
	session.sm = sm
	return session
}

var SessionClosedError = errors.New("Session Closed")
var SessionBlockedError = errors.New("Session Blocked")

func (s *Session) ID() int64 {
	return s.id
}

func (s *Session) isClosed() bool {
	return atomic.LoadInt32(&s.closeFlag) == 1
}

func (s *Session) AddCloseCallback(handler interface{}, key interface{}, callback func()) {
	if s.isClosed() {
		return
	}
	s.closeMu.Lock()
	defer s.closeMu.Unlock()

	if s.closeCallBackHead == nil {//链表的表头不存数据，只存下一个的指针
		s.closeCallBackHead = new(callbackList)
	}

	head := s.closeCallBackHead
	next := head
	tail := head

	for next != nil {
		tail = next
		next = next.next
	}

	node := &callbackList{
		key:key,
		callback:callback,
		handler:handler,
		next:nil,
	}

	tail.next = node
}

func (s *Session) DelCloseCallback(handler interface{}, key interface{}) {
	if s.isClosed() {
		return
	}
	s.closeMu.Lock()
	defer s.closeMu.Unlock()

	head := s.closeCallBackHead
	next,pre := head,head
	for next != nil {
		node := next
		if node.handler == handler && node.key == key {
			pre.next = node.next
			break
		}
		pre = node
		next = next.next
	}
}

func (s *Session) InvokeCallbackFun() {
	s.closeMu.Lock()
	defer s.closeMu.Unlock()

	head := s.closeCallBackHead
	next := head

	for next != nil {
		node := next
		if node.callback != nil {
			node.callback()
		}
		next = next.next
	}
}

func (s *Session) Receive() (interface{},error) {
	s.recvMu.Lock()
	defer s.recvMu.Unlock()

	msg,err := s.codec.Receive()
	if err != nil {
		return nil,err
	}
	return msg,nil
}

func (s *Session) Send(msg interface{}) error {
	//异步send,先写buffer chan，利用读锁来提高并发能力
	if s.sendChan != nil {
		s.sendMu.RLock()
		if s.isClosed() {
			s.sendMu.RUnlock()
			return SessionClosedError
		}

		select {
		case s.sendChan <- msg :
			s.sendMu.RUnlock()
			return nil
		default:
			s.sendMu.RUnlock()
			s.Close()
			return SessionBlockedError
		}
	}

	//同步send，利用写锁来确保原子性
	if s.isClosed() {
		return SessionClosedError
	}
	s.sendMu.Lock()
	defer s.sendMu.Unlock()

	if err := s.codec.Send(msg); err != nil {
		s.Close()
		return err
	}
	return nil
}

func (s *Session) Close() error {
	if atomic.CompareAndSwapInt32(&s.closeFlag, 0, 1) {
		close(s.closeChan)//关闭通道，让sendLoop goroutine 先退出
		if s.sendChan != nil {
			s.sendMu.Lock()
			s.clearSendChanBuff()//清除剩余的buff再关闭
			close(s.sendChan)
			s.sendMu.Unlock()
		}

		err := s.codec.Close()
		if s.sm != nil {
			go func(){
				s.InvokeCallbackFun()
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
				return
			}
			if err := s.codec.Send(msg); err != nil {
				return
			}
		case <- s.closeChan:
			return
		}
	}
}

