/*
Copyright 2018 The Kubernetes Authors

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

package podready

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apiserver/pkg/admission"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	api "k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/features"
)

var (
	readyEnabledFeature  = utilfeature.NewFeatureGate()
	readyDisabledFeature = utilfeature.NewFeatureGate()
)

func init() {
	if err := readyEnabledFeature.Add(map[utilfeature.Feature]utilfeature.FeatureSpec{features.PodReadinessGates: {Default: true}}); err != nil {
		panic(err)
	}
	if err := readyDisabledFeature.Add(map[utilfeature.Feature]utilfeature.FeatureSpec{features.PodReadinessGates: {Default: false}}); err != nil {
		panic(err)
	}
}

func TestPodReadyAdmission(t *testing.T) {

	tests := []struct {
		name                string
		obj                 runtime.Object
		features            utilfeature.FeatureGate
		expectReadinessGate bool
		expectError         bool
	}{
		{
			"it should add readiness gate to pod spec",
			&api.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-name",
					Namespace: "default",
				},
			},
			readyEnabledFeature,
			true,
			false,
		},
		{
			"it shouldn't add readiness gate to pod spec if PodReadinessGate isn't enabled",
			&api.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod-name",
					Namespace: "default",
				},
			},
			readyDisabledFeature,
			false,
			false,
		},
		{
			"it should error out if object type isn't pod",
			&api.Node{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "node-name",
					Namespace: "default",
				},
			},
			readyEnabledFeature,
			false,
			true,
		},
	}

	for _, test := range tests {
		plugin := NewPodReady()
		plugin.(*podReady).features = test.features

		attrs := admission.NewAttributesRecord(
			test.obj,
			nil,
			test.obj.GetObjectKind().GroupVersionKind(),
			"default",
			"",
			schema.GroupVersionResource{Resource: "pods"},
			"",
			admission.Create,
			nil,
		)

		err := plugin.Admit(attrs)
		if test.expectError {
			if err == nil {
				t.Errorf("Test %q: expected error and no error recevied", test.name)
			}
		} else {
			if err != nil {
				t.Errorf("Test %q: unexpected error received: %v", test.name, err)
			}
		}

		if test.expectReadinessGate {
			thePod := test.obj.(*api.Pod)
			if len(thePod.Spec.ReadinessGates) == 0 {
				t.Errorf("Test %q: unexpected empty pod.Spec.ReadinessGates", test.name)
			}

			found := 0
			for _, rg := range thePod.Spec.ReadinessGates {
				if rg.ConditionType == ReadinessConditionType {
					found++
				}
			}
			if found != 1 {
				t.Errorf("Test %q: expected find 1 ReadinessGates, but got %d", test.name, found)
			}
		}
	}
}
