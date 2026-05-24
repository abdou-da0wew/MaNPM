package ui

import (
	"fmt"
	"os"
	"strings"
)

type Progress struct {
	total    int
	complete int
	msg      string
	done     bool
}

func NewProgress(total int) *Progress {
	return &Progress{total: total}
}

func (p *Progress) Start() {
	p.complete = 0
	p.done = false
	p.line()
}

func (p *Progress) Inc(name string) {
	p.complete++
	p.msg = name
	p.line()
}

func (p *Progress) Update(msg string) {
	p.msg = msg
	p.line()
}

func (p *Progress) line() {
	if p.total == 0 || p.done {
		return
	}
	w := 10
	f := p.complete * w / p.total
	if f > w {
		f = w
	}
	bar := "[" + strings.Repeat("#", f) + strings.Repeat("·", w-f) + "]"
	name := p.msg
	if name == "" {
		name = "..."
	}
	fmt.Fprintf(os.Stderr, "\033[2K\r  %s %s%d/%d %s%s\033[K", Gray+bar+Reset, Cyan, p.complete, p.total, name, Reset)
}

func (p *Progress) Done(msg string) {
	if p.done || p.total == 0 {
		return
	}
	p.done = true
	fmt.Fprintf(os.Stderr, "\033[2K\r%s%s%s %s\n", Bold, Green, "✓", Reset+msg)
}
