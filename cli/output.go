package main

import (
	"fmt"
	"io"
)

// Output defines a uniform interface to write to some stream
type Output interface {
	Print(text string) (int, error)
	Println(text string) (int, error)
	Printf(format string, args ...interface{}) (int, error)
	Printfln(format string, args ...interface{}) (int, error)
}

// PlainOutput is a specific Output device which writes data in a raw format
type PlainOutput struct {
	device io.Writer
}

func (o *PlainOutput) Print(text string) (int, error) {
	return o.device.Write([]byte(text))
}

func (o *PlainOutput) Println(text string) (int, error) {
	n1, err1 := o.device.Write([]byte(text))
	if err1 != nil {
		return n1, err1
	}
	n2, err2 := o.device.Write([]byte{'\n'})
	return n1 + n2, err2
}

func (o *PlainOutput) Printf(format string, args ...interface{}) (int, error) {
	return o.device.Write([]byte(fmt.Sprintf(format, args...)))
}

func (o *PlainOutput) Printfln(format string, args ...interface{}) (int, error) {
	return o.device.Write([]byte(fmt.Sprintf(format+"\n", args...)))
}
