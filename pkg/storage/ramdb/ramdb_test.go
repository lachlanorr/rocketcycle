// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at https://mozilla.org/MPL/2.0/.

package ramdb

import (
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/lachlanorr/rocketcycle/pkg/rkcypb"
)

func TestRamDb(t *testing.T) {
	db := NewRamDb()

	if db.items == nil {
		t.Fatal("Nil items in new RamDb")
	}
	if len(db.items) != 0 {
		t.Fatal("Non-zero item count in new RamDb")
	}

	{
		_, _, _, err := db.Read("key0")
		if err == nil {
			t.Fatal("Non error when reading missing key")
		}
	}

	// write key0
	{
		cmpdOffset := rkcypb.CompoundOffset{
			Generation: 1,
			Partition:  10,
			Offset:     20,
		}

		// create
		inst, err := db.Create("key0", gTestItems["item0"].inst, &cmpdOffset)
		if err != nil {
			t.Fatalf("Error in ramdb.Create: %s", err.Error())
		}
		if !instEqual(inst, gTestItems["item0"].inst) {
			t.Fatal("Create returned different inst from input")
		}
		if inst == gTestItems["item0"].inst {
			t.Fatal("same pointer value returned")
		}

		// validate read
		inst, rel, retCmpdOffset, err := db.Read("key0")
		if err != nil {
			t.Fatalf("Error when reading: %s", err.Error())
		}
		if !instEqual(inst, gTestItems["item0"].inst) {
			t.Fatal("inst mismatch after Read")
		}
		if inst == gTestItems["item0"].inst {
			t.Fatal("same pointer value returned")
		}
		if rel != nil {
			t.Fatal("non-nil rel on Read")
		}
		if !cmpdOffsetEqual(&cmpdOffset, retCmpdOffset) {
			t.Fatal("inst mismatch after Read")
		}

		// modify and update
		cmpdOffset2 := cmpdOffset
		cmpdOffset2.Offset++
		inst2 := *gTestItems["item0"].inst
		inst2.Name += "modified"
		err = db.Update("key0", &inst2, gTestItems["item0"].rel, &cmpdOffset2)
		if err != nil {
			t.Fatalf("Error in ramdb.Create: %s", err.Error())
		}

		// validate read
		inst, rel, retCmpdOffset, err = db.Read("key0")
		if err != nil {
			t.Fatalf("Error when reading: %s", err.Error())
		}
		if !instEqual(inst, &inst2) {
			t.Fatal("inst mismatch after Read")
		}
		if &inst2 == inst {
			t.Fatal("same pointer value returned")
		}
		if !relEqual(rel, gTestItems["item0"].rel) {
			t.Fatal("rel mismatch after Read")
		}
		if rel == gTestItems["item0"].rel {
			t.Fatal("same pointer value returned")
		}
		if !cmpdOffsetEqual(&cmpdOffset2, retCmpdOffset) {
			t.Fatal("inst mismatch after Read")
		}

		// modify inst and rel, update and validate
		cmpdOffset3 := cmpdOffset2
		cmpdOffset3.Offset++
		inst3 := *gTestItems["item0"].inst
		inst3.Name += "modifiedAgain"
		rel3 := *gTestItems["item0"].rel
		rel3.Name += "relModified"
		err = db.Update("key0", &inst3, &rel3, &cmpdOffset3)
		if err != nil {
			t.Fatalf("Error in ramdb.Create: %s", err.Error())
		}

		// validate read
		inst, rel, retCmpdOffset, err = db.Read("key0")
		if err != nil {
			t.Fatalf("Error when reading: %s", err.Error())
		}
		if !instEqual(inst, &inst3) {
			t.Fatal("inst mismatch after Read")
		}
		if &inst3 == inst {
			t.Fatal("same pointer value returned")
		}
		if !relEqual(rel, &rel3) {
			t.Fatal("rel mismatch after Read")
		}
		if &rel3 == rel {
			t.Fatal("same pointer value returned")
		}
		if !cmpdOffsetEqual(&cmpdOffset3, retCmpdOffset) {
			t.Fatal("inst mismatch after Read")
		}

		// update with lower offset should be a no-op
		err = db.Update("key0", gTestItems["item0"].inst, gTestItems["item0"].rel, &cmpdOffset)
		if err != nil {
			t.Fatalf("Error in ramdb.Create: %s", err.Error())
		}

		// validate read hasn't changed
		inst, rel, retCmpdOffset, err = db.Read("key0")
		if err != nil {
			t.Fatalf("Error when reading: %s", err.Error())
		}
		if !instEqual(inst, &inst3) {
			t.Fatal("inst mismatch after Read")
		}
		if &inst3 == inst {
			t.Fatal("same pointer value returned")
		}
		if !relEqual(rel, &rel3) {
			t.Fatal("rel mismatch after Read")
		}
		if &rel3 == rel {
			t.Fatal("same pointer value returned")
		}
		if !cmpdOffsetEqual(&cmpdOffset3, retCmpdOffset) {
			t.Fatal("inst mismatch after Read")
		}

		cmpdOffsetB := rkcypb.CompoundOffset{
			Generation: 101,
			Partition:  1010,
			Offset:     2010,
		}
		instB, errB := db.Create("key1", gTestItems["item1"].inst, &cmpdOffsetB)
		if errB != nil {
			t.Fatalf("Error in ramdb.Create: %s", errB.Error())
		}
		if !instEqual(instB, gTestItems["item1"].inst) {
			t.Fatal("Create returned different inst from input")
		}
		if instB == gTestItems["item1"].inst {
			t.Fatal("same pointer value returned")
		}

		instB, relB, retCmpdOffsetB, errB := db.Read("key1")
		if errB != nil {
			t.Fatalf("Error when reading: %s", errB.Error())
		}
		if !instEqual(instB, gTestItems["item1"].inst) {
			t.Fatal("inst mismatch after Read")
		}
		if instB == gTestItems["item1"].inst {
			t.Fatal("same pointer value returned")
		}
		if relB != nil {
			t.Fatal("non-nil rel on Read")
		}
		if !cmpdOffsetEqual(&cmpdOffsetB, retCmpdOffsetB) {
			t.Fatal("inst mismatch after Read")
		}

		// delete key0 with invalid offset, make sure no-op
		err = db.Delete("key0", &cmpdOffset3)
		if err != nil {
			t.Fatalf("Error in ramdb.Delete: %s", err.Error())
		}
		// make sure delete didn't happen
		inst, rel, retCmpdOffset, err = db.Read("key0")
		if err != nil {
			t.Fatalf("Error when reading: %s", err.Error())
		}
		if !instEqual(inst, &inst3) {
			t.Fatal("inst mismatch after Read")
		}
		if &inst3 == inst {
			t.Fatal("same pointer value returned")
		}
		if !relEqual(rel, &rel3) {
			t.Fatal("rel mismatch after Read")
		}
		if !cmpdOffsetEqual(&cmpdOffset3, retCmpdOffset) {
			t.Fatal("inst mismatch after Read")
		}

		// delete key0
		cmpdOffset4 := cmpdOffset3
		cmpdOffset4.Offset++
		err = db.Delete("key0", &cmpdOffset4)
		if err != nil {
			t.Fatalf("Error in ramdb.Delete: %s", errB.Error())
		}
		inst, rel, retCmpdOffset, err = db.Read("key0")
		if err == nil {
			t.Fatal("Successful read after deleting")
		}

		// make sure key1 is still there
		instB, relB, retCmpdOffsetB, errB = db.Read("key1")
		if errB != nil {
			t.Fatalf("Error when reading: %s", errB.Error())
		}
		if instB == gTestItems["item1"].inst {
			t.Fatal("same pointer value returned")
		}
		if !instEqual(instB, gTestItems["item1"].inst) {
			t.Fatal("inst mismatch after Read")
		}
		if relB != nil {
			t.Fatal("non-nil rel on Read")
		}
		if !cmpdOffsetEqual(&cmpdOffsetB, retCmpdOffsetB) {
			t.Fatal("inst mismatch after Read")
		}

		// delete key1
		cmpdOffsetB2 := cmpdOffsetB
		cmpdOffsetB2.Generation++
		cmpdOffsetB2.Partition = 0
		cmpdOffsetB2.Offset = 1
		errB = db.Delete("key1", &cmpdOffsetB2)
		if errB != nil {
			t.Fatalf("Error in ramdb.Delete: %s", errB.Error())
		}
		instB, relB, retCmpdOffsetB, errB = db.Read("key1")
		if errB == nil {
			t.Fatal("Successful read after deleting")
		}
	}
}

