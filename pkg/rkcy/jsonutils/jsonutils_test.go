// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package jsonutils

import (
	"fmt"
	"testing"
	//	"google.golang.org/protobuf/proto"
	//"google.golang.org/protobuf/encoding/protojson"
)

func TestUnmarshalOrdered(t *testing.T) {
	{
		b := []byte("true")
		val := false
		err := UnmarshalOrdered(b, &val)
		if err != nil {
			t.Errorf(err.Error())
		}
		if val != true {
			t.Errorf("Wrong value parsed")
		}
	}
	{
		b := []byte("false")
		val := true
		err := UnmarshalOrdered(b, &val)
		if err != nil {
			t.Errorf(err.Error())
		}
		if val != false {
			t.Errorf("Wrong value parsed")
		}
	}
	{
		b := []byte("-100.34e-5")
		val := 123.0
		err := UnmarshalOrdered(b, &val)
		if err != nil {
			t.Errorf(err.Error())
		}
		if val != -100.34e-5 {
			t.Errorf("Wrong value parsed")
		}
	}
	{
		b := []byte("\"a string\"")
		val := ""
		err := UnmarshalOrdered(b, &val)
		if err != nil {
			t.Errorf(err.Error())
		}
		if val != "a string" {
			t.Errorf("Wrong value parsed")
		}
	}
	{
		b := []byte("[1, 2, 3, \"abc\"]")
		var val []interface{}
		err := UnmarshalOrdered(b, &val)
		if err != nil {
			t.Errorf(err.Error())
		}
		if val == nil {
			t.Errorf("Nil returned")
		}
		if len(val) != 4 {
			t.Errorf("Wrong array size")
		}

		itm0, ok := val[0].(float64)
		if !ok || itm0 != 1.0 {
			t.Errorf("Wrong value in array 0")
		}
		itm1, ok := val[1].(float64)
		if !ok || itm1 != 2.0 {
			t.Errorf("Wrong value in array 1")
		}
		itm2, ok := val[2].(float64)
		if !ok || itm2 != 3.0 {
			t.Errorf("Wrong value in array 2")
		}
		itm3, ok := val[3].(string)
		if !ok || itm3 != "abc" {
			t.Errorf("Wrong value in array 3")
		}
	}
	{
		b := []byte(`  {"zkey0": "val0", "wkey1": 123.345   , "key2": [1, 2, 3], "key3": true, "key4": null
, "0key5"   : {"subkey0": true, "subkey2": -300.82E10}}   `)
		var val *OrderedMap
		err := UnmarshalOrdered(b, &val)
		if err != nil {
			t.Errorf(err.Error())
		}
		if val == nil {
			t.Errorf("Nil returned")
		}
		if len(val.Keys) != 6 || len(val.KeyVals) != 6 {
			t.Errorf("Wrong key size: Keys: %d, KeyVals: %d", len(val.Keys), len(val.KeyVals))
		}
		if val.Keys[0] != "zkey0" || val.Keys[1] != "wkey1" || val.Keys[2] != "key2" || val.Keys[3] != "key3" ||
			val.Keys[4] != "key4" || val.Keys[5] != "0key5" {
			t.Errorf("Out of order for keys")
		}

		{
			mapi, _ := val.Get("zkey0")
			mapv, ok := mapi.(string)
			if !ok || mapv != "val0" {
				t.Errorf("Bad value")
			}
		}
		{
			mapi, _ := val.Get("wkey1")
			mapv, ok := mapi.(float64)
			if !ok || mapv != 123.345 {
				t.Errorf("Bad value")
			}
		}
		{
			mapi, _ := val.Get("key2")
			mapv, ok := mapi.([]interface{})
			if !ok {
				t.Errorf("Bad value")
			}
			if len(mapv) != 3 {
				t.Errorf("Bad array length")
			}
			itm1, ok := mapv[0].(float64)
			if !ok || itm1 != 1.0 {
				t.Errorf("Wrong value in array 1")
			}
			itm2, ok := mapv[1].(float64)
			if !ok || itm2 != 2.0 {
				t.Errorf("Wrong value in array 2")
			}
			itm3, ok := mapv[2].(float64)
			if !ok || itm3 != 3.0 {
				t.Errorf("Wrong value in array 3")
			}
		}
		{
			mapi, _ := val.Get("key3")
			mapv, ok := mapi.(bool)
			if !ok || mapv != true {
				t.Errorf("Bad value")
			}
		}
		{
			mapi, _ := val.Get("key4")
			if mapi != nil {
				t.Errorf("Bad value")
			}
		}
		{
			mapi, _ := val.Get("0key5")
			mapv, ok := mapi.(*OrderedMap)
			if !ok {
				t.Errorf("Bad value")
			}
			if len(mapv.Keys) != 2 || len(mapv.KeyVals) != 2 {
				t.Errorf("Bad array length")
			}

			{
				smapi, ok := mapv.Get("subkey0")
				if !ok {
					t.Errorf("missing subkey")
				}
				smapv, ok := smapi.(bool)
				if !ok || smapv != true {
					t.Errorf("Bad value")
				}
			}
			{
				smapi, ok := mapv.Get("subkey2")
				if !ok {
					t.Errorf("missing subkey")
				}
				smapv, ok := smapi.(float64)
				if !ok || smapv != -300.82e10 {
					t.Errorf("Bad value")
				}
			}
		}
	}
}

