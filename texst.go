package texst

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
)

// Line Tags
const (
	// Marks a comment line.
	TagComment = '#'

	// Preamble line regarding interleaving groups.
	TagIGroup = '%'

	// Global argument line
	TagGlobalArg = '*'

	// Reference lines have the text that is compared to the subject text.
	TagRefLine = '>'

	// Argument lines apply to the most recent '>' reference line up to the next
	// non-argument line.
	TagRefLineArg = ' '
)

type RefDoc interface {
	Name() string
	Line() int
	IGroups() []rune
	NextLine() (*RefLine, error)
	FreeLine(*RefLine)
}

func lineError(ref RefDoc, err error) error {
	return fmt.Errorf("%s:%d:%w", ref.Name(), ref.Line(), err)

}

func lineErrorf(ref RefDoc, form string, args ...any) error {
	var sb strings.Builder
	fmt.Fprintf(&sb, "%s:%d:", ref.Name(), ref.Line())
	fmt.Fprintf(&sb, form, args...)
	return errors.New(sb.String())
}

type MismatchFunc func(testedNo int, testedLine []byte, ref []*RefLine)
type MatchFunc func(testedNo int, testedLine []byte, ref *RefLine, match []int)

type Texst struct {
	MismatchLimit int
	OnMismatch    MismatchFunc
	OnMatch       MatchFunc
}

func (txs *Texst) mismatch(lno int, line []byte, ref []*RefLine) {
	if txs.OnMismatch != nil {
		txs.OnMismatch(lno, line, ref)
	}
}

func (txs *Texst) match(lno int, line []byte, ref *RefLine, match []int) {
	if txs.OnMatch != nil {
		txs.OnMatch(lno, line, ref, match)
	}
}

func (txs *Texst) Check(reference RefDoc, subject io.Reader) (mismatchCount int, err error) {
	igBacklog := make([]refLineQ, len(reference.IGroups()))
	subjScan := bufio.NewScanner(subject)
	subjLine := 0
	var mismatch []*RefLine
	for subjScan.Scan() {
		subjLine++
		if err = fillIGBacklog(reference, igBacklog); errors.Is(err, io.EOF) {
			txs.mismatch(subjLine, subjScan.Bytes(), nil)
			return mismatchCount + 1, nil
		} else if err != nil {
			return mismatchCount, err
		}
		var (
			matchLine  *RefLine
			regexMatch []int
		)
		clear(mismatch)
		mismatch = mismatch[:0]
	IGOUP_LOOP:
		for ig := range igBacklog {
			igbl := &igBacklog[ig]
			if igbl.empty() {
				continue IGOUP_LOOP
			}
			refLine := igbl.first
			regexMatch = refLine.match(subjScan.Bytes())
			if regexMatch == nil {
				mismatch = append(mismatch, refLine)
				continue IGOUP_LOOP
			}
			fail := false
			for i, seg := range refLine.masks {
				if len(seg.checks) == 0 {
					continue
				}
				segTxt := subjScan.Bytes()
				segTxt = segTxt[regexMatch[2*i]:regexMatch[2*i+1]]
				for _, check := range seg.checks {
					if check.Check(segTxt) != nil {
						fail = true
						break
					}
				}

			}
			if fail {
				mismatch = append(mismatch, refLine)
			} else {
				igbl.dropFirst()
				matchLine = refLine
				break IGOUP_LOOP
			}
		}
		if matchLine == nil {
			txs.mismatch(subjLine, subjScan.Bytes(), mismatch)
			mismatchCount++
			if txs.MismatchLimit > 0 && mismatchCount >= txs.MismatchLimit {
				break
			}
		} else {
			txs.match(subjLine, subjScan.Bytes(), matchLine, regexMatch)
			reference.FreeLine(matchLine)
		}
	}
	if err = fillIGBacklog(reference, igBacklog); err != nil && !errors.Is(err, io.EOF) {
		return mismatchCount, err
	}
	clear(mismatch)
	mismatch = mismatch[:0]
	for _, ig := range igBacklog {
		if !ig.empty() {
			mismatch = append(mismatch, ig.first)
		}
	}
	if len(mismatch) > 0 {
		txs.mismatch(subjLine+1, nil, mismatch)
		mismatchCount++
	}
	return mismatchCount, nil
}

func fillIGBacklog(ref RefDoc, igbl []refLineQ) error {
	empty := 0
	for i := range igbl {
		if igbl[i].empty() {
			empty++
		}
	}
	for empty > 0 {
		refLine, err := ref.NextLine()
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return err
			}
			if refLine == nil {
				if empty == len(igbl) {
					return io.EOF
				}
				return nil
			}
		}
		igIdx := slices.Index(ref.IGroups(), refLine.igName)
		if igIdx < 0 {
			return lineErrorf(ref, "unknown interleaving group: %c", refLine.igName)
		}
		if igbl[igIdx].empty() {
			empty--
		}
		igbl[igIdx].pushBack(refLine)
	}
	return nil
}

type refLineQ struct{ first, last *RefLine }

func (b *refLineQ) empty() bool { return b.first == nil }

func (b *refLineQ) dropFirst() {
	b.first = b.first.lsNext
	if b.first == nil {
		b.last = nil
	}
}

func (b *refLineQ) pushBack(l *RefLine) {
	if b.first == nil {
		b.first, b.last = l, l
	} else {
		b.last.lsNext = l
		b.last = l
	}
}
