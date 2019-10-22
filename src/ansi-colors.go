package main

type IAnsiColor interface {
	Colorize
}

type AnsiColor struct {
}
