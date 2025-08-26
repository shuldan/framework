package database

import (
	"github.com/google/uuid"
	"github.com/shuldan/framework/pkg/contracts"
	"strconv"
)

type UUID struct {
	value uuid.UUID
}

var _ contracts.ID = UUID{}

func NewUUID() UUID {
	return UUID{value: uuid.New()}
}

func ParseUUID(s string) (UUID, error) {
	val, err := uuid.Parse(s)
	return UUID{value: val}, err
}

func MustParseUUID(s string) UUID {
	val, err := uuid.Parse(s)
	if err != nil {
		panic(err)
	}
	return UUID{value: val}
}

func (u UUID) String() string {
	return u.value.String()
}

func (u UUID) IsValid() bool {
	return u.value != uuid.Nil
}

func (u UUID) UUID() uuid.UUID {
	return u.value
}

type IntID struct {
	value int64
}

var _ contracts.ID = IntID{}

func NewIntID(v int64) IntID {
	return IntID{value: v}
}

func ParseIntID(s string) (IntID, error) {
	val, err := strconv.ParseInt(s, 10, 64)
	return IntID{value: val}, err
}

func MustParseIntID(s string) IntID {
	val, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		panic(err)
	}
	return IntID{value: val}
}

func (i IntID) String() string {
	return strconv.FormatInt(i.value, 10)
}

func (i IntID) IsValid() bool {
	return i.value > 0
}

func (i IntID) Int64() int64 {
	return i.value
}

type StringID struct {
	value string
}

var _ contracts.ID = StringID{}

func NewStringID(v string) StringID {
	return StringID{value: v}
}

func (s StringID) String() string {
	return s.value
}

func (s StringID) IsValid() bool {
	return s.value != ""
}
