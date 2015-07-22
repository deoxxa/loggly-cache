package main

import (
	"github.com/julienc91/sizer"
	"gopkg.in/alecthomas/kingpin.v2"
)

type sizerValue sizer.Size

func (s *sizerValue) Set(str string) error {
	v, err := sizer.ParseStringSize(str)
	if err != nil {
		return err
	}

	*s = sizerValue(v)

	return nil
}

func (s *sizerValue) String() string {
	return ((*sizer.Size)(s)).String()
}

func kpSizerValue(s kingpin.Settings) *sizer.Size {
	var v sizer.Size
	s.SetValue((*sizerValue)(&v))
	return &v
}
