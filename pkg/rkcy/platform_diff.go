// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package rkcy

import (
	"github.com/lachlanorr/rocketcycle/pkg/rkcy/pb"
)

type platformDiff struct {
	progsToStop  []*pb.Program
	progsToStart []*pb.Program
}

func (rtPlat *rtPlatform) getAllProgs() map[string]*pb.Program {
	progs := make(map[string]*pb.Program)
	if rtPlat != nil {
		for _, concern := range rtPlat.Platform.Concerns {
			for _, topics := range concern.Topics {
				if topics.ConsumerProgram != nil {
					exProgs := expandProgs(concern, topics, rtPlat.Clusters)
					for _, p := range exProgs {
						progs[progKey(p)] = p
					}
				}
			}
		}
	}
	return progs
}

func (lhs *rtPlatform) diff(rhs *rtPlatform) *platformDiff {
	d := &platformDiff{
		progsToStop:  nil,
		progsToStart: nil,
	}

	newProgs := lhs.getAllProgs()
	oldProgs := rhs.getAllProgs()

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
