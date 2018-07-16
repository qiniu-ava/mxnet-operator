/*
Copyright 2017 The Kubernetes Authors.

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

package api

import (
	"reflect"
	"testing"

	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func nodeInfoEqual(l, r *NodeInfo) bool {
	if !reflect.DeepEqual(l, r) {
		return false
	}

	return true
}

func TestNodeInfo_AddPod(t *testing.T) {
	// case1
	case01_node := buildNode("n1", buildResourceList("8000m", "10G"))
	case01_pod1 := buildPod("c1", "p1", "n1", v1.PodRunning, buildResourceList("1000m", "1G"), []metav1.OwnerReference{}, make(map[string]string))
	case01_pod2 := buildPod("c1", "p2", "n1", v1.PodRunning, buildResourceList("2000m", "2G"), []metav1.OwnerReference{}, make(map[string]string))

	tests := []struct {
		name     string
		node     *v1.Node
		pods     []*v1.Pod
		expected *NodeInfo
	}{
		{
			name: "add 2 running non-owner pod",
			node: case01_node,
			pods: []*v1.Pod{case01_pod1, case01_pod2},
			expected: &NodeInfo{
				Name:        "n1",
				Node:        case01_node,
				Idle:        buildResource("5000m", "7G"),
				Used:        buildResource("3000m", "3G"),
				Releasing:   EmptyResource(),
				Allocatable: buildResource("8000m", "10G"),
				Capability:  buildResource("8000m", "10G"),
				Tasks: map[TaskID]*TaskInfo{
					"c1/p1": NewTaskInfo(case01_pod1),
					"c1/p2": NewTaskInfo(case01_pod2),
				},
			},
		},
	}

	for i, test := range tests {
		ni := NewNodeInfo(test.node)

		for _, pod := range test.pods {
			pi := NewTaskInfo(pod)
			ni.AddTask(pi)
		}

		if !nodeInfoEqual(ni, test.expected) {
			t.Errorf("node info %d: \n expected %v, \n got %v \n",
				i, test.expected, ni)
		}
	}
}

func TestNodeInfo_RemovePod(t *testing.T) {
	// case1
	case01_node := buildNode("n1", buildResourceList("8000m", "10G"))
	case01_pod1 := buildPod("c1", "p1", "n1", v1.PodRunning, buildResourceList("1000m", "1G"), []metav1.OwnerReference{}, make(map[string]string))
	case01_pod2 := buildPod("c1", "p2", "n1", v1.PodRunning, buildResourceList("2000m", "2G"), []metav1.OwnerReference{}, make(map[string]string))
	case01_pod3 := buildPod("c1", "p3", "n1", v1.PodRunning, buildResourceList("3000m", "3G"), []metav1.OwnerReference{}, make(map[string]string))

	tests := []struct {
		name     string
		node     *v1.Node
		pods     []*v1.Pod
		rmPods   []*v1.Pod
		expected *NodeInfo
	}{
		{
			name:   "add 3 running non-owner pod, remove 1 running non-owner pod",
			node:   case01_node,
			pods:   []*v1.Pod{case01_pod1, case01_pod2, case01_pod3},
			rmPods: []*v1.Pod{case01_pod2},
			expected: &NodeInfo{
				Name:        "n1",
				Node:        case01_node,
				Idle:        buildResource("4000m", "6G"),
				Used:        buildResource("4000m", "4G"),
				Releasing:   EmptyResource(),
				Allocatable: buildResource("8000m", "10G"),
				Capability:  buildResource("8000m", "10G"),
				Tasks: map[TaskID]*TaskInfo{
					"c1/p1": NewTaskInfo(case01_pod1),
					"c1/p3": NewTaskInfo(case01_pod3),
				},
			},
		},
	}

	for i, test := range tests {
		ni := NewNodeInfo(test.node)

		for _, pod := range test.pods {
			pi := NewTaskInfo(pod)
			ni.AddTask(pi)
		}

		for _, pod := range test.rmPods {
			pi := NewTaskInfo(pod)
			ni.RemoveTask(pi)
		}

		if !nodeInfoEqual(ni, test.expected) {
			t.Errorf("node info %d: \n expected %v, \n got %v \n",
				i, test.expected, ni)
		}
	}
}
