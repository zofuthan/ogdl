// Copyright 2012-2014, Rolf Veen and contributors.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ogdl

import "os"

// Log is a log store for binary OGDL objects.
//
// All objects are appended to a file, and a position is returned.
//
type Log struct {
	f        *os.File
	autoSync bool
}

// OpenLog opens a log file. If the file doesn't exist, it is created.
func OpenLog(file string) (*Log, error) {

	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		return nil, err
	}

	log := Log{f, true}

	return &log, nil
}

// Close closes a log file
func (log *Log) Close() {
	log.f.Close()
}

// Sync commits the changes to disk (the exact behavior is OS dependent).
func (log *Log) Sync() {
	log.f.Sync()
}

// Add adds an OGDL object to the log. The starting position into the log
// is returned.
func (log *Log) Add(g *Graph) int64 {

	b := g.Binary()

	if b == nil {
		return 0
	}

	i, _ := log.f.Seek(0, 2)

	log.f.Write(b)

	if log.autoSync {
		log.f.Sync()
	}

	return i
}

// AddBinary adds an OGDL binary object to the log. The starting position into
// the log is returned.
func (log *Log) AddBinary(b []byte) int64 {

	i, _ := log.f.Seek(0, 2)
	log.f.Write(b)

	if log.autoSync {
		log.f.Sync()
	}

	return i
}

// Get returns the OGDL object at the position given and the position of the
// next object, or an error.
func (log *Log) Get(i int64) (*Graph, error, int64) {

	/* Position in file */
	_, err := log.f.Seek(i, 0)
	if err != nil {
		return nil, err, -1
	}

	p := NewBinParser(log.f)
	g := p.Parse()

    if p.n == 0 {
        return g, nil, -1
    }
    
	return g, err, i + int64(p.n)
}

// GetBinary returns the OGDL object at the position given and the position of the
// next object, or an error. The object returned is in binary form, exactly
// as it is stored in the log.
func (log *Log) GetBinary(i int64) ([]byte, error, int64) {

	// Position in file
	_, err := log.f.Seek(i, 0)
	if err != nil {
		return nil, err, 0
	}

	/* Read until EOS of binary OGDL.

	   There should be a Header first.
	*/
	p := NewBinParser(log.f)

	if !p.header() {
		return nil, err, 0
	}
	for {
		lev, _, _ /* typ, b*/ := p.line(false)
		if lev == 0 {
			break
		}
	}

	n := p.n

	// Read bytes
	b := make([]byte, n)
	_, err = log.f.ReadAt(b, i)

	return b, err, int64(n)
}
