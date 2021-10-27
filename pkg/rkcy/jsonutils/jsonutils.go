// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package jsonutils

import (
	"fmt"
	"strconv"
	"strings"
)

type OrderedMap struct {
	Keys    []string
	KeyVals map[string]interface{}
}

func (m *OrderedMap) Get(k string) (interface{}, bool) {
	v, ok := m.KeyVals[k]
	return v, ok
}

func (m *OrderedMap) Set(k string, v interface{}) {
	_, ok := m.KeyVals[k]
	if !ok {
		m.Keys = append(m.Keys, k)
	}
	m.KeyVals[k] = v
}

func NewOrderedMap() *OrderedMap {
	return &OrderedMap{
		Keys:    make([]string, 0),
		KeyVals: make(map[string]interface{}),
	}
}

func marshal(v interface{}, bld *strings.Builder) error {
	if v == nil {
		bld.WriteString("null")
		return nil
	}

	switch val := v.(type) {
	case bool:
		bld.WriteString(strconv.FormatBool(val))
	case float64:
		bld.WriteString(strconv.FormatFloat(val, 'f', -1, 64))
	case string:
		bld.WriteByte('"')
		bld.WriteString(val)
		bld.WriteByte('"')
	case []interface{}:
		bld.WriteByte('[')
		for idx, itm := range val {
			err := marshal(itm, bld)
			if err != nil {
				return err
			}
			if idx != len(val)-1 {
				bld.WriteByte(',')
			}
		}
		bld.WriteByte(']')
	case *OrderedMap:
		bld.WriteByte('{')
		for idx, key := range val.Keys {
			bld.WriteByte('"')
			bld.WriteString(key)
			bld.WriteByte('"')
			bld.WriteByte(':')
			itmVal, ok := val.Get(key)
			if !ok {
				return fmt.Errorf("Missing key in OrderedMap: %s", key)
			}
			err := marshal(itmVal, bld)
			if err != nil {
				return err
			}
			if idx != len(val.Keys)-1 {
				bld.WriteByte(',')
			}
		}
		bld.WriteByte('}')
	default:
		return fmt.Errorf("Invalid type to marshal")
	}
	return nil
}

func MarshalOrdered(v interface{}) ([]byte, error) {
	var bld strings.Builder
	err := marshal(v, &bld)
	if err != nil {
		return nil, err
	}
	return []byte(bld.String()), nil
}

type tokenType int

const (
	Unknown     tokenType = 0
	String                = 1
	Number                = 2
	Bool                  = 3
	Null                  = 4
	ObjectStart           = 5
	ObjectEnd             = 6
	ArrayStart            = 7
	ArrayEnd              = 8
	Comma                 = 9
	Colon                 = 10
)

type token struct {
	Type tokenType
	Val  interface{}
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\n' || c == '\r' || c == '\t'
}

func skipSpace(b []byte) []byte {
	for i, v := range b {
		if !isSpace(v) {
			return b[i:]
		}
	}
	return b[len(b):]
}

func isNumber(c byte) bool {
	return (c >= byte('0') && c <= byte('9')) || c == '-' || c == '.' || c == 'e' || c == 'E'
}

func skipNumber(b []byte) []byte {
	for i, v := range b {
		if !isNumber(v) {
			return b[i:]
		}
	}
	return b[len(b):]
}

func parseObject(b []byte) ([]byte, *OrderedMap, error) {
	obj := NewOrderedMap()

	for {
		var err error
		// parse key
		var tokKey *token
		b, tokKey, err = nextToken(b)
		if err != nil || tokKey == nil || tokKey.Type != String || len(tokKey.Val.(string)) == 0 {
			return b, nil, fmt.Errorf("Failed to parse key: %s", string(b))
		}

		// colon
		var tokColon *token
		b, tokColon, err = nextToken(b)
		if err != nil || tokColon == nil || tokColon.Type != Colon {
			return b, nil, fmt.Errorf("Failed to parse colon: %s", string(b))
		}

		// parse value
		var val interface{}
		b, val, err = parse(b)
		if err != nil {
			return b, nil, fmt.Errorf("Failed to parse value: %s", string(b))
		}

		obj.Set(tokKey.Val.(string), val)

		// check for end or next
		var tokCommaOrEnd *token
		b, tokCommaOrEnd, err = nextToken(b)
		if err != nil || tokCommaOrEnd == nil || (tokCommaOrEnd.Type != Comma && tokCommaOrEnd.Type != ObjectEnd) {
			return b, nil, fmt.Errorf("Failed to parse comma or end: %s", string(b))
		}

		if tokCommaOrEnd.Type == ObjectEnd {
			break
		}
	}
	return b, obj, nil
}

func parseArray(b []byte) ([]byte, []interface{}, error) {
	arr := make([]interface{}, 0)
	for {
		var val interface{}
		var err error
		b, val, err = parse(b)
		if err != nil {
			return b, nil, err
		}
		arr = append(arr, val)
		// check for end or next
		var tokCommaOrEnd *token
		b, tokCommaOrEnd, err = nextToken(b)
		if err != nil || tokCommaOrEnd == nil || (tokCommaOrEnd.Type != Comma && tokCommaOrEnd.Type != ArrayEnd) {
			return b, nil, fmt.Errorf("Failed to parse comma or end: %s", string(b))
		}

		if tokCommaOrEnd.Type == ArrayEnd {
			break
		}
	}
	return b, arr, nil
}

