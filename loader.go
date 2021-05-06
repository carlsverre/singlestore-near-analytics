package main

import (
	"database/sql"
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"sync"

	"github.com/go-sql-driver/mysql"
	uuid "github.com/satori/go.uuid"
)

type Stream struct {
	table string
	w     *csv.Writer
	pw    io.WriteCloser
	pr    io.Reader
}

func NewStream(table string) *Stream {
	pr, pw := io.Pipe()
	w := csv.NewWriter(pw)
	w.Comma = '\t'

	return &Stream{
		table: table,
		w:     w,
		pw:    pw,
		pr:    pr,
	}
}

func (s *Stream) LoadData(sdbConn *sql.DB) error {
	readID := uuid.NewV4().String()
	mysql.RegisterReaderHandler(readID, func() io.Reader { return s.pr })
	defer mysql.DeregisterReaderHandler(readID)

	query := fmt.Sprintf(`
		LOAD DATA LOCAL INFILE 'Reader::%s'
		REPLACE INTO TABLE %s
		COLUMNS TERMINATED BY '\t'
		OPTIONALLY ENCLOSED BY '"'
	`, readID, s.table)

	_, err := sdbConn.Exec(query)
	return err
}

func (s *Stream) WriteRow(row []string) error {
	//fmt.Printf("got row: %+v\n", row)
	return s.w.Write(row)
}

func (s *Stream) Flush() error {
	s.w.Flush()
	return s.w.Error()
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

func NewLoader(sdbConn *sql.DB, tables []string) *Loader {
	l := &Loader{
		streams:    make(map[string]*Stream),
		streamErrs: make(chan LoadErr),
		wg:         &sync.WaitGroup{},
	}

	for _, table := range tables {
		stream := NewStream(table)
		l.streams[table] = stream
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

func (l *Loader) WriteRow(table string, row []string) error {
	s, ok := l.streams[table]
	if !ok {
		return fmt.Errorf("no table with name %s", table)
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

	// flush the streams, we need to ensure that all streams safely flushed
	// before finishing the commit
	for _, stream := range l.streams {
		err := stream.Flush()
		if err != nil {
			return err
		}
	}

	err = l.Error()
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
