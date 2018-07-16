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
package job

import (
	"path/filepath"

	"github.com/spf13/cobra"
)

type commonFlags struct {
	Master        string
	Kubeconfig    string
	SchedulerName string
}

func initFlags(cmd *cobra.Command, cf *commonFlags) {
	cmd.Flags().StringVarP(&cf.SchedulerName, "scheduler", "", "kar-scheduler", "the scheduler for this job")
	cmd.Flags().StringVarP(&cf.Master, "master", "s", "", "the address of apiserver")

	if home := homeDir(); home != "" {
		cmd.Flags().StringVarP(&cf.Kubeconfig, "kubeconfig", "", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		cmd.Flags().StringVarP(&cf.Kubeconfig, "kubeconfig", "", "", "(optional) absolute path to the kubeconfig file")
	}
}
