package handlers

type Func struct {
	name    string
	params  []string
	returns []string
}

func NewFunc() *Func {
	return &Func{
		params:  []string{},
		returns: []string{},
	}
}

func (f *Func) SetName(name string) {
	f.name = name
}

func (f *Func) AddParameter(typ string) {
	f.params = append(f.params, typ)
}

func (f *Func) AddReturn(typ string) {
	f.returns = append(f.returns, typ)
}

func (l *Func) Equals(r *Func) bool {
	if l.name != r.name ||
		len(l.params) != len(r.params) ||
		len(l.returns) != len(r.returns) {
		return false
	}

	// there is a more optimized way to do this
	// with less iterations but I'm lazy and it's not worth it
	for idx := range l.params {
		if l.params[idx] != r.params[idx] {
			return false
		}
	}
	for idx := range l.returns {
		if l.returns[idx] != r.returns[idx] {
			return false
		}
	}
	return true
}

func (f *Func) GetFunctionName() string {
	return f.name
}

func (f *Func) GetParameters() []string {
	return f.params
}

func (f *Func) GetReturns() []string {
	return f.returns
}
