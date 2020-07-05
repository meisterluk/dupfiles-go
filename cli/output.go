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
	Device io.Writer
}

// Print writes text to this output stream
func (o *PlainOutput) Print(text string) (int, error) {
	return o.Device.Write([]byte(text))
}

// Println writes text and a line break to this output stream
func (o *PlainOutput) Println(text string) (int, error) {
	n1, err1 := o.Device.Write([]byte(text))
	if err1 != nil {
		return n1, err1
	}
	n2, err2 := o.Device.Write([]byte{'\n'})
	return n1 + n2, err2
}

// Printf writes text to this output stream and the text is generated
// by applying args to the given format string.
func (o *PlainOutput) Printf(format string, args ...interface{}) (int, error) {
	return o.Device.Write([]byte(fmt.Sprintf(format, args...)))
}

// Printfln writes text and a line break to this output stream and the text is generated
// by applying args to the given format string.
func (o *PlainOutput) Printfln(format string, args ...interface{}) (int, error) {
	return o.Device.Write([]byte(fmt.Sprintf(format+"\n", args...)))
}
