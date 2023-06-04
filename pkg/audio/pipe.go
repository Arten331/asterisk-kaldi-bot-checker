package audio

import (
	"context"
	"io"
	"os"
)

const BUFFSIZE = 8000

type PipeCommand interface {
	Name() string
	Handle(ctx context.Context, errChan ErrChan) (in io.Writer, out io.Reader)
}

type PipeErrer interface {
	Command() PipeCommand
	Error() string
}

type ErrChan chan PipeErrer

type Pipe struct {
	StdErr   ErrChan
	StdIn    io.Writer
	StdOut   io.Reader
	Commands []PipeCommand
}

func NewPipe(commands []PipeCommand) (*Pipe, error) {
	rp, wp, err := os.Pipe()
	if err != nil {
		return nil, err
	}

	p := Pipe{
		StdErr:   make(ErrChan, 1),
		StdIn:    wp,
		StdOut:   rp,
		Commands: commands,
	}

	return &p, nil
}

func (p *Pipe) Run(ctx context.Context) error {
	for _, command := range p.Commands {
		var curReader io.Reader

		curReader = p.StdOut
		stdIn, stdOut := command.Handle(ctx, p.StdErr)

		rp, wp, err := os.Pipe()
		if err != nil {
			return err
		}

		go func() {
			buf := make([]byte, BUFFSIZE)

			for {
				_, err := io.ReadFull(stdOut, buf)
				if err != nil {
					p.StdErr <- &CommandErr{command: command, err: err.Error()}
				}

				_, _ = wp.Write(buf)
			}
		}()

		go func(curReader io.Reader) {
			for {
				buf := make([]byte, BUFFSIZE)

				_, err := io.ReadFull(curReader, buf)
				if err != nil {
					p.StdErr <- &CommandErr{command: command, err: err.Error()}
				}

				_, _ = stdIn.Write(buf)
			}
		}(curReader)

		p.StdOut = rp
	}

	return nil
}

func (p *Pipe) Write(dat []byte) error {
	_, err := p.StdIn.Write(dat)
	if err != nil {
		return err
	}

	return nil
}

type CommandErr struct {
	command PipeCommand
	err     string
}

func NewErr(command PipeCommand, err string) *CommandErr {
	return &CommandErr{command: command, err: err}
}

func (p *CommandErr) Command() PipeCommand {
	return p.command
}

func (p *CommandErr) Error() string {
	return p.err
}
