/*
 * Copyright 2024 The HAMi Authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package internal

import "sort"

// DevicePhyInfo is the minimal topology needed to expand package sibling dies.
type DevicePhyInfo struct {
	PhyID  int32
	CardID int32
}

// ExpandPackagePhyIDs adds PhyIDs of sibling dies that share the same CardID.
// Ascend A3 packages expose two dies per card; npu-smi board-type queries need both.
func ExpandPackagePhyIDs(devs []DevicePhyInfo, phyIDs []int32) []int32 {
	if len(phyIDs) == 0 || len(devs) == 0 {
		return phyIDs
	}
	phyToCard := make(map[int32]int32, len(devs))
	for _, d := range devs {
		phyToCard[d.PhyID] = d.CardID
	}
	seen := make(map[int32]struct{}, len(phyIDs)*2)
	out := make([]int32, 0, len(phyIDs)*2)
	add := func(id int32) {
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	for _, id := range phyIDs {
		add(id)
		card, ok := phyToCard[id]
		if !ok {
			continue
		}
		for _, d := range devs {
			if d.CardID == card {
				add(d.PhyID)
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
