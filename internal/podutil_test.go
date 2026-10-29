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
	"encoding/json"
	"testing"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIsHAMiCorePod(t *testing.T) {
	if IsHAMiCorePod(nil) {
		t.Fatal("nil pod should not be hami-core")
	}
	pod := &v1.Pod{ObjectMeta: metav1.ObjectMeta{Annotations: map[string]string{
		VNPUModeAnnotation: VNPUModeHamiCore,
	}}}
	if !IsHAMiCorePod(pod) {
		t.Fatal("expected hami-core pod")
	}
	pod.Annotations[VNPUModeAnnotation] = "template"
	if IsHAMiCorePod(pod) {
		t.Fatal("template mode must not be hami-core")
	}
}

func TestAllocRuntimeInfoJSON(t *testing.T) {
	raw := `[{"UUID":"dev-0","temp":"vir05_1c_16g","memory":16384,"core":5},{"UUID":"dev-1","memory":20480,"core":40}]`
	var infos []AllocRuntimeInfo
	if err := json.Unmarshal([]byte(raw), &infos); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(infos) != 2 {
		t.Fatalf("len=%d", len(infos))
	}
	if infos[0].Temp != "vir05_1c_16g" || infos[0].Memory != 16384 {
		t.Fatalf("hard-split info: %+v", infos[0])
	}
	if infos[1].Temp != "" || infos[1].Memory != 20480 || infos[1].Core != 40 {
		t.Fatalf("soft-split info: %+v", infos[1])
	}
}

func TestGetResourceLimitFromPod(t *testing.T) {
	pod := &v1.Pod{Spec: v1.PodSpec{Containers: []v1.Container{{
		Resources: v1.ResourceRequirements{
			Limits: v1.ResourceList{
				"huawei.com/Ascend910C-memory": resource.MustParse("20480"),
				"huawei.com/Ascend910C-core":   resource.MustParse("40"),
			},
		},
	}}}}
	if got := GetResourceLimitFromPod(pod, "huawei.com/Ascend910C-memory"); got != 20480 {
		t.Fatalf("memory got %d", got)
	}
	if got := GetResourceLimitFromPod(pod, "huawei.com/Ascend910C-core"); got != 40 {
		t.Fatalf("core got %d", got)
	}
	if got := GetResourceLimitFromPod(pod, ""); got != 0 {
		t.Fatalf("empty name got %d", got)
	}
}
