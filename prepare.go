package texst

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

type Prepare struct {
	DefaultIGroup rune
}

func (p Prepare) Text(ref io.Writer, subj io.Reader) (err error) {
	if p.DefaultIGroup == 0 {
		p.DefaultIGroup = ' '
	} else if p.DefaultIGroup != ' ' {
		fmt.Fprintf(ref, "%%%%%c\n", p.DefaultIGroup)
	}
	var sep lineSepScanner
	scn := bufio.NewScanner(subj)
	scn.Split(sep.ScanLines)
	prefix := []byte{TagRefLine, byte(p.DefaultIGroup)}
	for scn.Scan() {
		if _, err = ref.Write(prefix); err != nil {
			return err
		}
		if _, err = ref.Write(scn.Bytes()); err != nil {
			return err
		}
		if _, err = ref.Write(sep); err != nil {
			return err
		}
	}
	return nil
}

type lineSepScanner []byte

func (lsc *lineSepScanner) ScanLines(data []byte, atEOF bool) (advance int, token []byte, err error) {
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
