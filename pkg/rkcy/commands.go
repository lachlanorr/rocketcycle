// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

const (
	CREATE       = "Create"
	READ         = "Read"
	UPDATE       = "Update"
	UPDATE_ASYNC = "UpdateAsync"
	DELETE       = "Delete"

	VALIDATE_CREATE = "ValidateCreate"
	VALIDATE_UPDATE = "ValidateUpdate"

	REFRESH_INSTANCE = "RefreshInstance"
	FLUSH_INSTANCE   = "FlushInstance"

	REQUEST_RELATED = "RequestRelated"
	REFRESH_RELATED = "RefreshRelated"
)

var gReservedCommands map[string]bool
var gTxnProhibitedCommands map[string]bool
var gSecondaryStorageCommands map[string]bool

func IsReservedCommand(cmd string) bool {
	if gReservedCommands == nil {
		gReservedCommands = make(map[string]bool)
		gReservedCommands[CREATE] = true
		gReservedCommands[READ] = true
		gReservedCommands[UPDATE] = true
		gReservedCommands[UPDATE_ASYNC] = true
		gReservedCommands[DELETE] = true

		gReservedCommands[VALIDATE_CREATE] = true
		gReservedCommands[VALIDATE_UPDATE] = true

		gReservedCommands[REFRESH_INSTANCE] = true
		gReservedCommands[FLUSH_INSTANCE] = true

		gReservedCommands[REQUEST_RELATED] = true
		gReservedCommands[REFRESH_RELATED] = true
	}
	if gReservedCommands[cmd] {
		return true
	}
	return false
}

func IsPlatformCommand(cmd string) bool {
	return !IsReservedCommand(cmd)
}

func IsTxnProhibitedCommand(cmd string) bool {
	if gTxnProhibitedCommands == nil {
		gTxnProhibitedCommands = make(map[string]bool)

		gTxnProhibitedCommands[VALIDATE_CREATE] = true
		gTxnProhibitedCommands[VALIDATE_UPDATE] = true

		gTxnProhibitedCommands[REFRESH_INSTANCE] = true
		gTxnProhibitedCommands[FLUSH_INSTANCE] = true

		gTxnProhibitedCommands[REQUEST_RELATED] = true
		gTxnProhibitedCommands[REFRESH_RELATED] = true
	}
	if gTxnProhibitedCommands[cmd] {
		return true
	}
	return false
}