func parse(b []byte) ([]byte, interface{}, error) {
	b, tok, err := nextToken(b)
	if err != nil {
		return b, nil, err
	}
	if tok == nil {
		return b, nil, fmt.Errorf("No tokens: %s", string(b))
	}
	switch tok.Type {
	case ObjectStart:
		return parseObject(b)
	case ArrayStart:
		return parseArray(b)
	case String:
		fallthrough
	case Number:
		fallthrough
	case Bool:
		fallthrough
	case Null:
		return b, tok.Val, nil
	default:
		return b, nil, fmt.Errorf("Invalid token %d: %s", tok.Type, string(b))
	}
}

func UnmarshalOrdered(b []byte, v interface{}) error {
	if b == nil || len(b) == 0 {
		return fmt.Errorf("No data given")
	}
	b, tok, err := nextToken(b)
	if err != nil {
		return err
	}
	if tok == nil {
		return fmt.Errorf("No tokens: %s", string(b))
	}
	switch val := v.(type) {
	case *bool:
		if tok.Type != Bool {
			return fmt.Errorf("Invalid token: %s", string(b))
		}
		*val = tok.Val.(bool)
	case *float64:
		if tok.Type != Number {
			return fmt.Errorf("Invalid token: %s", string(b))
		}
		*val = tok.Val.(float64)
	case *string:
		if tok.Type != String {
			return fmt.Errorf("Invalid token: %s", string(b))
		}
		*val = tok.Val.(string)
	case *[]interface{}:
		if tok.Type == Null {
			*val = nil
		} else if tok.Type != ArrayStart {
			return fmt.Errorf("Invalid token: %s", string(b))
		}
		var err error
		b, *val, err = parseArray(b)
		if err != nil {
			return err
		}
	case **OrderedMap:
		if tok.Type == Null {
			*val = nil
		} else if tok.Type != ObjectStart {
			return fmt.Errorf("Invalid token: %s", string(b))
		}
		var err error
		b, *val, err = parseObject(b)
		if err != nil {
			return err
		}
	default:
		return fmt.Errorf("Invalid type to Unmarshal into: %T", v)
	}
	return nil
}

func nextToken(b []byte) ([]byte, *token, error) {
	b = skipSpace(b)
	if isNumber(b[0]) {
		bSkip := skipNumber(b)
		numStr := string(b[:len(b)-len(bSkip)])
		num, err := strconv.ParseFloat(numStr, 64)
		if err != nil {
			return b, nil, fmt.Errorf("Invalid number: %s", numStr)
		}
		return bSkip, &token{Type: Number, Val: num}, nil
	} else {
		switch b[0] {
		case '{':
			return b[1:], &token{Type: ObjectStart}, nil
		case '}':
			return b[1:], &token{Type: ObjectEnd}, nil
		case '[':
			return b[1:], &token{Type: ArrayStart}, nil
		case ']':
			return b[1:], &token{Type: ArrayEnd}, nil
		case ',':
			return b[1:], &token{Type: Comma}, nil
		case ':':
			return b[1:], &token{Type: Colon}, nil
		case '"':
			var bld strings.Builder
			// read to end
			for i := 1; i < len(b); i++ {
				switch b[i] {
				case '"':
					return b[i+1:], &token{Type: String, Val: bld.String()}, nil
				case '\\':
					if i == len(b)-1 {
						return b[i:], nil, fmt.Errorf("'\\' as last character in buffer")
					}
					switch b[i+1] {
					case '"':
						bld.WriteByte('"')
						i++
					case '\\':
						bld.WriteByte('\\')
						i++
					case '/':
						bld.WriteByte('/')
						i++
					case 'b':
						bld.WriteByte('\b')
						i++
					case 'f':
						bld.WriteByte('\f')
						i++
					case 'n':
						bld.WriteByte('\n')
						i++
					case 'r':
						bld.WriteByte('\r')
						i++
					case 't':
						bld.WriteByte('\t')
						i++
					case 'u':
						return b[i:], nil, fmt.Errorf("\\u unicode parsing not supported")
					}
				default:
					bld.WriteByte(b[i])
				}
			}
		case 't':
			if len(b) >= 4 && "true" == string(b[:4]) {
				return b[4:], &token{Type: Bool, Val: true}, nil
			} else {
				return b, nil, fmt.Errorf("Unknown token: %s", string(b[:10]))
			}
		case 'f':
			if len(b) >= 5 && "false" == string(b[:5]) {
				return b[5:], &token{Type: Bool, Val: false}, nil
			} else {
				return b, nil, fmt.Errorf("Unknown token: %s", string(b[:10]))
			}
		case 'n':
			if len(b) >= 4 && "null" == string(b[:4]) {
				return b[4:], &token{Type: Null, Val: nil}, nil
			} else {
				return b, nil, fmt.Errorf("Unknown token: %s", string(b[:10]))
			}
		}
	}
	return nil, nil, nil
}
