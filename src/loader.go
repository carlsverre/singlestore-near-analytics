package src

import (
	"database/sql"
	"fmt"
	"io"
	"io/ioutil"
	"sort"
	"strings"
	"sync"
	"unicode"

	"github.com/go-sql-driver/mysql"
	"github.com/hamba/avro"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
)

type Stream struct {
	touched       bool
	table         string
	readID        string
	loadDataQuery string
	w             *avro.Encoder
	pw            io.WriteCloser
	pr            io.Reader
}

func NewStream(model ModelInfo) *Stream {
	pr, pw := io.Pipe()
	w := avro.NewEncoderForSchema(model.Schema, pw)

	var columnMap []string
	for fieldName, columnName := range model.FieldMap {
		columnMap = append(columnMap, fmt.Sprintf("%s <- %s", columnName, fieldName))
	}
	sort.Strings(columnMap)

	readID := uuid.NewV4().String()
	query := fmt.Sprintf(`
		LOAD DATA LOCAL INFILE 'Reader::%s'
		REPLACE INTO TABLE %s
		FORMAT AVRO
		( %s )
		SCHEMA '%s'
		ERRORS HANDLE '%s'
	`, readID, model.Table, strings.Join(columnMap, ", "), model.Schema.String(), model.Table)

	return &Stream{
		table:         model.Table,
		readID:        readID,
		loadDataQuery: query,
		w:             w,
		pw:            pw,
		pr:            pr,
	}
}

func (s *Stream) LoadData(sdbConn *sql.DB) error {
	mysql.RegisterReaderHandler(s.readID, func() io.Reader { return s.pr })
	defer mysql.DeregisterReaderHandler(s.readID)

	_, err := sdbConn.Exec(s.loadDataQuery)
	return err
}

func isInBasicMultilingualPlane(r rune) bool {
	return r <= 0xffff
}

// BMP represents all runes in the Basic Multilingual Plane
// Runes above this range are not supported until SingleStore 7.5
var BMP = &unicode.RangeTable{
	R16: []unicode.Range16{
		{0x0000, 0xffff, 1},
	},
}

var NotBMP = runes.NotIn(BMP)

var MapNotBMP = runes.Map(func(r rune) rune {
	if NotBMP.Contains(r) {
		return '�'
	}
	return r
})

func (s *Stream) WriteRow(row Model) error {
	var err error
	switch r := row.(type) {
	case *ActionReceiptAction:
		r.Args, _, err = transform.String(MapNotBMP, r.Args)
		if err != nil {
			panic(fmt.Sprintf("failed to sanitize non-bmp characters in string %q", r.Args))
		}
	case *TransactionAction:
		r.Args, _, err = transform.String(MapNotBMP, r.Args)
		if err != nil {
			panic(fmt.Sprintf("failed to sanitize non-bmp characters in string %q", r.Args))
		}
	}

	return s.w.Encode(row)
}

func (s *Stream) Touch() {
	s.touched = true
}

func (s *Stream) Close() error {
	return s.pw.Close()
}

type LoadErr struct {
	table string
	err   error
}

type Loader struct {
	streams    map[string]*Stream
	streamErrs chan LoadErr
	wg         *sync.WaitGroup
}

func NewLoader(sdbConn *sql.DB) *Loader {
	l := &Loader{
		streams:    make(map[string]*Stream),
		streamErrs: make(chan LoadErr),
		wg:         &sync.WaitGroup{},
	}

	for _, model := range Models {
		stream := NewStream(model)
		l.streams[model.Table] = stream
		l.wg.Add(1)
		go func(stream *Stream) {
			err := stream.LoadData(sdbConn)
			l.wg.Done()
			if err != nil {
				l.streamErrs <- LoadErr{table: stream.table, err: err}

				// flush the stream so the writer side of the pipe doesn't deadlock
				io.Copy(ioutil.Discard, stream.pr)
			}
		}(stream)
	}

	return l
}

func (l *Loader) Touch(table string) error {
	s, ok := l.streams[table]
	if !ok {
		return errors.Errorf("no table with name %s", table)
	}

	s.Touch()
	return nil
}

func (l *Loader) UntouchedTables() []string {
	out := make([]string, 0)
	for _, stream := range l.streams {
		if !stream.touched {
			out = append(out, stream.table)
		}
	}
	return out
}

func (l *Loader) WriteRow(table string, row Model) error {
	s, ok := l.streams[table]
	if !ok {
		return errors.Errorf("no table with name %s", table)
	}

	return s.WriteRow(row)
}

func (l *Loader) Error() error {
	errs := make([]LoadErr, 0)
outer:
	for {
		select {
		case err := <-l.streamErrs:
			errs = append(errs, err)
		default:
			break outer
		}
	}
	if len(errs) > 0 {
		fmt.Printf("%d loads failed\n", len(errs))
		for i, err := range errs {
			fmt.Printf("\t%d: table %s error %+v\n", i, err.table, err.err)
		}
		return errs[0].err
	}
	return nil
}

func (l *Loader) Close() error {
	err := l.Error()
	if err != nil {
		return err
	}

	// no errors... should be safe to close all the streams which will cause the
	// load data queries to complete (hopefully without issue)
	for _, stream := range l.streams {
		err := stream.Close()
		if err != nil {
			return err
		}
	}

	// streams are closed, now we need to wait for the loads to finish
	l.wg.Wait()

	// final check for errors
	return l.Error()
}
