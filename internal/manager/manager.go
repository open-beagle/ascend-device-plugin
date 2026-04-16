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

package manager

import (
	"fmt"
	"sort"

	"ascend-common/devmanager"
	"ascend-common/devmanager/dcmi"

	"github.com/Project-HAMi/ascend-device-plugin/internal"
	"k8s.io/klog/v2"
)

type Device struct {
	UUID     string
	LogicID  int32
	PhyID    int32
	CardID   int32
	DeviceID int32
	Memory   int64
	AICore   int32
	Health   bool
}

type AscendManager struct {
	mgr *devmanager.DeviceManager
	//nodeName string
	config          internal.VNPUConfig
	devs            []*Device
	vDeviceCount    int
	currentTemplate *internal.Template
}

func NewAscendManager() (*AscendManager, error) {
	mgr, err := devmanager.AutoInit("", 30)
	if err != nil {
		return nil, err
	}
	return &AscendManager{
		mgr:  mgr,
		devs: []*Device{},
	}, nil
}

func (am *AscendManager) LoadConfig(path string) error {
	config, err := internal.LoadConfig(path)
	if err != nil {
		return err
	}
	chipInfo, err := am.mgr.GetValidChipInfo()
	if err != nil {
		return err
	}
	if chipInfo.Type != "Ascend" {
		return fmt.Errorf("chip type is not Ascend")
	}
	idx := -1
	for i, vnpu := range config.VNPUs {
		if vnpu.ChipName == chipInfo.Name {
			idx = i
			break
		}
	}
	if idx == -1 {
		return fmt.Errorf("can not find vnpu config for chip %s", chipInfo.Name)
	}
	am.config = config.VNPUs[idx]
	sort.Slice(am.config.Templates, func(i, j int) bool {
		return am.config.Templates[i].Memory < am.config.Templates[j].Memory
	})
	klog.Infof("load config: %v", am.config)
	am.ApplySplitCount(0) // 0 implies default split logic
	return nil
}

// ApplySplitCount applies the dynamic split-count annotation to discover the matching template.
func (am *AscendManager) ApplySplitCount(splitCount uint) bool {
	if len(am.config.Templates) == 0 {
		am.currentTemplate = nil
		if am.vDeviceCount != 1 {
			am.vDeviceCount = 1
			return true
		}
		return false
	}

	var selectedTemplate *internal.Template
	if splitCount == 0 {
		// If 0 (annotation removed), fallback to the smallest template (original behavior)
		selectedTemplate = &am.config.Templates[0]
	} else {
		targetMem := am.config.MemoryAllocatable / int64(splitCount)
		for i := range am.config.Templates {
			if am.config.Templates[i].Memory >= targetMem {
				selectedTemplate = &am.config.Templates[i]
				break
			}
		}
		if selectedTemplate == nil {
			// Fallback if TargetMem is too big (shouldn't happen since valid templates are within capacity)
			// or if there's no template big enough. Actually if splitCount is very large, TargetMem is tiny.
			// The sorted array has the smallest template first. It should hit index 0 immediately if TargetMem is tiny.
			selectedTemplate = &am.config.Templates[0]
		}
	}

	newVCount := int(am.config.MemoryAllocatable / selectedTemplate.Memory)
	if am.currentTemplate == selectedTemplate && am.vDeviceCount == newVCount {
		return false // No change
	}

	am.currentTemplate = selectedTemplate
	am.vDeviceCount = newVCount
	klog.Infof("Applied dynamic split-count=%d, selected template='%s' (Memory: %d, AICore: %d), resulting vDeviceCount=%d",
		splitCount, selectedTemplate.Name, selectedTemplate.Memory, selectedTemplate.AICore, am.vDeviceCount)
	return true
}

func (am *AscendManager) CommonWord() string {
	return am.config.CommonWord
}

func (am *AscendManager) ResourceName() string {
	return am.config.ResourceName
}
func (am *AscendManager) ResourceMemoryName() string {
	return am.config.ResourceMemoryName
}

func (am *AscendManager) CurrentTemplateName() string {
	if am.currentTemplate != nil {
		return am.currentTemplate.Name
	}
	return ""
}

func (am *AscendManager) VDeviceCount() int {
	return am.vDeviceCount
}

func (am *AscendManager) UpdateDevice() error {
	_, IDs, err := am.mgr.GetDeviceList()
	if err != nil {
		klog.Errorf("failed to get device list: %v", err)
		return err
	}

	am.devs = make([]*Device, 0, len(IDs))
	for _, ID := range IDs {
		phyID, err := am.mgr.GetPhysicIDFromLogicID(ID)
		if err != nil {
			klog.Errorf("failed to get physic id from logic id: %v", err)
			return err
		}
		cardID, deviceID, err := am.mgr.GetCardIDDeviceID(ID)
		if err != nil {
			klog.Errorf("failed to get card id from device id: %v", err)
			return err
		}
		uuid, err := am.mgr.GetDieID(ID, dcmi.VDIE)
		if err != nil {
			klog.Errorf("failed to get uuid from device id: %v", err)
			return err
		}
		health, err := am.mgr.GetDeviceHealth(ID)
		if err != nil {
			klog.Errorf("failed to get device health: %v", err)
			return err
		}
		am.devs = append(am.devs, &Device{
			UUID:     uuid,
			LogicID:  ID,
			PhyID:    phyID,
			CardID:   cardID,
			DeviceID: deviceID,
			Memory:   am.config.MemoryAllocatable,
			AICore:   am.config.AICore,
			Health:   health == 0,
		})
	}
	return nil
}

func (am *AscendManager) GetDevices() []*Device {
	return am.devs
}

func (am *AscendManager) GetDeviceByUUID(UUID string) *Device {
	for _, dev := range am.devs {
		if dev.UUID == UUID {
			return dev
		}
	}
	return nil
}

func (am *AscendManager) GetIDs() []int32 {
	_, IDs, err := am.mgr.GetDeviceList()
	if err != nil {
		return nil
	}
	return IDs
}

func (am *AscendManager) GetUnHealthIDs() []int32 {
	_, IDs, err := am.mgr.GetDeviceList()
	if err != nil {
		return nil
	}
	var unhealthy []int32
	for _, d := range IDs {
		healthCode, err := am.mgr.GetDeviceHealth(d)
		if err != nil {
			continue
		}
		if healthCode != 0 {
			unhealthy = append(unhealthy, d)
		}
	}
	return unhealthy
}
