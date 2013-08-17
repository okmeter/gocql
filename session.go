// Copyright (c) 2012 The gocql Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package gocql

import (
	"errors"
)

// Session is the interface used by users to interact with the database.
//
// It extends the Node interface by adding a convinient query builder and
// automatically sets a default consinstency level on all operations
// that do not have a consistency level set.
type Session struct {
	Node Node
	Cons Consistency
}

// NewSession wraps an existing Node.
func NewSession(node Node) *Session {
	if s, ok := node.(*Session); ok {
		return &Session{Node: s.Node}
	}
	return &Session{Node: node, Cons: Quorum}
}

// Query can be used to build new queries that should be executed on this
// session.
func (s *Session) Query(stmt string, args ...interface{}) QueryBuilder {
	return QueryBuilder{NewQuery(stmt, args...), s}
}

// Do can be used to modify a copy of an existing query before it is
// executed on this session.
func (s *Session) Do(qry *Query) QueryBuilder {
	q := *qry
	return QueryBuilder{&q, s}
}

// Close closes all connections. The session is unuseable after this
// operation.
func (s *Session) Close() {
	s.Node.Close()
}

// ExecuteBatch executes a Batch on the underlying Node.
func (s *Session) ExecuteBatch(batch *Batch) error {
	if batch.Cons == 0 {
		batch.Cons = s.Cons
	}
	return s.Node.ExecuteBatch(batch)
}

// ExecuteQuery executes a Query on the underlying Node.
func (s *Session) ExecuteQuery(qry *Query) (*Iter, error) {
	if qry.Cons == 0 {
		qry.Cons = s.Cons
	}
	return s.Node.ExecuteQuery(qry)
}

type Query struct {
	Stmt     string
	Args     []interface{}
	Cons     Consistency
	Token    string
	PageSize int
	Trace    bool
}

func NewQuery(stmt string, args ...interface{}) *Query {
	return &Query{Stmt: stmt, Args: args}
}

type QueryBuilder struct {
	qry *Query
	ctx Node
}

// Args specifies the query parameters.
func (b QueryBuilder) Args(args ...interface{}) {
	b.qry.Args = args
}

// Consistency sets the consistency level for this query. If no consistency
// level have been set, the default consistency level of the cluster
// is used.
func (b QueryBuilder) Consistency(cons Consistency) QueryBuilder {
	b.qry.Cons = cons
	return b
}

func (b QueryBuilder) Token(token string) QueryBuilder {
	b.qry.Token = token
	return b
}

func (b QueryBuilder) Trace(trace bool) QueryBuilder {
	b.qry.Trace = trace
	return b
}

func (b QueryBuilder) PageSize(size int) QueryBuilder {
	b.qry.PageSize = size
	return b
}

func (b QueryBuilder) Exec() error {
	_, err := b.ctx.ExecuteQuery(b.qry)
	return err
}

func (b QueryBuilder) Iter() *Iter {
	iter, err := b.ctx.ExecuteQuery(b.qry)
	if err != nil {
		return &Iter{err: err}
	}
	return iter
}

func (b QueryBuilder) Scan(values ...interface{}) error {
	iter := b.Iter()
	iter.Scan(values...)
	return iter.Close()
}

type Iter struct {
	err     error
	pos     int
	rows    [][][]byte
	columns []ColumnInfo
}

func (iter *Iter) Columns() []ColumnInfo {
	return iter.columns
}

func (iter *Iter) Scan(values ...interface{}) bool {
	if iter.err != nil || iter.pos >= len(iter.rows) {
		return false
	}
	if len(values) != len(iter.columns) {
		iter.err = errors.New("count mismatch")
		return false
	}
	for i := 0; i < len(iter.columns); i++ {
		err := Unmarshal(iter.columns[i].TypeInfo, iter.rows[iter.pos][i], values[i])
		if err != nil {
			iter.err = err
			return false
		}
	}
	iter.pos++
	return true
}

func (iter *Iter) Close() error {
	return iter.err
}

type Batch struct {
	Type    BatchType
	Entries []BatchEntry
	Cons    Consistency
}

func NewBatch(typ BatchType) *Batch {
	return &Batch{Type: typ}
}

func (b *Batch) Query(stmt string, args ...interface{}) {
	b.Entries = append(b.Entries, BatchEntry{Stmt: stmt, Args: args})
}

type BatchType int

const (
	LoggedBatch   BatchType = 0
	UnloggedBatch BatchType = 1
	CounterBatch  BatchType = 2
)

type BatchEntry struct {
	Stmt string
	Args []interface{}
}

type Consistency int

const (
	Any Consistency = 1 + iota
	One
	Two
	Three
	Quorum
	All
	LocalQuorum
	EachQuorum
	Serial
	LocalSerial
)

var consinstencyNames = []string{
	0:           "default",
	Any:         "any",
	One:         "one",
	Two:         "two",
	Three:       "three",
	Quorum:      "quorum",
	All:         "all",
	LocalQuorum: "localquorum",
	EachQuorum:  "eachquorum",
	Serial:      "serial",
	LocalSerial: "localserial",
}

func (c Consistency) String() string {
	return consinstencyNames[c]
}

type ColumnInfo struct {
	Keyspace string
	Table    string
	Name     string
	TypeInfo *TypeInfo
}

type Error struct {
	Code    int
	Message string
}

func (e Error) Error() string {
	return e.Message
}

var (
	ErrNotFound    = errors.New("not found")
	ErrUnavailable = errors.New("unavailable")
	ErrProtocol    = errors.New("protocol error")
)