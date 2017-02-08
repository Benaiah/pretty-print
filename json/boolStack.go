package prettyPrintJson

// This type keeps track of whether we're in an array or not. It works
// as follows:
//
// - When we hit an opening object delimiter, we push `false`
// - When we hit an opening array delimiter, we push `true`
// - When we hit a closing delimiter, we pop
//
// This is needed in order to tell if an incoming string is a key or a
// value. Without a stack to keep track of it, arrays with nested
// structures would be impossible to keep track of.
type boolStack struct {
	s []bool
}

func newBoolStack() *boolStack {
	return &boolStack{s: make([]bool, 0)}
}

func (s *boolStack) push(b bool) {
	s.s = append(s.s, b)
}

func (s *boolStack) pop() bool {
	l := len(s.s)
	// We don't care about over-popping, as the json decoder will already
	// error out if there are mismatched tokens. When the array is empty,
	// we just return `false`
	if l == 0 {
		return false
	}

	res := s.s[l-1]
	s.s = s.s[:l-1]
	return res
}

func (s *boolStack) peek() bool {
	l := len(s.s)
	if l == 0 {
		return false
	}

	res := s.s[l-1]
	return res
}

func (s *boolStack) len() int {
	return len(s.s)
}
