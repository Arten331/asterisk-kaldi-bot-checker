package phrase

import (
	"errors"
)

var (
	ErrPhraseNotFound = errors.New("stop phrase not found")
	ErrLoadPhrase     = errors.New("unable load stop phrase")
)

type Repository interface {
	Find(find string) (*StopPhrase, error)
	FindCloser(find string) (*StopPhrase, error)
	ReadAll() ([]*StopPhrase, error)
	Load(phrase []*StopPhrase) error
	Truncate() error
}
