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
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_910CTemplatesAndCore(t *testing.T) {
	content := `
hamiVnpuCore: true
vnpus:
- chipName: Ascend910
  commonWord: Ascend910C
  resourceName: huawei.com/Ascend910C
  resourceMemoryName: huawei.com/Ascend910C-memory
  resourceCoreName: huawei.com/Ascend910C-core
  memoryAllocatable: 65536
  memoryCapacity: 65536
  aiCore: 20
  aiCPU: 7
  superPod: true
  templates:
    - name: vir05_1c_16g
      memory: 16384
      aiCore: 5
      aiCPU: 1
    - name: vir10_3c_32g
      memory: 32768
      aiCore: 10
      aiCPU: 3
`
	dir := t.TempDir()
	path := filepath.Join(dir, "device-config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if !cfg.HamiVnpuCore {
		t.Fatal("expected hamiVnpuCore=true")
	}
	if len(cfg.VNPUs) != 1 {
		t.Fatalf("vnpus len=%d", len(cfg.VNPUs))
	}
	vnpu := cfg.VNPUs[0]
	if vnpu.CommonWord != "Ascend910C" || vnpu.ResourceCoreName != "huawei.com/Ascend910C-core" {
		t.Fatalf("unexpected vnpu: %+v", vnpu)
	}
	if !vnpu.SuperPod {
		t.Fatal("expected superPod=true")
	}
	if len(vnpu.Templates) != 2 || vnpu.Templates[0].Name != "vir05_1c_16g" {
		t.Fatalf("templates: %+v", vnpu.Templates)
	}
}
