// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

var gTraceParentRe *regexp.Regexp = regexp.MustCompile("([0-9a-f]{2})-([0-9a-f]{32})-([0-9a-f]{16})-([0-9a-f]{2})")

func NewTraceId() string {
	return strings.ReplaceAll(uuid.NewString(), "-", "")
}

func NewSpanId() string {
	return strings.ReplaceAll(uuid.NewString(), "-", "")[:16]
}

func TraceParentIsValid(traceParent string) bool {
	return gTraceParentRe.MatchString(traceParent)
}

func TraceParentParts(traceParent string) []string {
	matches := gTraceParentRe.FindAllStringSubmatch(traceParent, -1)
	if len(matches) != 1 || len(matches[0]) != 5 {
		return make([]string, 0)
	}
	return matches[0][1:]
}

func TraceIdFromTraceParent(traceParent string) string {
	parts := TraceParentParts(traceParent)
	if len(parts) != 4 {
		log.Warn().Msgf("TraceIdFromTraceParent with invalid traceParent: '%s'", traceParent)
		return ""
	}
	return parts[1]
}
