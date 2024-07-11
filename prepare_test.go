package texst

import (
	"bufio"
	"strings"
	"testing"
)

func TestLineSepScanner_crnl(t *testing.T) {
	t.Run("between lines", func(t *testing.T) {
		rd := strings.NewReader("line1\r\nline2")
		scn := bufio.NewScanner(rd)
		var sep lineSepScanner
		scn.Split(sep.ScanLines)
		line := 0
		for scn.Scan() {
			line++
			switch line {
			case 1:
				if txt := scn.Text(); txt != "line1" {
					t.Errorf("line %d: wrong text '%s'", line, txt)
				}
				if string(sep) != "\r\n" {
					t.Errorf("line %d: separator '%v'", line, sep)
				}
			case 2:
				if txt := scn.Text(); txt != "line2" {
					t.Errorf("line %d: wrong text '%s'", line, txt)
				}
				if string(sep) != "" {
					t.Errorf("line %d: separator '%v'", line, sep)
				}
			}
		}
		if line != 2 {
			t.Errorf("wrong number of lines: %d", line)
		}
	})
	t.Run("last line", func(t *testing.T) {
		rd := strings.NewReader("line1\r\n")
		scn := bufio.NewScanner(rd)
		var sep lineSepScanner
		scn.Split(sep.ScanLines)
		line := 0
		for scn.Scan() {
			line++
			switch line {
			case 1:
				if txt := scn.Text(); txt != "line1" {
					t.Errorf("line %d: wrong text '%s'", line, txt)
				}
				if string(sep) != "\r\n" {
					t.Errorf("line %d: separator '%v'", line, sep)
				}
			}
		}
		if line != 1 {
			t.Errorf("wrong number of lines: %d", line)
		}
	})
}

func TestLineSepScanner_nl(t *testing.T) {
	t.Run("between lines", func(t *testing.T) {
		rd := strings.NewReader("line1\nline2")
		scn := bufio.NewScanner(rd)
		var sep lineSepScanner
		scn.Split(sep.ScanLines)
		line := 0
		for scn.Scan() {
			line++
			switch line {
			case 1:
				if txt := scn.Text(); txt != "line1" {
					t.Errorf("line %d: wrong text '%s'", line, txt)
				}
				if string(sep) != "\n" {
					t.Errorf("line %d: separator '%v'", line, sep)
				}
			case 2:
				if txt := scn.Text(); txt != "line2" {
					t.Errorf("line %d: wrong text '%s'", line, txt)
				}
				if string(sep) != "" {
					t.Errorf("line %d: separator '%v'", line, sep)
				}
			}
		}
		if line != 2 {
			t.Errorf("wrong number of lines: %d", line)
		}
	})
	t.Run("last line", func(t *testing.T) {
		rd := strings.NewReader("line1\n")
		scn := bufio.NewScanner(rd)
		var sep lineSepScanner
		scn.Split(sep.ScanLines)
		line := 0
		for scn.Scan() {
			line++
			switch line {
			case 1:
				if txt := scn.Text(); txt != "line1" {
					t.Errorf("line %d: wrong text '%s'", line, txt)
				}
				if string(sep) != "\n" {
					t.Errorf("line %d: separator '%v'", line, sep)
				}
			}
		}
		if line != 1 {
			t.Errorf("wrong number of lines: %d", line)
		}
	})
}
