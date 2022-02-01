package main

import (
	"errors"
	"fmt"
)

var (
	ErrInvalidImageBlock = errors.New("invalid image block")
)

// ErrUnsupportedArchiveVersion is an error returned when trying to import an
// archive with a version that this server does not support.
type ErrUnsupportedArchiveVersion struct {
	got  int
	want int
}

// NewErrUnsupportedArchiveVersion creates a ErrUnsupportedArchiveVersion error.
func NewErrUnsupportedArchiveVersion(got int, want int) ErrUnsupportedArchiveVersion {
	return ErrUnsupportedArchiveVersion{
		got:  got,
		want: want,
	}
}

func (e ErrUnsupportedArchiveVersion) Error() string {
	return fmt.Sprintf("unsupported archive version; got %d, want %d", e.got, e.want)
}

// ErrUnsupportedArchiveLineType is an error returned when trying to import an
// archive containing an unsupported line type.
type ErrUnsupportedArchiveLineType struct {
	line int
	got  string
}

// NewErrUnsupportedArchiveLineType creates a ErrUnsupportedArchiveLineType error.
func NewErrUnsupportedArchiveLineType(line int, got string) ErrUnsupportedArchiveLineType {
	return ErrUnsupportedArchiveLineType{
		line: line,
		got:  got,
	}
}

func (e ErrUnsupportedArchiveLineType) Error() string {
	return fmt.Sprintf("unsupported archive line type; got %s, line %d", e.got, e.line)
}
