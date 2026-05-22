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

package overcommit

import (
	"encoding/json"
	"testing"
	"time"
)

func TestMaxMemory(t *testing.T) {
	tests := []struct {
		utilization uint32
		level       string
		percent     int32
		limit       int32
	}{
		{4, LoadLevelIdle, 100, 65536},
		{19, LoadLevelLow, 100, 65536},
		{20, LoadLevelNormal, 50, 32768},
		{60, LoadLevelHighWatermark, 25, 16384},
		{80, LoadLevelFull, 10, 6553},
	}
	for _, test := range tests {
		level, percent, limit := MaxMemory(65536, test.utilization)
		if level != test.level || percent != test.percent || limit != test.limit {
			t.Fatalf("MaxMemory(%d) = (%s,%d,%d), want (%s,%d,%d)", test.utilization, level, percent, limit, test.level, test.percent, test.limit)
		}
	}
}

func TestMergeAndClearState(t *testing.T) {
	now := time.Now()
	rawState := State{
		"GPU-0":   {GPUUtilization: 10, LoadLevel: LoadLevelLow, MaxMemoryPercent: 100, MaxMemoryLimit: 24000, UpdatedAt: now},
		"NPU-old": {GPUUtilization: 20, LoadLevel: LoadLevelNormal, MaxMemoryPercent: 50, MaxMemoryLimit: 32768, UpdatedAt: now},
	}
	rawData, err := json.Marshal(rawState)
	if err != nil {
		t.Fatal(err)
	}

	merged, err := MergeState(string(rawData), []string{"NPU-old", "NPU-0"}, State{
		"NPU-0": {GPUUtilization: 80, LoadLevel: LoadLevelFull, MaxMemoryPercent: 10, MaxMemoryLimit: 6553, UpdatedAt: now},
	})
	if err != nil {
		t.Fatal(err)
	}
	var got State
	if err := json.Unmarshal([]byte(merged), &got); err != nil {
		t.Fatal(err)
	}
	if _, ok := got["NPU-old"]; ok {
		t.Fatal("old owned NPU state was not removed")
	}
	if _, ok := got["NPU-0"]; !ok {
		t.Fatal("new NPU state was not merged")
	}
	if _, ok := got["GPU-0"]; !ok {
		t.Fatal("foreign GPU state should be preserved")
	}

	cleared, err := ClearState(merged, []string{"NPU-0"})
	if err != nil {
		t.Fatal(err)
	}
	got = nil
	if err := json.Unmarshal([]byte(cleared), &got); err != nil {
		t.Fatal(err)
	}
	if _, ok := got["NPU-0"]; ok {
		t.Fatal("owned NPU state was not cleared")
	}
	if _, ok := got["GPU-0"]; !ok {
		t.Fatal("foreign GPU state should be preserved after clear")
	}
}
