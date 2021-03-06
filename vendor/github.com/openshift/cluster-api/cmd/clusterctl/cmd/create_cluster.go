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

package cmd

import (
	"io/ioutil"

	"github.com/openshift/cluster-api/cmd/clusterctl/clusterdeployer"
	"github.com/openshift/cluster-api/cmd/clusterctl/clusterdeployer/bootstrap"
	"github.com/openshift/cluster-api/cmd/clusterctl/clusterdeployer/clusterclient"
	clustercommon "github.com/openshift/cluster-api/pkg/apis/cluster/common"
	"github.com/openshift/cluster-api/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/klog"
)

type CreateOptions struct {
	Cluster            string
	Machine            string
	ProviderComponents string
	AddonComponents    string
	Provider           string
	KubeconfigOutput   string
	BootstrapFlags     bootstrap.Options
}

var co = &CreateOptions{}

var createClusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Create kubernetes cluster",
	Long:  `Create a kubernetes cluster with one command`,
	Run: func(cmd *cobra.Command, args []string) {
		if co.Cluster == "" {
			exitWithHelp(cmd, "Please provide yaml file for cluster definition.")
		}
		if co.Machine == "" {
			exitWithHelp(cmd, "Please provide yaml file for machine definition.")
		}
		if co.ProviderComponents == "" {
			exitWithHelp(cmd, "Please provide yaml file for provider component definition.")
		}
		if err := RunCreate(co); err != nil {
			klog.Exit(err)
		}
	},
}

func RunCreate(co *CreateOptions) error {
	c, err := util.ParseClusterYaml(co.Cluster)
	if err != nil {
		return err
	}
	m, err := util.ParseMachinesYaml(co.Machine)
	if err != nil {
		return err
	}

	bootstrapProvider, err := bootstrap.Get(co.BootstrapFlags)
	if err != nil {
		return err
	}

	pd, err := getProvider(co.Provider)
	if err != nil {
		return err
	}
	pc, err := ioutil.ReadFile(co.ProviderComponents)
	if err != nil {
		return errors.Wrapf(err, "error loading provider components file %q", co.ProviderComponents)
	}
	var ac []byte
	if co.AddonComponents != "" {
		ac, err = ioutil.ReadFile(co.AddonComponents)
		if err != nil {
			return errors.Wrapf(err, "error loading addons file %q", co.AddonComponents)
		}
	}
	pcsFactory := clusterdeployer.NewProviderComponentsStoreFactory()

	d := clusterdeployer.New(
		bootstrapProvider,
		clusterclient.NewFactory(),
		string(pc),
		string(ac),
		co.BootstrapFlags.Cleanup)

	return d.Create(c, m, pd, co.KubeconfigOutput, pcsFactory)
}

func init() {
	// Required flags
	createClusterCmd.Flags().StringVarP(&co.Cluster, "cluster", "c", "", "A yaml file containing cluster object definition. Required.")
	createClusterCmd.MarkFlagRequired("cluster")
	createClusterCmd.Flags().StringVarP(&co.Machine, "machines", "m", "", "A yaml file containing machine object definition(s). Required.")
	createClusterCmd.MarkFlagRequired("machines")
	createClusterCmd.Flags().StringVarP(&co.ProviderComponents, "provider-components", "p", "", "A yaml file containing cluster api provider controllers and supporting objects. Required.")
	createClusterCmd.MarkFlagRequired("provider-components")
	// TODO: Remove as soon as code allows https://github.com/kubernetes-sigs/cluster-api/issues/157
	createClusterCmd.Flags().StringVarP(&co.Provider, "provider", "", "", "Which provider deployment logic to use (google/vsphere/azure). Required.")
	createClusterCmd.MarkFlagRequired("provider")

	// Optional flags
	createClusterCmd.Flags().StringVarP(&co.AddonComponents, "addon-components", "a", "", "A yaml file containing cluster addons to apply to the internal cluster")
	createClusterCmd.Flags().StringVarP(&co.KubeconfigOutput, "kubeconfig-out", "", "kubeconfig", "Where to output the kubeconfig for the provisioned cluster")

	co.BootstrapFlags.AddFlags(createClusterCmd.Flags())
	createCmd.AddCommand(createClusterCmd)
}

func getProvider(name string) (clusterdeployer.ProviderDeployer, error) {
	provisioner, err := clustercommon.ClusterProvisioner(name)
	if err != nil {
		return nil, err
	}
	provider, ok := provisioner.(clusterdeployer.ProviderDeployer)
	if !ok {
		return nil, errors.Errorf("provider for %s does not implement ProviderDeployer interface", name)
	}
	return provider, nil
}
