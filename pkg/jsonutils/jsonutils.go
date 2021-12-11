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

func (m *OrderedMap) SetAfter(k string, v interface{}, kbefore string) {
	_, ok := m.KeyVals[k]
	if ok {
		m.KeyVals[k] = v
		return
	}

	// look for our kbefore
	idx := -1
	for i, kc := range m.Keys {
		if kc == kbefore {
			idx = i + 1
			break
		}
	}

	// check if not found or at end
	if idx == -1 || idx > len(m.Keys)-1 {
		m.Keys = append(m.Keys, k)
	} else {
		// in the middle
		m.Keys = append(m.Keys[:idx+1], m.Keys[idx:]...)
		m.Keys[idx] = k
	}
	m.KeyVals[k] = v
}

func NewOrderedMap() *OrderedMap {
	return &OrderedMap{
		Keys:    make([]string, 0),
		KeyVals: make(map[string]interface{}),
	}
}

func isTrivialObject(obj *OrderedMap) bool {
	if len(obj.Keys) == 0 {
		return true
	}
	if len(obj.Keys) == 1 {
		v, _ := obj.Get(obj.Keys[0])
		switch val := v.(type) {
		case []interface{}:
			return isTrivialArray(val)
		case *OrderedMap:
			return isTrivialObject(val)
		default:
			return true
		}
	}
	return false
}

func isTrivialArray(arr []interface{}) bool {
	if len(arr) == 0 {
		return true
	}
	if len(arr) == 1 {
		switch val := arr[0].(type) {
		case []interface{}:
			return isTrivialArray(val)
		case *OrderedMap:
			return isTrivialObject(val)
		default:
			return true
		}
	}
	return false
}

func EncodeString(bld *strings.Builder, s string) string {
	if bld == nil {
		bld = &strings.Builder{}
	}
	bld.WriteByte('"')
	for _, c := range s {
		switch c {
		case '"':
			bld.WriteString(`\"`)
		case '\\':
			bld.WriteString(`\\`)
		case '\b':
			bld.WriteString(`\b`)
		case '\f':
			bld.WriteString(`\f`)
		case '\n':
			bld.WriteString(`\n`)
		case '\r':
			bld.WriteString(`\r`)
		case '\t':
			bld.WriteString(`\t`)
		default:
			bld.WriteRune(c)
		}
	}
	bld.WriteByte('"')
	return bld.String()
}

func marshal(v interface{}, bld *strings.Builder, eol string, indent string, currIndent string, colonSpace string) error {
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
		EncodeString(bld, val)
	case []interface{}:
		newIndent := currIndent + indent
		if isTrivialArray(val) {
			eol = ""
			currIndent = ""
			newIndent = ""
		}
		bld.WriteByte('[')
		bld.WriteString(eol)
		for idx, itm := range val {
			bld.WriteString(newIndent)
			err := marshal(itm, bld, eol, indent, newIndent, colonSpace)
			if err != nil {
				return err
			}
			if idx != len(val)-1 {
				bld.WriteByte(',')
			}
			bld.WriteString(eol)
		}
		bld.WriteString(currIndent)
		bld.WriteByte(']')
	case *OrderedMap:
		newIndent := currIndent + indent
		if isTrivialObject(val) {
			eol = ""
			currIndent = ""
			newIndent = ""
		}
		bld.WriteByte('{')
		bld.WriteString(eol)
		for idx, key := range val.Keys {
			bld.WriteString(newIndent)
			bld.WriteByte('"')
			bld.WriteString(key)
			bld.WriteByte('"')
			bld.WriteByte(':')
			bld.WriteString(colonSpace)
			itmVal, ok := val.Get(key)
			if !ok {
				return fmt.Errorf("Missing key in OrderedMap: %s", key)
			}
			err := marshal(itmVal, bld, eol, indent, newIndent, colonSpace)
			if err != nil {
				return err
			}
			if idx != len(val.Keys)-1 {
				bld.WriteByte(',')
			}
			bld.WriteString(eol)
		}
		bld.WriteString(currIndent)
		bld.WriteByte('}')
	default:
		return fmt.Errorf("Invalid type to marshal: %T", v)
	}
	return nil
}

func MarshalOrdered(v interface{}) ([]byte, error) {
	var bld strings.Builder
	err := marshal(v, &bld, "", "", "", "")
	if err != nil {
		return nil, err
	}
	return []byte(bld.String()), nil
}

func MarshalOrderedIndent(v interface{}, prefix string, indent string) ([]byte, error) {
	var bld strings.Builder
	eol := "\n" + prefix
	err := marshal(v, &bld, eol, indent, "", " ")
	if err != nil {
		return nil, err
	}
	mar := bld.String()
	return []byte(mar), nil
}

type tokenType int

