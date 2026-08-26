package domain

import "fmt"

type ParseCode string

const (
	CodeUnsupported ParseCode = "unsupported_container"
	CodeCorrupt     ParseCode = "corrupt_container"
	CodeLimit       ParseCode = "resource_limit"
)

type ParseError struct {
	Code   ParseCode
	Offset int64
	Reason string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("%s at byte %d: %s", e.Code, e.Offset, e.Reason)
}
func Corrupt(offset int64, reason string) error {
	return &ParseError{Code: CodeCorrupt, Offset: offset, Reason: reason}
}
func Unsupported(reason string) error { return &ParseError{Code: CodeUnsupported, Reason: reason} }
func Limit(reason string) error       { return &ParseError{Code: CodeLimit, Reason: reason} }
