// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"

	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
)

type ClientCode struct {
	StorageInits        map[string]StorageInit
	ConcernHandlers     ConcernHandlers
	CustomCobraCommands []*cobra.Command
}

func NewClientCode() *ClientCode {
	return &ClientCode{
		StorageInits:    make(map[string]StorageInit),
		ConcernHandlers: GlobalConcernHandlerRegistry(),
	}
}

func (clientCode *ClientCode) AddStorageInit(storageType string, storageInit StorageInit) {
	if clientCode.StorageInits[storageType] != nil {
		log.Fatal().
			Msgf("Multiple AddStorageInit for storageType %s", storageType)
	}
	clientCode.StorageInits[storageType] = storageInit
}

func (clientCode *ClientCode) AddLogicHandler(concern string, handler interface{}) {
	clientCode.ConcernHandlers.RegisterLogicHandler(concern, handler)
}

func (clientCode *ClientCode) AddCrudHandler(concern string, storageType string, handler interface{}) {
	clientCode.ConcernHandlers.RegisterCrudHandler(concern, storageType, handler)
}

func (clientCode *ClientCode) UpdateStorageTargets(storageTargets map[string]*rkcypb.StorageTarget) {
	stgTgtInits := make(map[string]*StorageTargetInit)
	for _, platStgTgt := range storageTargets {
		tgt := &StorageTargetInit{
			StorageTarget: platStgTgt,
		}
		if tgt.Config == nil {
			tgt.Config = make(map[string]string)
		}
		var ok bool
		tgt.Init, ok = clientCode.StorageInits[platStgTgt.Type]
		if !ok {
			log.Warn().
				Str("StorageTarget", platStgTgt.Name).
				Msgf("No StorageInit for target")
		}
		stgTgtInits[platStgTgt.Name] = tgt
	}
	for _, cncHdlr := range clientCode.ConcernHandlers {
		cncHdlr.SetStorageTargets(stgTgtInits)
	}
}
