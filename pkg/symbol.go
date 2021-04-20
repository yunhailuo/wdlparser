package wdlparser

type symboler interface {
	GetName() string
	GetType() string
	GetValue() interface{}
}

type symbol struct {
	name  string
	typ   string
	value string
}

func (s *symbol) GetName() string {
	return s.name
}

func (s *symbol) SetName(name string) {
	s.name = name
}

func (s *symbol) GetType() string {
	return s.typ
}

func (s *symbol) SetType(t string) {
	s.typ = t
}

func (s *symbol) GetValue() interface{} {
	return s.value
}

func (s *symbol) SetValue(val string) {
	s.value = val
}

type scopedSymbol struct {
	scope
	symbol
}
