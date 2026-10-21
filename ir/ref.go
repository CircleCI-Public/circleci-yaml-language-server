package ir

import (
	"crypto/sha256"
	"encoding/hex"
	"reflect"

	"github.com/fxamacker/cbor/v2"
)

// Ref content-addresses an encoded value as "<version>:sha256:<hex>".
type Ref string

var (
	encMode = mustEncMode()
	decMode = mustDecMode()
)

func mustEncMode() cbor.EncMode {
	m, err := cbor.CoreDetEncOptions().EncMode()
	if err != nil {
		panic(err)
	}
	return m
}

func mustDecMode() cbor.DecMode {
	m, err := cbor.DecOptions{
		DefaultMapType: reflect.TypeFor[map[string]any](),
		IntDec:         cbor.IntDecConvertSignedOrFail,
	}.DecMode()
	if err != nil {
		panic(err)
	}
	return m
}

// Encode is the canonical encoding: core deterministic CBOR (RFC 8949
// section 4.2.1), keyed by the json struct tags.
func Encode(v any) ([]byte, error) {
	return encMode.Marshal(v)
}

// Decode reverses Encode. Untyped maps decode as map[string]any and untyped
// integers as int64.
func Decode(data []byte, v any) error {
	return decMode.Unmarshal(data, v)
}

// RefOf encodes v and returns its ref along with the encoding.
func RefOf(version string, v any) (Ref, []byte, error) {
	b, err := Encode(v)
	if err != nil {
		return "", nil, err
	}
	sum := sha256.Sum256(b)
	return Ref(version + ":sha256:" + hex.EncodeToString(sum[:])), b, nil
}
