package document

import (
	"bufio"
	"errors"
	"io"
)

// ErrStreamClosed is used to indicate that a stream must be closed.
var ErrStreamClosed = errors.New("stream closed")

// An Iterator can iterate over documents.
type Iterator interface {
	// Iterate goes through all the documents and calls the given function by passing each one of them.
	// If the given function returns an error, the iteration stops.
	Iterate(func(d Document) error) error
}

// NewIterator creates an iterator that iterates over documents.
func NewIterator(documents ...Document) Iterator {
	return documentsIterator(documents)
}

type documentsIterator []Document

func (rr documentsIterator) Iterate(fn func(d Document) error) error {
	var err error

	for _, d := range rr {
		err = fn(d)
		if err != nil {
			return err
		}
	}

	return nil
}

// The IteratorFunc type is an adapter to allow the use of ordinary functions as Iterators.
// If f is a function with the appropriate signature, HandlerFunc(f) is a Handler that calls f.
type IteratorFunc func(func(d Document) error) error

// Iterate calls f(fn).
func (f IteratorFunc) Iterate(fn func(d Document) error) error {
	return f(fn)
}

// IteratorToJSON encodes all the documents of an iterator to JSON stream.
func IteratorToJSON(w io.Writer, s Iterator) error {
	buf := bufio.NewWriter(w)
	defer buf.Flush()

	return s.Iterate(func(d Document) error {
		data, err := jsonDocument{d}.MarshalJSON()
		if err != nil {
			return err
		}

		_, err = buf.Write(data)
		return err
	})
}

// IteratorToJSONArray encodes all the documents of an iterator to a JSON array.
func IteratorToJSONArray(w io.Writer, s Iterator) error {
	buf := bufio.NewWriter(w)

	buf.WriteByte('[')

	first := true
	err := s.Iterate(func(d Document) error {
		if !first {
			buf.WriteString(", ")
		} else {
			first = false
		}

		data, err := jsonDocument{d}.MarshalJSON()
		if err != nil {
			return err
		}

		_, err = buf.Write(data)
		return err
	})
	if err != nil {
		return err
	}

	buf.WriteByte(']')
	return buf.Flush()
}

// Stream reads documents of an iterator one by one and passes them
// through a list of functions for transformation.
type Stream struct {
	it Iterator
	op StreamOperator
}

// NewStream creates a stream using the given iterator.
func NewStream(it Iterator) Stream {
	return Stream{it: it}
}

// IsEmpty returns whether the stream doesn't contain any iterator.
func (s Stream) IsEmpty() bool {
	return s.it == nil
}

// Iterate calls the underlying iterator's iterate method.
// If this stream was created using the Pipe method, it will apply fn
// to any document passed by the underlying iterator.
// If fn returns a document, it will be passed to the next stream.
// If it returns a nil document, the document will be ignored.
// If it returns an error, the stream will be interrupted and that error will bubble up
// and returned by fn, unless that error is ErrStreamClosed, in which case
// the Iterate method will stop the iteration and return nil.
// It implements the Iterator interface.
func (s Stream) Iterate(fn func(d Document) error) error {
	if s.it == nil {
		return nil
	}

	if s.op == nil {
		return s.it.Iterate(fn)
	}

	opFn := s.op()

	err := s.it.Iterate(func(d Document) error {
		d, err := opFn(d)
		if err != nil {
			return err
		}

		if d == nil {
			return nil
		}

		return fn(d)
	})
	if err != ErrStreamClosed {
		return err
	}

	return nil
}

// Pipe creates a new Stream who can read its data from s and apply
// op to every document passed by its Iterate method.
func (s Stream) Pipe(op StreamOperator) Stream {
	return Stream{
		it: s,
		op: op,
	}
}

// Map applies fn to each received document and passes it to the next stream.
// If fn returns an error, the stream is interrupted.
func (s Stream) Map(fn func(d Document) (Document, error)) Stream {
	return s.Pipe(func() func(d Document) (Document, error) {
		return fn
	})
}

// Filter each received document using fn.
// If fn returns true, the document is kept, otherwise it is skipped.
// If fn returns an error, the stream is interrupted.
func (s Stream) Filter(fn func(d Document) (bool, error)) Stream {
	return s.Pipe(func() func(d Document) (Document, error) {
		return func(d Document) (Document, error) {
			ok, err := fn(d)
			if err != nil {
				return nil, err
			}

			if !ok {
				return nil, nil
			}

			return d, nil
		}
	})
}

