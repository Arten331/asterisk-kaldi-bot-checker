package memdb

import (
	"strings"

	"github.com/Arten331/bot-checker/internal/domain/phrase"
	"github.com/hashicorp/go-memdb"
	"github.com/pkg/errors"
)

const (
	StopPhraseTable = "stopPhrases"
	StopPhraseIndex = "id"
)

type Repository struct {
	db *memdb.MemDB
}

func NewPhraseMemDBRepository() (Repository, error) {
	schema := &memdb.DBSchema{
		Tables: map[string]*memdb.TableSchema{
			StopPhraseTable: {
				Name: StopPhraseTable,
				Indexes: map[string]*memdb.IndexSchema{
					StopPhraseIndex: {
						Name:    StopPhraseIndex,
						Unique:  true,
						Indexer: &memdb.StringFieldIndex{Field: "Phrase"},
					},
				},
			},
		},
	}

	db, err := memdb.NewMemDB(schema)
	if err != nil {
		return Repository{}, err
	}

	nr := Repository{
		db: db,
	}

	return nr, err
}

func (n *Repository) ReadAll() ([]*phrase.StopPhrase, error) {
	tx := n.db.Txn(false)
	defer tx.Abort()

	res, err := tx.Get(StopPhraseTable, StopPhraseIndex)
	if err != nil {
		return nil, errors.Wrap(err, phrase.ErrPhraseNotFound.Error())
	}

	allPhrases := make([]*phrase.StopPhrase, 0)

	for obj := res.Next(); obj != nil; obj = res.Next() {
		p, ok := obj.(*phrase.StopPhrase)
		if ok {
			allPhrases = append(allPhrases, p)
		}
	}

	return allPhrases, nil
}

func (n *Repository) Find(find string) (*phrase.StopPhrase, error) {
	var (
		res memdb.ResultIterator
		err error
	)

	tx := n.db.Txn(false)
	defer tx.Abort()

	res, err = tx.Get(StopPhraseTable, StopPhraseIndex, find)
	if err != nil {
		return nil, errors.Wrap(err, phrase.ErrPhraseNotFound.Error())
	}

	obj := res.Next()
	if obj == nil {
		return nil, phrase.ErrPhraseNotFound
	}

	p, ok := obj.(*phrase.StopPhrase)
	if !ok {
		return nil, phrase.ErrPhraseNotFound
	}

	return p, nil
}

func (n *Repository) FindCloser(find string) (*phrase.StopPhrase, error) {
	var (
		res memdb.ResultIterator
		err error
	)

	tx := n.db.Txn(false)
	defer tx.Abort()

	res, err = tx.LowerBound(StopPhraseTable, StopPhraseIndex, find)
	if err != nil {
		return nil, errors.Wrap(err, phrase.ErrPhraseNotFound.Error())
	}

	obj := res.Next()
	if obj == nil {
		return nil, phrase.ErrPhraseNotFound
	}

	p, ok := obj.(*phrase.StopPhrase)
	if !ok {
		return nil, phrase.ErrPhraseNotFound
	}

	if strings.Contains(p.Phrase, find) {
		return p, nil
	}

	res, err = tx.ReverseLowerBound(StopPhraseTable, StopPhraseIndex, find)
	if err != nil {
		return nil, errors.Wrap(err, phrase.ErrPhraseNotFound.Error())
	}

	obj = res.Next()
	if obj == nil {
		return nil, phrase.ErrPhraseNotFound
	}

	p, ok = obj.(*phrase.StopPhrase)
	if !ok {
		return nil, phrase.ErrPhraseNotFound
	}

	return p, nil
}

func (n *Repository) Load(phrases []*phrase.StopPhrase) error {
	var err error

	tx := n.db.Txn(true)
	defer tx.Abort()

	for _, p := range phrases {
		err = tx.Insert(StopPhraseTable, p)
		if err != nil {
			err = errors.Wrap(phrase.ErrLoadPhrase, err.Error())

			return err
		}
	}

	tx.Commit()

	return err
}

func (n *Repository) Truncate() error {
	tx := n.db.Txn(true)
	_, _ = tx.DeleteAll(StopPhraseTable, StopPhraseIndex)

	tx.Commit()

	return nil
}
