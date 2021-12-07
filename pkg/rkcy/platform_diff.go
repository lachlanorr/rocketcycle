// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
)

type platformDiff struct {
	progsToStop  []*rkcypb.Program
	progsToStart []*rkcypb.Program
}

func (rtPlatDef *rtPlatformDef) getAllProgs(adminBrokers string, otelcolEndpoint string) map[string]*rkcypb.Program {
	progs := make(map[string]*rkcypb.Program)
	if rtPlatDef != nil {
		for _, concern := range rtPlatDef.PlatformDef.Concerns {
			for _, topics := range concern.Topics {
				if topics.ConsumerPrograms != nil {
					exProgs := expandProgs(
						rtPlatDef.PlatformDef.Name,
						rtPlatDef.PlatformDef.Environment,
						adminBrokers,
						otelcolEndpoint,
						concern,
						topics,
						rtPlatDef.Clusters,
					)
					for _, p := range exProgs {
						progs[progKey(p)] = p
					}
				}
			}
		}
	}
	return progs
}

func (lhs *rtPlatformDef) diff(rhs *rtPlatformDef, adminBrokers string, otelcolEndpoint string) *platformDiff {
	d := &platformDiff{
		progsToStop:  nil,
		progsToStart: nil,
	}

	newProgs := lhs.getAllProgs(adminBrokers, otelcolEndpoint)
	oldProgs := rhs.getAllProgs(adminBrokers, otelcolEndpoint)

	for k, v := range newProgs {
		if _, ok := oldProgs[k]; !ok {
			d.progsToStart = append(d.progsToStart, v)
		}
	}

	for k, v := range oldProgs {
		if _, ok := newProgs[k]; !ok {
			d.progsToStop = append(d.progsToStop, v)
		}
	}

	return d
}
