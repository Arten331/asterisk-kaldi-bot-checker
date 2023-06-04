//go:build test && !integration

package memdb

import (
	"encoding/csv"
	"io"
	"testing"

	testdata "github.com/Arten331/bot-checker/data/test"
	"github.com/Arten331/bot-checker/internal/domain/phrase"
	"github.com/stretchr/testify/require"
)

func TestMemDBRepository(t *testing.T) {
	memRepo, err := NewPhraseMemDBRepository()

	require.NoError(t, err)

	tfs := testdata.GetTestFS()

	phrasesFile, err := tfs.Open("records.csv")
	if err != nil {
		require.NoError(t, err)
	}

	reader := csv.NewReader(phrasesFile)
	phrases := make([]*phrase.StopPhrase, 0, 0)
	for {
		row, err := reader.Read()
		if err == io.EOF || cap(row) == 1 {
			break
		}

		if row[0] != "" && row[1] != "" {
			phrases = append(phrases, phrase.New(row[0], row[1]))
		}
	}

	err = memRepo.Load(phrases)
	if err != nil {
		require.NoError(t, err)
	}

	resAll, err := memRepo.ReadAll()
	require.NoError(t, err)

	require.Len(t, resAll, 67)

	f, err := memRepo.Find("абонент занят")
	require.NotNil(t, f)
	require.Equal(t, f.Phrase, "абонент занят")

	f, err = memRepo.FindCloser("абонент временно недоступен попробуйте")
	require.NotNil(t, f)
	require.Equal(t, f.Phrase, "абонент временно недоступен")

	f, err = memRepo.FindCloser("абонент")
	require.NotNil(t, f)
	require.Equal(t, f.Phrase, "абонент в сети")

	err = memRepo.Truncate()
	require.NoError(t, err)

	resAll, err = memRepo.ReadAll()
	require.NoError(t, err)

	require.Len(t, resAll, 0)
}
