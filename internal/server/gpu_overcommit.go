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

package server

import (
	"context"
	"encoding/json"
	"time"

	"github.com/Project-HAMi/ascend-device-plugin/internal/overcommit"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"
	"k8s.io/klog/v2"

	"github.com/Project-HAMi/HAMi/pkg/util"
	"github.com/Project-HAMi/HAMi/pkg/util/client"
)

func (ps *PluginServer) collectGPUOvercommitState() (overcommit.State, []string, error) {
	devices := ps.mgr.GetDevices()
	now := time.Now()
	state := make(overcommit.State, len(devices))
	ownedIDs := ps.gpuOvercommitDeviceIDs()

	for _, dev := range devices {
		if dev == nil {
			continue
		}
		utilization, err := ps.mgr.GetAICoreUtilization(dev.LogicID)
		if err != nil {
			klog.ErrorS(err, "get ascend ai core utilization failed", "uuid", dev.UUID, "logicID", dev.LogicID)
			continue
		}
		level, percent, limit := overcommit.MaxMemory(int32(dev.Memory), utilization)
		state[dev.UUID] = overcommit.DeviceState{
			GPUUtilization:   utilization,
			LoadLevel:        level,
			MaxMemoryPercent: percent,
			MaxMemoryLimit:   limit,
			UpdatedAt:        now,
		}
	}

	return state, ownedIDs, overcommit.WriteStateFile(overcommit.AscendStateFile, state)
}

func (ps *PluginServer) gpuOvercommitDeviceIDs() []string {
	devices := ps.mgr.GetDevices()
	ids := make([]string, 0, len(devices))
	for _, dev := range devices {
		if dev == nil {
			continue
		}
		ids = append(ids, dev.UUID)
	}
	return ids
}

func (ps *PluginServer) patchGPUOvercommitState(merge func(map[string]string) (string, error)) error {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		node, err := util.GetNode(ps.nodeName)
		if err != nil {
			return err
		}
		stateRaw, err := merge(node.Annotations)
		if err != nil {
			return err
		}
		patch, err := gpuOvercommitStatePatch(node.ResourceVersion, node.Annotations, stateRaw)
		if err != nil {
			return err
		}
		_, err = client.GetClient().CoreV1().Nodes().Patch(context.Background(), ps.nodeName, k8stypes.JSONPatchType, patch, metav1.PatchOptions{})
		if err == nil {
			return nil
		}
		lastErr = err
		if !apierrors.IsConflict(err) {
			return err
		}
		klog.V(3).InfoS("retry gpu overcommit state patch after conflict", "node", ps.nodeName, "attempt", attempt+1)
	}
	return lastErr
}

func gpuOvercommitStatePatch(resourceVersion string, annotations map[string]string, stateRaw string) ([]byte, error) {
	type patchOperation struct {
		Op    string `json:"op"`
		Path  string `json:"path"`
		Value any    `json:"value"`
	}
	ops := []patchOperation{
		{Op: "test", Path: "/metadata/resourceVersion", Value: resourceVersion},
	}
	if annotations == nil {
		ops = append(ops, patchOperation{
			Op:    "add",
			Path:  "/metadata/annotations",
			Value: map[string]string{overcommit.StateAnnotation: stateRaw},
		})
	} else {
		ops = append(ops, patchOperation{
			Op:    "add",
			Path:  "/metadata/annotations/" + jsonPatchEscape(overcommit.StateAnnotation),
			Value: stateRaw,
		})
	}
	return json.Marshal(ops)
}

func jsonPatchEscape(key string) string {
	escaped := ""
	for _, ch := range key {
		switch ch {
		case '~':
			escaped += "~0"
		case '/':
			escaped += "~1"
		default:
			escaped += string(ch)
		}
	}
	return escaped
}
