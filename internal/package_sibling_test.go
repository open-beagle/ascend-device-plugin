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

import (
	"reflect"
	"testing"
)

func TestExpandPackagePhyIDs(t *testing.T) {
	devs := []DevicePhyInfo{
		{PhyID: 0, CardID: 0},
		{PhyID: 1, CardID: 0},
		{PhyID: 2, CardID: 1},
		{PhyID: 3, CardID: 1},
	}
	tests := []struct {
		name string
		in   []int32
		want []int32
	}{
		{name: "single die expands to package pair", in: []int32{1}, want: []int32{0, 1}},
		{name: "already pair stays pair", in: []int32{0, 1}, want: []int32{0, 1}},
		{name: "two packages", in: []int32{1, 2}, want: []int32{0, 1, 2, 3}},
		{name: "unknown phy kept", in: []int32{9}, want: []int32{9}},
		{name: "empty", in: nil, want: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExpandPackagePhyIDs(devs, tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("ExpandPackagePhyIDs(%v)=%v want %v", tt.in, got, tt.want)
			}
		})
	}
}
