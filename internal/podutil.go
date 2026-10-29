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
	v1 "k8s.io/api/core/v1"
)

func IsHAMiCorePod(pod *v1.Pod) bool {
	if pod == nil || pod.Annotations == nil {
		return false
	}
	return pod.Annotations[VNPUModeAnnotation] == VNPUModeHamiCore
}

func GetResourceLimitFromPod(pod *v1.Pod, resourceName string) int64 {
	if resourceName == "" || pod == nil {
		return 0
	}
	for _, c := range pod.Spec.Containers {
		for rName, rVal := range c.Resources.Limits {
			if string(rName) == resourceName {
				return rVal.Value()
			}
		}
	}
	return 0
}

func PodHasEnv(pod *v1.Pod, envName string) bool {
	if pod == nil {
		return false
	}
	for _, c := range pod.Spec.Containers {
		for _, e := range c.Env {
			if e.Name == envName {
				return true
			}
		}
	}
	return false
}
