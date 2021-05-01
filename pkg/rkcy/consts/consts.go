// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package consts

const (
	Rkcy            string = "rkcy"
	DirectiveHeader string = "rkcy.directive"
	TraceIdHeader   string = "rkcy.trace_id"
	RkcyTopic       string = "rkcy.topic"
	RkcyPartition   string = "rkcy.partition"

	MaxPartition                int32 = 1024
	PlatformAdminRetentionBytes int32 = 10 * 1024 * 1024
)

type StandardTopicName string

const (
	Admin    StandardTopicName = "admin"
	Process                    = "process"
	Error                      = "error"
	Complete                   = "complete"
	Storage                    = "storage"
)
