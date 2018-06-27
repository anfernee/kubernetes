/*
Copyright 2018 The Kubernetes Authors.

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
	"io"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apiserver/pkg/admission"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	api "k8s.io/kubernetes/pkg/apis/core"
	"k8s.io/kubernetes/pkg/features"
)

// PluginName indicates name of admission plugin.
const PluginName = "PodReady"

// ReadinessConditionType is the condition type that indicates pod readiness
const ReadinessConditionType = "dummy"

// Register registers a plugin
func Register(plugins *admission.Plugins) {
	plugins.Register(PluginName, func(config io.Reader) (admission.Interface, error) {
		return NewPodReady(), nil
	})
}

type podReady struct {
	*admission.Handler

	// for testing
	features utilfeature.FeatureGate
}

var _ admission.MutationInterface = &podReady{}

// Admit makes an admission decision based on the request attributes
func (p *podReady) Admit(a admission.Attributes) (err error) {
	switch a.GetResource().GroupResource() {
	case api.Resource("pods"):
		return p.admitPod(a)
	default:
		return nil
	}

}

func (p *podReady) admitPod(a admission.Attributes) (err error) {
	if !p.features.Enabled(features.PodReadinessGates) {
		return nil
	}

	pod, ok := a.GetObject().(*api.Pod)
	if !ok {
		return errors.NewBadRequest("Resource was marked with kind Pod but was unable to be converted")
	}

	pod.Spec.ReadinessGates = append(pod.Spec.ReadinessGates, api.PodReadinessGate{ReadinessConditionType})
	return nil
}

// NewPodReady creates a new pod ready admit admission handler
func NewPodReady() admission.MutationInterface {
	return &podReady{
		Handler:  admission.NewHandler(admission.Create),
		features: utilfeature.DefaultFeatureGate,
	}
}
