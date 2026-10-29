/*
Copyright 2024 The HAMi Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package internal

import (
	"os"

	"k8s.io/apimachinery/pkg/util/yaml"
)

type Template struct {
	Name   string `json:"name"`
	Memory int64  `json:"memory"`
	AICore int32  `json:"aiCore,omitempty"`
	AICPU  int32  `json:"aiCPU,omitempty"`
}

type VNPUConfig struct {
	CommonWord         string     `json:"commonWord"`
	ChipName           string     `json:"chipName"`
	ResourceName       string     `json:"resourceName"`
	ResourceMemoryName string     `json:"resourceMemoryName"`
	ResourceCoreName   string     `json:"resourceCoreName"`
	MemoryAllocatable  int64      `json:"memoryAllocatable"`
	MemoryCapacity     int64      `json:"memoryCapacity"`
	AICore             int32      `json:"aiCore"`
	AICPU              int32      `json:"aiCPU"`
	Templates          []Template `json:"templates"`
	// SuperPod is consumed by HAMi scheduler; kept here for config parity.
	SuperPod bool `json:"superPod"`
}

// AllocRuntimeInfo mirrors HAMi ascend.RuntimeInfo for Allocate annotation parsing.
// Local struct avoids depending on HAMi module version for Memory/Core fields.
type AllocRuntimeInfo struct {
	UUID   string `json:"UUID,omitempty"`
	Temp   string `json:"temp,omitempty"`
	Memory int64  `json:"memory,omitempty"`
	Core   int32  `json:"core,omitempty"`
}

const (
	VNPUModeAnnotation         = "huawei.com/vnpu-mode"
	VNPUModeHamiCore           = "hami-core"
	VNPUNodeSelectorAnnotation = "hami-vnpu-core"
)

type Config struct {
	// HamiVnpuCore enables soft-split by default on this node (writes hami-vnpu-core=true).
	HamiVnpuCore bool         `json:"hamiVnpuCore"`
	VNPUs        []VNPUConfig `json:"vnpus"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var yamlData Config
	err = yaml.Unmarshal(data, &yamlData)
	if err != nil {
		return nil, err
	}
	return &yamlData, nil
}
