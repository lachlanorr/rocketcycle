// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"encoding/base64"
	"encoding/binary"
	"fmt"
)

func IsPackedPayload(payload []byte) bool {
	return payload != nil && len(payload) > 4 && string(payload[:4]) == "rkcy"
}

func ParsePayload(payload []byte) ([]byte, []byte, error) {
	var (
		instBytes []byte
		relBytes  []byte
		err       error
	)
	if IsPackedPayload(payload) {
		instBytes, relBytes, err = UnpackPayloads(payload)
		if err != nil {
			return nil, nil, err
		}
	} else {
		instBytes = payload
	}
	return instBytes, relBytes, nil
}

func PackPayloads(payload0 []byte, payload1 []byte) []byte {
	if IsPackedPayload(payload0) || IsPackedPayload(payload1) {
		panic(fmt.Sprintf("PackPayloads of already packed payload payload0=%s payload1=%s", base64.StdEncoding.EncodeToString(payload0), base64.StdEncoding.EncodeToString(payload1)))
	}
	packed := make([]byte, 8+len(payload0)+len(payload1))
	copy(packed, "rkcy")
	offset := 8 + len(payload0)
	binary.LittleEndian.PutUint32(packed[4:8], uint32(offset))
	copy(packed[8:], payload0)
	copy(packed[offset:], payload1)
	return packed
}

func UnpackPayloads(packed []byte) ([]byte, []byte, error) {
	if packed == nil || len(packed) < 8 {
		return nil, nil, fmt.Errorf("Not a packed payload, too small: %s", base64.StdEncoding.EncodeToString(packed))
	}
	if string(packed[:4]) != "rkcy" {
		return nil, nil, fmt.Errorf("Not a packed payload, missing rkcy: %s", base64.StdEncoding.EncodeToString(packed))
	}
	offset := binary.LittleEndian.Uint32(packed[4:8])
	if offset < uint32(8) || offset > uint32(len(packed)) {
		return nil, nil, fmt.Errorf("Not a packed payload, invalid offset %d: %s", offset, base64.StdEncoding.EncodeToString(packed))
	}

	instBytes := packed[8:offset]
	relBytes := packed[offset:]

	// Return nils if there is no data in slices
	if len(instBytes) == 0 {
		return nil, nil, fmt.Errorf("No instance contained in packed payload: %s", base64.StdEncoding.EncodeToString(packed))
	}
	if len(relBytes) == 0 {
		relBytes = nil
	}

	return instBytes, relBytes, nil
}
