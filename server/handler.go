package server

type Handler interface {
	Handle(session *Session)
}

type HandlerFunc func(session *Session)

func (f HandlerFunc) Handle(session *Session){
	f(session)
}
