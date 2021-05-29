package texst

import (
	"bufio"
	"bytes"
	"io"
	"os"
)

type LineSepScanner []byte

func (lsc *LineSepScanner) ScanLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
	// modificated version of bufio.Scan
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.IndexByte(data, '\n'); i >= 0 {
		res, cr := dropCR(data[0:i])
		*lsc = data[i-cr : i+1]
		return i + 1, res, nil
	}
	if atEOF {
		res, cr := dropCR(data)
		*lsc = data[len(data)-cr:]
		return len(data), res, nil
	}
	return 0, nil, nil
}

func dropCR(data []byte) ([]byte, int) {
	// modificated version of bufio.dropCR
	if len(data) > 0 && data[len(data)-1] == '\r' {
		return data[0 : len(data)-1], 1
	}
	return data, 0
}

func Prepare(prepared io.Writer, subj io.Reader) (err error) {
	var sep LineSepScanner
	scn := bufio.NewScanner(subj)
	scn.Split(sep.ScanLines)
	prefix := []byte{TagRefLine, ' '}
	for scn.Scan() {
		if _, err = prepared.Write(prefix); err != nil {
			return err
		}
		if _, err = prepared.Write(scn.Bytes()); err != nil {
			return err
		}
		if _, err = prepared.Write(sep); err != nil {
			return err
		}
	}
	return nil
}

func PrepareFile(prepared string, subj io.Reader) error {
	wr, err := os.Create(prepared)
	if err != nil {
		return err
	}
	return Prepare(wr, subj)
}
