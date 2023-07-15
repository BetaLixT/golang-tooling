package handlers

import (
	"go/format"
	"strings"
)

type source struct {
	sourceCode string
}

func (s *source) Println(strs ...string) {
	s.sourceCode = s.sourceCode + strings.Join(strs, "") + "\n"
}

func (s *source) Source() []byte {
	src, err := format.Source([]byte(s.sourceCode))
	if err != nil {
		println(s.sourceCode)
		panic(err)
	}
	return src
}