const (
	UNKNOWN      tokenType = 0
	STRING                 = 1
	NUMBER                 = 2
	BOOL                   = 3
	NULL                   = 4
	OBJECT_START           = 5
	OBJECT_END             = 6
	ARRAY_START            = 7
	ARRAY_END              = 8
	COMMA                  = 9
	COLON                  = 10
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

		// peek to see if there's a }
		_, tok, err := nextToken(b)
		if err != nil {
			return b, nil, err
		}

		// if not at end, parse value
		if tok.Type != OBJECT_END {

			// parse key
			var tokKey *token
			b, tokKey, err = nextToken(b)
			if err != nil {
				return b, nil, err
			}
			if tokKey == nil || tokKey.Type != STRING || len(tokKey.Val.(string)) == 0 {
				return b, nil, fmt.Errorf("Failed to parse key: %s", string(b))
			}

			// colon
			var tokColon *token
			b, tokColon, err = nextToken(b)
			if err != nil {
				return b, nil, err
			}
			if tokColon == nil || tokColon.Type != COLON {
				return b, nil, fmt.Errorf("Failed to parse colon: %s", string(b))
			}

			// parse value
			var val interface{}
			b, val, err = parse(b)
			if err != nil {
				return b, nil, err
			}
			if err != nil {
				return b, nil, fmt.Errorf("Error parsing value '%s': %s", err.Error(), string(b))
			}

			obj.Set(tokKey.Val.(string), val)
		}

		// check for end or next
		var tokCommaOrEnd *token
		b, tokCommaOrEnd, err = nextToken(b)
		if err != nil {
			return b, nil, err
		}
		if tokCommaOrEnd == nil || (tokCommaOrEnd.Type != COMMA && tokCommaOrEnd.Type != OBJECT_END) {
			return b, nil, fmt.Errorf("Failed to parse comma or end: %s", string(b))
		}

		if tokCommaOrEnd.Type == OBJECT_END {
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

		// peek to see if there's a ]
		_, tok, err := nextToken(b)
		if err != nil {
			return b, nil, err
		}

		// if not at end, parse value
		if tok.Type != ARRAY_END {
			b, val, err = parse(b)
			if err != nil {
				return b, nil, err
			}
			arr = append(arr, val)
		}

		// check for end or next
		var tokCommaOrEnd *token
		b, tokCommaOrEnd, err = nextToken(b)
		if err != nil {
			return b, nil, err
		}
		if tokCommaOrEnd == nil || (tokCommaOrEnd.Type != COMMA && tokCommaOrEnd.Type != ARRAY_END) {
			return b, nil, fmt.Errorf("Failed to parse comma or end: %s", string(b))
		}

		if tokCommaOrEnd.Type == ARRAY_END {
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
	case OBJECT_START:
		return parseObject(b)
	case ARRAY_START:
		return parseArray(b)
	case STRING:
		fallthrough
	case NUMBER:
		fallthrough
	case BOOL:
		fallthrough
	case NULL:
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
		if tok.Type != BOOL {
			return fmt.Errorf("Invalid token: %s", string(b))
		}
		*val = tok.Val.(bool)
	case *float64:
		if tok.Type != NUMBER {
			return fmt.Errorf("Invalid token: %s", string(b))
		}
		*val = tok.Val.(float64)
	case *string:
		if tok.Type != STRING {
			return fmt.Errorf("Invalid token: %s", string(b))
		}
		*val = tok.Val.(string)
	case *[]interface{}:
		if tok.Type == NULL {
			*val = nil
		} else if tok.Type != ARRAY_START {
			return fmt.Errorf("Invalid token: %s", string(b))
		}
		var err error
		b, *val, err = parseArray(b)
		if err != nil {
			return err
		}
	case **OrderedMap:
		if tok.Type == NULL {
			*val = nil
		} else if tok.Type != OBJECT_START {
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
		return bSkip, &token{Type: NUMBER, Val: num}, nil
	} else {
		switch b[0] {
		case '{':
			return b[1:], &token{Type: OBJECT_START}, nil
		case '}':
			return b[1:], &token{Type: OBJECT_END}, nil
		case '[':
			return b[1:], &token{Type: ARRAY_START}, nil
		case ']':
			return b[1:], &token{Type: ARRAY_END}, nil
		case ',':
			return b[1:], &token{Type: COMMA}, nil
		case ':':
			return b[1:], &token{Type: COLON}, nil
		case '"':
			var bld strings.Builder
			// read to end
			for i := 1; i < len(b); i++ {
				switch b[i] {
				case '"':
					return b[i+1:], &token{Type: STRING, Val: bld.String()}, nil
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
				return b[4:], &token{Type: BOOL, Val: true}, nil
			} else {
				return b, nil, fmt.Errorf("Unknown token: %s", string(b[:10]))
			}
		case 'f':
			if len(b) >= 5 && "false" == string(b[:5]) {
				return b[5:], &token{Type: BOOL, Val: false}, nil
			} else {
				return b, nil, fmt.Errorf("Unknown token: %s", string(b[:10]))
			}
		case 'n':
			if len(b) >= 4 && "null" == string(b[:4]) {
				return b[4:], &token{Type: NULL, Val: nil}, nil
			} else {
				return b, nil, fmt.Errorf("Unknown token: %s", string(b[:10]))
			}
		}
	}
	return nil, nil, nil
}