func instEqual(lhs proto.Message, rhs proto.Message) bool {
	lhsT, ok := lhs.(*rkcypb.StorageTarget)
	if !ok {
		return false
	}
	rhsT, ok := rhs.(*rkcypb.StorageTarget)
	if !ok {
		return false
	}

	return lhsT.String() == rhsT.String()
}

func relEqual(lhs proto.Message, rhs proto.Message) bool {
	lhsT, ok := lhs.(*rkcypb.Program)
	if !ok {
		return false
	}
	rhsT, ok := rhs.(*rkcypb.Program)
	if !ok {
		return false
	}

	return lhsT.String() == rhsT.String()
}

func cmpdOffsetEqual(lhs proto.Message, rhs proto.Message) bool {
	lhsT, ok := lhs.(*rkcypb.CompoundOffset)
	if !ok {
		return false
	}
	rhsT, ok := rhs.(*rkcypb.CompoundOffset)
	if !ok {
		return false
	}

	return lhsT.String() == rhsT.String()
}

type TestItem struct {
	inst *rkcypb.StorageTarget
	rel  *rkcypb.Program
}

// Use some simple core rkcy protobufs just for testing.
// Obviously inst and rel would generally be real concern types.
var gTestItems = map[string]*TestItem{
	"item0": &TestItem{
		inst: &rkcypb.StorageTarget{
			Name:      "item0_Inst_Name",
			Type:      "item0_Inst_Type",
			IsPrimary: true,
			Config: map[string]string{
				"item0_Inst_key0": "item0_Inst_val0",
				"item0_Inst_key1": "item0_Inst_val1",
				"item0_Inst_key2": "item0_Inst_val2",
			},
		},
		rel: &rkcypb.Program{
			Name:   "item0_Rel_Name",
			Args:   []string{"item0_Rel_arg0", "item0_Rel_arg1", "item0_Rel_arg2", "item0_Rel_arg3"},
			Abbrev: "item0_Rel_Abbrev",
			Tags: map[string]string{
				"item0_Rel_key0": "item0_Rel_val0",
				"item0_Rel_key1": "item0_Rel_val1",
				"item0_Rel_key2": "item0_Rel_val2",
			},
		},
	},
	"item1": &TestItem{
		inst: &rkcypb.StorageTarget{
			Name:      "item1_Inst_Name",
			Type:      "item1_Inst_Type",
			IsPrimary: true,
			Config: map[string]string{
				"item1_Inst_key0": "item1_Inst_val0",
				"item1_Inst_key1": "item1_Inst_val1",
				"item1_Inst_key2": "item1_Inst_val2",
				"item1_Inst_key3": "item1_Inst_val3",
				"item1_Inst_key4": "item1_Inst_val4",
				"item1_Inst_key5": "item1_Inst_val5",
			},
		},
		rel: &rkcypb.Program{
			Name:   "item1_Rel_Name",
			Args:   []string{"item1_Rel_arg0", "item1_Rel_arg1"},
			Abbrev: "item1_Rel_Abbrev",
			Tags: map[string]string{
				"item1_Rel_key0": "item1_Rel_val0",
				"item1_Rel_key1": "item1_Rel_val1",
			},
		},
	},
}