// Limit interrupts the stream once the number of passed documents have reached n.
func (s Stream) Limit(n int) Stream {
	return s.Pipe(func() func(d Document) (Document, error) {
		var count int

		return func(d Document) (Document, error) {
			if count < n {
				count++
				return d, nil
			}

			return nil, ErrStreamClosed
		}
	})
}

// Offset ignores n documents then passes the subsequent ones to the stream.
func (s Stream) Offset(n int) Stream {
	return s.Pipe(func() func(d Document) (Document, error) {
		var skipped int

		return func(d Document) (Document, error) {
			if skipped < n {
				skipped++
				return nil, nil
			}

			return d, nil
		}
	})
}

// Append adds the given iterator to the stream.
func (s Stream) Append(it Iterator) Stream {
	if mr, ok := s.it.(multiIterator); ok {
		mr.iterators = append(mr.iterators, it)
		s.it = mr
	} else {
		s.it = multiIterator{
			iterators: []Iterator{s, it},
		}
	}

	return s
}

// Count counts all the documents from the stream.
func (s Stream) Count() (int, error) {
	counter := 0

	err := s.Iterate(func(d Document) error {
		counter++
		return nil
	})

	return counter, err
}

// First runs the stream, returns the first document found and closes the stream.
// If the stream is empty, all return values are nil.
func (s Stream) First() (d Document, err error) {
	err = s.Iterate(func(doc Document) error {
		d = doc
		return ErrStreamClosed
	})

	if err == ErrStreamClosed {
		err = nil
	}

	return
}

// GroupBy tags each document with the group they belong to. The group is determined by the value returned by groupFn.
func (s Stream) GroupBy(groupFn func(d Document) (Value, error)) Stream {
	return s.Pipe(func() func(d Document) (Document, error) {
		var gd groupedDocument

		return func(d Document) (Document, error) {
			v, err := groupFn(d)
			if err != nil && err != ErrFieldNotFound {
				return nil, err
			}
			if err == ErrFieldNotFound {
				v = NewNullValue()
			}

			gd.group = v
			gd.Document = d

			return &gd, nil
		}
	})
}

// Aggregate builds a list of aggregators for each group of documents and passes each document of the stream to them.
func (s Stream) Aggregate(aggregatorBuilders ...AggregatorBuilder) Stream {
	return NewStream(IteratorFunc(func(fn func(d Document) error) error {
		aggregates := make(map[Value][]Aggregator)
		var groups []Value

		nullValue := NewNullValue()

		err := s.Iterate(func(d Document) error {
			group := nullValue

			if gd, ok := d.(*groupedDocument); ok {
				group = gd.group
			}

			aggs, ok := aggregates[group]
			if !ok {
				groups = append(groups, group)
				aggs = make([]Aggregator, len(aggregatorBuilders))
				for i, builder := range aggregatorBuilders {
					aggs[i] = builder.NewAggregator(group)
				}
				aggregates[group] = aggs
			}

			var err error
			for _, agg := range aggs {
				err = agg.Add(d)
				if err != nil {
					return err
				}
			}

			return nil
		})
		if err != nil {
			return err
		}

		for _, group := range groups {
			fb := NewFieldBuffer()
			aggs := aggregates[group]
			for _, agg := range aggs {
				err = agg.Aggregate(fb)
				if err != nil {
					return err
				}
			}

			err = fn(fb)
			if err != nil {
				return err
			}
		}

		return nil
	}))
}

// An Aggregator aggregates documents into a single one.
type Aggregator interface {
	Add(d Document) error
	Aggregate(fb *FieldBuffer) error
}

// An AggregatorBuilder can build aggregators on demand.
type AggregatorBuilder interface {
	NewAggregator(group Value) Aggregator
}

// groupedDocument tags a document with a group value.
// It is used by GroupBy to group documents in a stream.
type groupedDocument struct {
	Document

	group Value
}

// An StreamOperator is used to modify a stream.
// If a stream operator returns a document, it will be passed to the next stream.
// If it returns a nil document, the document will be ignored.
// If it returns an error, the stream will be interrupted and that error will bubble up
// and returned by this function, unless that error is ErrStreamClosed, in which case
// the Iterate method will stop the iteration and return nil.
// Stream operators can be reused, and thus, any state or side effect should be kept within the operator closure
// unless the nature of the operator prevents that.
type StreamOperator func() func(d Document) (Document, error)

type multiIterator struct {
	iterators []Iterator
}

func (m multiIterator) Iterate(fn func(d Document) error) error {
	for _, it := range m.iterators {
		err := it.Iterate(fn)
		if err != nil {
			return err
		}
	}

	return nil
}
