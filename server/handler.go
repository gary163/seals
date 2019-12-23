package server

type Handler interface {
	Handle(*Session)
}

type HandlerFunc func(session *Session)

func (f HandlerFunc) Handle(session *Session){
	f(session)
}