func TestParseJsonToken(t *testing.T) {
	{
		mixed := []byte("\n \t{\"key0\": \"val0\"\t,\"key2\":123.45,\n\n\t   \"key3\":[1, 2, 3,563.23,true, false, null, \"string\\\"val\\\"\"] \t\t\n  }   \t")
		toks := []*token{
			&token{Type: ObjectStart},
			&token{Type: String, Val: "key0"},
			&token{Type: Colon},
			&token{Type: String, Val: "val0"},
			&token{Type: Comma},
			&token{Type: String, Val: "key2"},
			&token{Type: Colon},
			&token{Type: Number, Val: 123.45},
			&token{Type: Comma},
			&token{Type: String, Val: "key3"},
			&token{Type: Colon},
			&token{Type: ArrayStart},
			&token{Type: Number, Val: 1.0},
			&token{Type: Comma},
			&token{Type: Number, Val: 2.0},
			&token{Type: Comma},
			&token{Type: Number, Val: 3.0},
			&token{Type: Comma},
			&token{Type: Number, Val: 563.23},
			&token{Type: Comma},
			&token{Type: Bool, Val: true},
			&token{Type: Comma},
			&token{Type: Bool, Val: false},
			&token{Type: Comma},
			&token{Type: Null},
			&token{Type: Comma},
			&token{Type: String, Val: "string\"val\""},
			&token{Type: ArrayEnd},
			&token{Type: ObjectEnd},
		}
		b := mixed
		var tok *token
		var err error

		for _, testTok := range toks {
			b, tok, err = nextToken(b)
			if err != nil {
				t.Fatalf("nextToken error: '%s'... %s", err.Error(), b)
			}
			if tok == nil {
				t.Fatalf("nextToken nil tok... %s", b)
			}
			if tok.Type != testTok.Type {
				t.Fatalf("nextToken wrong token type: %d... %s", tok.Type, b)
			}
			if tok.Val != testTok.Val {
				t.Fatalf("nextToken wrong value: %+v vs %+v... %s", tok.Val, testTok.Val, b)
			}
		}
	}

	// String
	{
		testVals := map[string]string{
			`a string`: "a string",

			`quote at end \"`:      "quote at end \"",
			`\"quote at beginning`: "\"quote at beginning",
			`quote \" in middle`:   "quote \" in middle",

			`backslash at end \\`:      "backslash at end \\",
			`\\backslash at beginning`: "\\backslash at beginning",
			`backslash \\ in middle`:   "backslash \\ in middle",

			`slash at end \/`:      "slash at end /",
			`\/slash at beginning`: "/slash at beginning",
			`slash \/ in middle`:   "slash / in middle",

			`backspace at end \b`:      "backspace at end \b",
			`\bbackspace at beginning`: "\bbackspace at beginning",
			`backspace \b in middle`:   "backspace \b in middle",

			`formfeed at end \f`:      "formfeed at end \f",
			`\fformfeed at beginning`: "\fformfeed at beginning",
			`formfeed \f in middle`:   "formfeed \f in middle",

			`newline at end \n`:      "newline at end \n",
			`\nnewline at beginning`: "\nnewline at beginning",
			`newline \n in middle`:   "newline \n in middle",

			`return at end \r`:      "return at end \r",
			`\rreturn at beginning`: "\rreturn at beginning",
			`return \r in middle`:   "return \r in middle",

			`tab at end \t`:      "tab at end \t",
			`\ttab at beginning`: "\ttab at beginning",
			`tab \t in middle`:   "tab \t in middle",
		}

		for tv, exp := range testVals {
			qtv := fmt.Sprintf(`"%s"`, tv)
			b, tok, err := nextToken([]byte(qtv))
			if err != nil {
				t.Errorf("String decode error %s, %s", err.Error(), tv)
			}
			if len(b) != 0 {
				t.Errorf("String decode returned non empty slice: %s", tv)
			}
			if tok.Val.(string) != exp {
				t.Errorf("String decode match failure: %s != %s", tok.Val.(string), exp)
			}
		}
	}

	// Number
	{
		testVals := map[string]float64{
			"123":       123.0,
			"123.34e10": 123.34e10,
			"-0.534E-2": -0.534e-2,
		}

		for tv, exp := range testVals {
			b, tok, err := nextToken([]byte(tv))
			if err != nil {
				t.Errorf("Number decode error %s, %s", err.Error(), tv)
			}
			if len(b) != 0 {
				t.Errorf("Number decode returned non empty slice: %s", tv)
			}
			if tok.Val.(float64) != exp {
				t.Errorf("Number decode match failure: %f != %f", tok.Val.(float64), exp)
			}
		}
	}
}
