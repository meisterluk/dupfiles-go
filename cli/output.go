package main

import (
	"fmt"
	"io"
)

type Output interface {
	Print(text string) (int, error)
	Println(text string) (int, error)
	Printf(format string, args ...interface{}) (int, error)
	Printfln(format string, args ...interface{}) (int, error)
}

type plainOutput struct {
	device io.Writer
}

func (o *plainOutput) Print(text string) (int, error) {
	return o.device.Write([]byte(text))
}

func (o *plainOutput) Println(text string) (int, error) {
	n1, err1 := o.device.Write([]byte(text))
	if err1 != nil {
		return n1, err1
	}
	n2, err2 := o.device.Write([]byte{'\n'})
	return n1 + n2, err2
}

func (o *plainOutput) Printf(format string, args ...interface{}) (int, error) {
	return o.device.Write([]byte(fmt.Sprintf(format, args...)))
}

func (o *plainOutput) Printfln(format string, args ...interface{}) (int, error) {
	return o.device.Write([]byte(fmt.Sprintf(format+"\n", args...)))
}
