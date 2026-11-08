// Package threat содержит бизнес-модели сведений об угрозах.
package threat

import (
	"encoding/hex"
	"errors"
	"strings"
)

var (
	// ErrNotFound означает отсутствие записи с указанным MD5.
	ErrNotFound = errors.New("threat not found")

	// ErrAlreadyExists означает, что запись с указанным MD5 уже существует.
	ErrAlreadyExists = errors.New("threat already exists")

	// ErrInvalidMD5Length означает, что длина MD5 отличается от 32 символов.
	ErrInvalidMD5Length = errors.New("MD5 must contain exactly 32 hexadecimal characters")

	// ErrInvalidMD5Characters означает наличие недопустимых hex-символов.
	ErrInvalidMD5Characters = errors.New("MD5 contains invalid hexadecimal characters")
)

// Threat — сведения об угрозе с MD5 файла в качестве ключа.
type Threat struct {
	MD5          MD5
	Name         string
	Description  string
	MalwareTypes []MalwareType
	Confidence   *int
}

// MD5 — хеш файла: 32 шестнадцатеричных символа в нижнем регистре.
//
// Используйте ParseMD5: прямое приведение типа не проверяет значение.
type MD5 string

// MalwareType — категория из поддерживаемого поднабора словаря STIX.
type MalwareType string

const (
	// MalwareTypeBackdoor — бэкдор.
	MalwareTypeBackdoor MalwareType = "backdoor"

	// MalwareTypeRansomware — программа-вымогатель.
	MalwareTypeRansomware MalwareType = "ransomware"

	// MalwareTypeSpyware — шпионская программа.
	MalwareTypeSpyware MalwareType = "spyware"

	// MalwareTypeTrojan — троянская программа.
	MalwareTypeTrojan MalwareType = "trojan"

	// MalwareTypeVirus — вирус.
	MalwareTypeVirus MalwareType = "virus"

	// MalwareTypeWorm — червь.
	MalwareTypeWorm MalwareType = "worm"
)

// ParseMD5 проверяет хеш и возвращает его в нижнем регистре.
// Принимает ровно 32 ASCII hex-символа без пробелов и префиксов.
func ParseMD5(value string) (MD5, error) {
	if len(value) != 32 {
		return "", ErrInvalidMD5Length
	}

	if _, err := hex.DecodeString(value); err != nil {
		return "", ErrInvalidMD5Characters
	}

	return MD5(strings.ToLower(value)), nil
}
