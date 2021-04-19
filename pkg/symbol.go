package wdlparser

type Symboler interface {
	GetName() string
	GetType() string
	GetValue() interface{}
}

type Symbol struct {
	name  string
	typ   string
	value string
}

func (s *Symbol) GetName() string {
	return s.name
}

func (s *Symbol) SetName(name string) {
	s.name = name
}

func (s *Symbol) GetType() string {
	return s.typ
}

func (s *Symbol) SetType(t string) {
	s.typ = t
}

func (s *Symbol) GetValue() interface{} {
	return s.value
}

func (s *Symbol) SetValue(val string) {
	s.value = val
}

type ScopedSymbol struct {
	Scope
	Symbol
}
