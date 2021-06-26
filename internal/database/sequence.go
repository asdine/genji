package database

import (
	"errors"

	"github.com/genjidb/genji/document"
	errs "github.com/genjidb/genji/errors"
	"github.com/genjidb/genji/internal/stringutil"
)

const (
	SequenceTableName = "__genji_sequence"
)

var sequenceTableInfo = &TableInfo{
	TableName: SequenceTableName,
	StoreName: []byte(SequenceTableName),
	FieldConstraints: []*FieldConstraint{
		{
			Path: document.Path{
				document.PathFragment{
					FieldName: "name",
				},
			},
			Type:         document.TextValue,
			IsPrimaryKey: true,
		},
		{
			Path: document.Path{
				document.PathFragment{
					FieldName: "seq",
				},
			},
			Type: document.IntegerValue,
		},
	},
}

// A Sequence manages a sequence of numbers.
// It is not thread safe.
type Sequence struct {
	Info *SequenceInfo

	CurrentValue *int64
	Cached       uint64
}

func (s *Sequence) Init(tx *Transaction) error {
	tb, err := s.GetOrCreateTable(tx)
	if err != nil {
		return err
	}

	_, err = tb.Insert(document.NewFieldBuffer().Add("name", document.NewTextValue(s.Info.Name)))
	return err
}

func (s *Sequence) Next(tx *Transaction) (int64, error) {
	if !tx.Writable {
		return 0, errors.New("cannot increment sequence on read-only transaction")
	}

	var newValue int64
	if s.CurrentValue == nil {
		newValue = s.Info.Start
	} else {
		newValue = *s.CurrentValue + s.Info.IncrementBy
	}

	if newValue < s.Info.Min {
		if !s.Info.Cycle {
			return 0, stringutil.Errorf("reached minimum value of sequence %s", s.Info.Name)
		}

		newValue = s.Info.Max
	}
	if newValue > s.Info.Max {
		if !s.Info.Cycle {
			return 0, stringutil.Errorf("reached maximum value of sequence %s", s.Info.Name)
		}

		newValue = s.Info.Min
	}

	var newLease int64

	s.Cached++

	// if the number of cached values is less than or equal to the cache,
	// we don't increase the lease.
	if s.CurrentValue != nil && s.Cached <= s.Info.Cache {
		s.CurrentValue = &newValue
		return newValue, nil
	}

	// we need to reset the number of cached values to 1
	if s.CurrentValue != nil {
		s.Cached = 1
	}

	// calculate the new lease depending on the direction
	// of the sequence
	if s.Info.IncrementBy > 0 {
		newLease = newValue + int64(s.Info.Cache) - 1
		if newLease > s.Info.Max {
			newLease = s.Info.Max
		}
	} else {
		newLease = newValue - int64(s.Info.Cache) + 1
		if newLease < s.Info.Min {
			newLease = s.Info.Min
		}
	}

	// store the new lease
	err := s.SetLease(tx, s.Info.Name, newLease)
	if err != nil {
		return 0, err
	}

	s.CurrentValue = &newValue
	return newValue, nil
}

func (s *Sequence) SetLease(tx *Transaction, name string, v int64) error {
	tb, err := s.GetOrCreateTable(tx)
	if err != nil {
		return err
	}

	_, err = tb.Replace([]byte(name),
		document.NewFieldBuffer().
			Add("name", document.NewTextValue(name)).
			Add("seq", document.NewIntegerValue(v)),
	)
	return err
}

func (s *Sequence) GetOrCreateTable(tx *Transaction) (*Table, error) {
	tb, err := tx.Catalog.GetTable(tx, SequenceTableName)
	if err == nil || err != errs.ErrTableNotFound {
		return tb, err
	}

	err = tx.Catalog.CreateTable(tx, SequenceTableName, sequenceTableInfo)
	if err != nil {
		return nil, err
	}

	return tx.Catalog.GetTable(tx, SequenceTableName)
}
