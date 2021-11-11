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

package scope

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	autorestazure "github.com/Azure/go-autorest/autorest/azure"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"

	infrav1 "sigs.k8s.io/cluster-api-provider-azure/api/v1beta1"
	"sigs.k8s.io/cluster-api-provider-azure/azure"
	"sigs.k8s.io/cluster-api-provider-azure/azure/services/disks"
	"sigs.k8s.io/cluster-api-provider-azure/azure/services/inboundnatrules"
)

func specArrayToString(specs []azure.ResourceSpecGetter) string {
	var sb strings.Builder
	sb.WriteString("[ ")
	for _, spec := range specs {
		sb.WriteString(fmt.Sprintf("%+v ", spec))
	}
	sb.WriteString("]")

	return sb.String()
}

func TestMachineScope_Name(t *testing.T) {
	tests := []struct {
		name         string
		machineScope MachineScope
		want         string
		testLength   bool
	}{
		{
			name: "if provider ID exists, use it",
			machineScope: MachineScope{
				AzureMachine: &infrav1.AzureMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine-with-a-long-name",
					},
					Spec: infrav1.AzureMachineSpec{
						ProviderID: to.StringPtr("azure://compute/virtual-machines/machine-name"),
						OSDisk: infrav1.OSDisk{
							OSType: "Windows",
						},
					},
				},
			},
			want: "machine-name",
		},
		{
			name: "linux can be any length",
			machineScope: MachineScope{
				AzureMachine: &infrav1.AzureMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine-with-really-really-long-name",
					},
					Spec: infrav1.AzureMachineSpec{
						OSDisk: infrav1.OSDisk{
							OSType: "Linux",
						},
					},
				},
			},
			want: "machine-with-really-really-long-name",
		},
		{
			name: "Windows name with long MachineName and short cluster name",
			machineScope: MachineScope{
				ClusterScoper: &ClusterScope{
					Cluster: &clusterv1.Cluster{
						ObjectMeta: metav1.ObjectMeta{
							Name: "cluster",
						},
					},
				},
				AzureMachine: &infrav1.AzureMachine{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine-90123456",
					},
					Spec: infrav1.AzureMachineSpec{
						OSDisk: infrav1.OSDisk{
							OSType: "Windows",
						},
					},
					Status: infrav1.AzureMachineStatus{},
				},
			},
			want:       "machine-9-23456",
			testLength: true,
		},
		{
			name: "Windows name with long MachineName and long cluster name",
			machineScope: MachineScope{
				ClusterScoper: &ClusterScope{
					Cluster: &clusterv1.Cluster{
						ObjectMeta: metav1.ObjectMeta{
							Name: "cluster8901234",
						},
					},
				},
				AzureMachine: &infrav1.AzureMachine{
					TypeMeta: metav1.TypeMeta{},
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine-90123456",
					},
					Spec: infrav1.AzureMachineSpec{
						OSDisk: infrav1.OSDisk{
							OSType: "Windows",
						},
					},
					Status: infrav1.AzureMachineStatus{},
				},
			},
			want:       "machine-9-23456",
			testLength: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.machineScope.Name()
			if got != tt.want {
				t.Errorf("MachineScope.Name() = %v, want %v", got, tt.want)
			}

			if tt.testLength && len(got) > 15 {
				t.Errorf("Length of MachineScope.Name() = %v, want less than %v", len(got), 15)
			}
		})
	}
}

func TestMachineScope_GetVMID(t *testing.T) {
	tests := []struct {
		name         string
		machineScope MachineScope
		want         string
	}{
		{
			name: "returns the vm name from provider ID",
			machineScope: MachineScope{
				AzureMachine: &infrav1.AzureMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "not-this-name",
					},
					Spec: infrav1.AzureMachineSpec{
						ProviderID: to.StringPtr("azure://compute/virtual-machines/machine-name"),
					},
				},
			},
			want: "machine-name",
		},
		{
			name: "returns empty if provider ID is invalid",
			machineScope: MachineScope{
				AzureMachine: &infrav1.AzureMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine-name",
					},
					Spec: infrav1.AzureMachineSpec{
						ProviderID: to.StringPtr("foo"),
					},
				},
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.machineScope.GetVMID()
			if got != tt.want {
				t.Errorf("MachineScope.GetVMID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMachineScope_ProviderID(t *testing.T) {
	tests := []struct {
		name         string
		machineScope MachineScope
		want         string
	}{
		{
			name: "returns the entire provider ID",
			machineScope: MachineScope{
				AzureMachine: &infrav1.AzureMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "not-this-name",
					},
					Spec: infrav1.AzureMachineSpec{
						ProviderID: to.StringPtr("azure://compute/virtual-machines/machine-name"),
					},
				},
			},
			want: "azure://compute/virtual-machines/machine-name",
		},
		{
			name: "returns empty if provider ID is invalid",
			machineScope: MachineScope{
				AzureMachine: &infrav1.AzureMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine-name",
					},
					Spec: infrav1.AzureMachineSpec{
						ProviderID: to.StringPtr("foo"),
					},
				},
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.machineScope.ProviderID()
			if got != tt.want {
				t.Errorf("MachineScope.ProviderID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMachineScope_PublicIPSpecs(t *testing.T) {
	tests := []struct {
		name         string
		machineScope MachineScope
		want         []azure.PublicIPSpec
	}{
		{
			name: "returns nil if AllocatePublicIP is false",
			machineScope: MachineScope{
				AzureMachine: &infrav1.AzureMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine-name",
					},
					Spec: infrav1.AzureMachineSpec{
						AllocatePublicIP: false,
					},
				},
			},
			want: nil,
		},
		{
			name: "appends to PublicIPSpec for node if AllocatePublicIP is true",
			machineScope: MachineScope{
				AzureMachine: &infrav1.AzureMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine-name",
					},
					Spec: infrav1.AzureMachineSpec{
						AllocatePublicIP: true,
					},
				},
			},
			want: []azure.PublicIPSpec{
				{
					Name: "pip-machine-name",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.machineScope.PublicIPSpecs(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PublicIPSpecs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMachineScope_InboundNatSpecs(t *testing.T) {
	tests := []struct {
		name         string
		machineScope MachineScope
		want         []azure.ResourceSpecGetter
	}{
		{
			name: "returns empty when infra is not control plane",
			machineScope: MachineScope{
				Machine: &clusterv1.Machine{},
				AzureMachine: &infrav1.AzureMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine-name",
					},
				},
			},
			want: []azure.ResourceSpecGetter{},
		},
		{
			name: "returns InboundNatSpec when infra is control plane",
			machineScope: MachineScope{
				Machine: &clusterv1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							clusterv1.MachineControlPlaneLabelName: "",
						},
					},
				},
				AzureMachine: &infrav1.AzureMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine-name",
					},
				},
				ClusterScoper: &ClusterScope{
					AzureClients: AzureClients{
						EnvironmentSettings: auth.EnvironmentSettings{
							Values: map[string]string{
								auth.SubscriptionID: "123",
							},
						},
					},
					AzureCluster: &infrav1.AzureCluster{
						Spec: infrav1.AzureClusterSpec{
							ResourceGroup:  "my-rg",
							SubscriptionID: "123",
							NetworkSpec: infrav1.NetworkSpec{
								APIServerLB: infrav1.LoadBalancerSpec{
									Name: "foo-loadbalancer",
									FrontendIPs: []infrav1.FrontendIP{
										{
											Name: "foo-frontend-ip",
										},
									},
								},
							},
						},
					},
				},
			},
			want: []azure.ResourceSpecGetter{
				&inboundnatrules.InboundNatSpec{
					Name:                      "machine-name",
					LoadBalancerName:          "foo-loadbalancer",
					ResourceGroup:             "my-rg",
					FrontendIPConfigurationID: to.StringPtr(azure.FrontendIPConfigID("123", "my-rg", "foo-loadbalancer", "foo-frontend-ip")),
					PortsInUse:                make(map[int32]struct{}),
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.machineScope.InboundNatSpecs(make(map[int32]struct{})); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("InboundNatSpecs() = %s, want %s", specArrayToString(got), specArrayToString(tt.want))
			}
		})
	}
}

func TestMachineScope_RoleAssignmentSpecs(t *testing.T) {
	tests := []struct {
		name         string
		machineScope MachineScope
		want         []azure.RoleAssignmentSpec
	}{
		{
			name: "returns empty if VM identity is system assigned",
			machineScope: MachineScope{
				Machine: &clusterv1.Machine{},
				AzureMachine: &infrav1.AzureMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine-name",
					},
				},
			},
			want: []azure.RoleAssignmentSpec{},
		},
		{
			name: "returns RoleAssignmentSpec if VM identity is not system assigned",
			machineScope: MachineScope{
				Machine: &clusterv1.Machine{},
				AzureMachine: &infrav1.AzureMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine-name",
					},
					Spec: infrav1.AzureMachineSpec{
						Identity:           infrav1.VMIdentitySystemAssigned,
						RoleAssignmentName: "azure-role-assignment-name",
					},
				},
			},
			want: []azure.RoleAssignmentSpec{
				{
					MachineName:  "machine-name",
					Name:         "azure-role-assignment-name",
					ResourceType: azure.VirtualMachine,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.machineScope.RoleAssignmentSpecs(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RoleAssignmentSpecs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMachineScope_VMExtensionSpecs(t *testing.T) {
	tests := []struct {
		name         string
		machineScope MachineScope
		want         []azure.ExtensionSpec
	}{
		{
			name: "If OS type is Linux and cloud is AzurePublicCloud, it returns ExtensionSpec",
			machineScope: MachineScope{
				Machine: &clusterv1.Machine{},
				AzureMachine: &infrav1.AzureMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine-name",
					},
					Spec: infrav1.AzureMachineSpec{
						OSDisk: infrav1.OSDisk{
							OSType: "Linux",
						},
					},
				},
				ClusterScoper: &ClusterScope{
					AzureClients: AzureClients{
						EnvironmentSettings: auth.EnvironmentSettings{
							Environment: autorestazure.Environment{
								Name: autorestazure.PublicCloud.Name,
							},
						},
					},
				},
			},
			want: []azure.ExtensionSpec{
				{
					Name:      "CAPZ.Linux.Bootstrapping",
					VMName:    "machine-name",
					Publisher: "Microsoft.Azure.ContainerUpstream",
					Version:   "1.0",
					ProtectedSettings: map[string]string{
						"commandToExecute": azure.LinuxBootstrapExtensionCommand,
					},
				},
			},
		},
		{
			name: "If OS type is Linux and cloud is not AzurePublicCloud, it returns empty",
			machineScope: MachineScope{
				Machine: &clusterv1.Machine{},
				AzureMachine: &infrav1.AzureMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine-name",
					},
					Spec: infrav1.AzureMachineSpec{
						OSDisk: infrav1.OSDisk{
							OSType: "Linux",
						},
					},
				},
				ClusterScoper: &ClusterScope{
					AzureClients: AzureClients{
						EnvironmentSettings: auth.EnvironmentSettings{
							Environment: autorestazure.Environment{
								Name: autorestazure.USGovernmentCloud.Name,
							},
						},
					},
				},
			},
			want: []azure.ExtensionSpec{},
		},
		{
			name: "If OS type is Windows and cloud is AzurePublicCloud, it returns ExtensionSpec",
			machineScope: MachineScope{
				Machine: &clusterv1.Machine{},
				AzureMachine: &infrav1.AzureMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine-name",
					},
					Spec: infrav1.AzureMachineSpec{
						OSDisk: infrav1.OSDisk{
							OSType: "Windows",
						},
					},
				},
				ClusterScoper: &ClusterScope{
					AzureClients: AzureClients{
						EnvironmentSettings: auth.EnvironmentSettings{
							Environment: autorestazure.Environment{
								Name: autorestazure.PublicCloud.Name,
							},
						},
					},
				},
			},
			want: []azure.ExtensionSpec{
				{
					Name:      "CAPZ.Windows.Bootstrapping",
					VMName:    "machine-name",
					Publisher: "Microsoft.Azure.ContainerUpstream",
					Version:   "1.0",
					ProtectedSettings: map[string]string{
						"commandToExecute": azure.WindowsBootstrapExtensionCommand,
					},
				},
			},
		},
		{
			name: "If OS type is Windows and cloud is not AzurePublicCloud, it returns empty",
			machineScope: MachineScope{
				Machine: &clusterv1.Machine{},
				AzureMachine: &infrav1.AzureMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine-name",
					},
					Spec: infrav1.AzureMachineSpec{
						OSDisk: infrav1.OSDisk{
							OSType: "Windows",
						},
					},
				},
				ClusterScoper: &ClusterScope{
					AzureClients: AzureClients{
						EnvironmentSettings: auth.EnvironmentSettings{
							Environment: autorestazure.Environment{
								Name: autorestazure.USGovernmentCloud.Name,
							},
						},
					},
				},
			},
			want: []azure.ExtensionSpec{},
		},
		{
			name: "If OS type is not Linux or Windows and cloud is AzurePublicCloud, it returns empty",
			machineScope: MachineScope{
				Machine: &clusterv1.Machine{},
				AzureMachine: &infrav1.AzureMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine-name",
					},
					Spec: infrav1.AzureMachineSpec{
						OSDisk: infrav1.OSDisk{
							OSType: "Other",
						},
					},
				},
				ClusterScoper: &ClusterScope{
					AzureClients: AzureClients{
						EnvironmentSettings: auth.EnvironmentSettings{
							Environment: autorestazure.Environment{
								Name: autorestazure.PublicCloud.Name,
							},
						},
					},
				},
			},
			want: []azure.ExtensionSpec{},
		},
		{
			name: "If OS type is not Windows or Linux and cloud is not AzurePublicCloud, it returns empty",
			machineScope: MachineScope{
				Machine: &clusterv1.Machine{},
				AzureMachine: &infrav1.AzureMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine-name",
					},
					Spec: infrav1.AzureMachineSpec{
						OSDisk: infrav1.OSDisk{
							OSType: "Other",
						},
					},
				},
				ClusterScoper: &ClusterScope{
					AzureClients: AzureClients{
						EnvironmentSettings: auth.EnvironmentSettings{
							Environment: autorestazure.Environment{
								Name: autorestazure.USGovernmentCloud.Name,
							},
						},
					},
				},
			},
			want: []azure.ExtensionSpec{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.machineScope.VMExtensionSpecs(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("VMExtensionSpecs() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMachineScope_Subnet(t *testing.T) {
	tests := []struct {
		name         string
		machineScope MachineScope
		want         infrav1.SubnetSpec
	}{
		{
			name: "returns empty if no subnet is found at cluster scope",
			machineScope: MachineScope{
				AzureMachine: &infrav1.AzureMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine-name",
					},
					Spec: infrav1.AzureMachineSpec{
						SubnetName: "machine-name-subnet",
					},
				},
				ClusterScoper: &ClusterScope{
					AzureCluster: &infrav1.AzureCluster{
						Spec: infrav1.AzureClusterSpec{
							NetworkSpec: infrav1.NetworkSpec{
								Subnets: []infrav1.SubnetSpec{},
							},
						},
					},
				},
			},
			want: infrav1.SubnetSpec{},
		},
		{
			name: "returns the machine subnet name if the same is present in the cluster scope",
			machineScope: MachineScope{
				AzureMachine: &infrav1.AzureMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine-name",
					},
					Spec: infrav1.AzureMachineSpec{
						SubnetName: "machine-name-subnet",
					},
				},
				ClusterScoper: &ClusterScope{
					AzureCluster: &infrav1.AzureCluster{
						Spec: infrav1.AzureClusterSpec{
							NetworkSpec: infrav1.NetworkSpec{
								Subnets: []infrav1.SubnetSpec{
									{
										Name: "machine-name-subnet",
									},
									{
										Name: "another-machine-name-subnet",
									},
								},
							},
						},
					},
				},
			},
			want: infrav1.SubnetSpec{
				Name: "machine-name-subnet",
			},
		},
		{
			name: "returns empty if machine subnet name is not present in the cluster scope",
			machineScope: MachineScope{
				AzureMachine: &infrav1.AzureMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine-name",
					},
					Spec: infrav1.AzureMachineSpec{
						SubnetName: "machine-name-subnet",
					},
				},
				ClusterScoper: &ClusterScope{
					AzureCluster: &infrav1.AzureCluster{
						Spec: infrav1.AzureClusterSpec{
							NetworkSpec: infrav1.NetworkSpec{},
						},
					},
				},
			},
			want: infrav1.SubnetSpec{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.machineScope.Subnet(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Subnet() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMachineScope_AvailabilityZone(t *testing.T) {
	tests := []struct {
		name         string
		machineScope MachineScope
		want         string
	}{
		{
			name: "returns empty if no failure domain is present",
			machineScope: MachineScope{
				Machine: &clusterv1.Machine{
					Spec: clusterv1.MachineSpec{},
				},
				AzureMachine: &infrav1.AzureMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine-name",
					},
					Spec: infrav1.AzureMachineSpec{},
				},
			},
			want: "",
		},
		{
			name: "returns failure domain from the machine spec",
			machineScope: MachineScope{
				Machine: &clusterv1.Machine{
					Spec: clusterv1.MachineSpec{
						FailureDomain: pointer.String("dummy-failure-domain-from-machine-spec"),
					},
				},
				AzureMachine: &infrav1.AzureMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine-name",
					},
					Spec: infrav1.AzureMachineSpec{
						FailureDomain: pointer.String("dummy-failure-domain-from-azuremachine-spec"),
					},
				},
			},
			want: "dummy-failure-domain-from-machine-spec",
		},
		{
			name: "returns failure domain from the azuremachine spec",
			machineScope: MachineScope{
				Machine: &clusterv1.Machine{
					Spec: clusterv1.MachineSpec{},
				},
				AzureMachine: &infrav1.AzureMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine-name",
					},
					Spec: infrav1.AzureMachineSpec{
						FailureDomain: pointer.String("dummy-failure-domain-from-azuremachine-spec"),
					},
				},
			},
			want: "dummy-failure-domain-from-azuremachine-spec",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.machineScope.AvailabilityZone(); got != tt.want {
				t.Errorf("AvailabilityZone() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMachineScope_Namespace(t *testing.T) {
	tests := []struct {
		name         string
		machineScope MachineScope
		want         string
	}{
		{
			name: "returns azure machine namespace",
			machineScope: MachineScope{
				AzureMachine: &infrav1.AzureMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "machine-name",
						Namespace: "foo",
					},
				},
			},
			want: "foo",
		},
		{
			name: "returns azure machine namespace as empty if namespace is no specified",
			machineScope: MachineScope{
				AzureMachine: &infrav1.AzureMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine-name",
					},
				},
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.machineScope.Namespace(); got != tt.want {
				t.Errorf("Namespace() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMachineScope_IsControlPlane(t *testing.T) {
	tests := []struct {
		name         string
		machineScope MachineScope
		want         bool
	}{
		{
			name: "returns false when machine is not control plane",
			machineScope: MachineScope{
				Machine: &clusterv1.Machine{},
				AzureMachine: &infrav1.AzureMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine-name",
					},
				},
			},
			want: false,
		},
		{
			name: "returns true when machine is control plane",
			machineScope: MachineScope{
				Machine: &clusterv1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							clusterv1.MachineControlPlaneLabelName: "",
						},
					},
				},
				AzureMachine: &infrav1.AzureMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine-name",
					},
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.machineScope.IsControlPlane(); got != tt.want {
				t.Errorf("IsControlPlane() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMachineScope_Role(t *testing.T) {
	tests := []struct {
		name         string
		machineScope MachineScope
		want         string
	}{
		{
			name: "returns node when machine is worker",
			machineScope: MachineScope{
				Machine: &clusterv1.Machine{},
				AzureMachine: &infrav1.AzureMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine-name",
					},
				},
			},
			want: infrav1.Node,
		},
		{
			name: "returns control-plane when machine is control plane",
			machineScope: MachineScope{
				Machine: &clusterv1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							clusterv1.MachineControlPlaneLabelName: "",
						},
					},
				},
				AzureMachine: &infrav1.AzureMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine-name",
					},
				},
			},
			want: infrav1.ControlPlane,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.machineScope.Role(); got != tt.want {
				t.Errorf("Role() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMachineScope_AvailabilitySet(t *testing.T) {
	tests := []struct {
		name                         string
		machineScope                 MachineScope
		wantAvailabilitySetName      string
		wantAvailabilitySetExistence bool
	}{
		{
			name: "returns empty and false if availability set is not enabled",
			machineScope: MachineScope{
				ClusterScoper: &ClusterScope{
					Cluster: &clusterv1.Cluster{
						ObjectMeta: metav1.ObjectMeta{
							Name: "cluster",
						},
					},
					AzureCluster: &infrav1.AzureCluster{
						Status: infrav1.AzureClusterStatus{
							FailureDomains: clusterv1.FailureDomains{
								"foo-failure-domain": clusterv1.FailureDomainSpec{},
							},
						},
					},
				},
				Machine: &clusterv1.Machine{},
			},
			wantAvailabilitySetName:      "",
			wantAvailabilitySetExistence: false,
		},
		{
			name: "returns AvailabilitySet name and true if availability set is enabled and machine is control plane",
			machineScope: MachineScope{

				ClusterScoper: &ClusterScope{
					Cluster: &clusterv1.Cluster{
						ObjectMeta: metav1.ObjectMeta{
							Name: "cluster",
						},
					},
					AzureCluster: &infrav1.AzureCluster{
						Status: infrav1.AzureClusterStatus{},
					},
				},
				Machine: &clusterv1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							clusterv1.MachineControlPlaneLabelName: "",
						},
					},
				},
			},
			wantAvailabilitySetName:      "cluster_control-plane-as",
			wantAvailabilitySetExistence: true,
		},
		{
			name: "returns AvailabilitySet name and true if AvailabilitySet is enabled for worker machine which is part of machine deployment",
			machineScope: MachineScope{

				ClusterScoper: &ClusterScope{
					Cluster: &clusterv1.Cluster{
						ObjectMeta: metav1.ObjectMeta{
							Name: "cluster",
						},
					},
					AzureCluster: &infrav1.AzureCluster{
						Status: infrav1.AzureClusterStatus{},
					},
				},
				Machine: &clusterv1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							clusterv1.MachineDeploymentLabelName: "foo-machine-deployment",
						},
					},
				},
			},
			wantAvailabilitySetName:      "cluster_foo-machine-deployment-as",
			wantAvailabilitySetExistence: true,
		},
		{
			name: "returns AvailabilitySet name and true if AvailabilitySet is enabled for worker machine which is part of machine set",
			machineScope: MachineScope{

				ClusterScoper: &ClusterScope{
					Cluster: &clusterv1.Cluster{
						ObjectMeta: metav1.ObjectMeta{
							Name: "cluster",
						},
					},
					AzureCluster: &infrav1.AzureCluster{
						Status: infrav1.AzureClusterStatus{},
					},
				},
				Machine: &clusterv1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							clusterv1.MachineSetLabelName: "foo-machine-set",
						},
					},
				},
			},
			wantAvailabilitySetName:      "cluster_foo-machine-set-as",
			wantAvailabilitySetExistence: true,
		},
		{
			name: "returns AvailabilitySet name and true if AvailabilitySet is enabled for worker machine and machine deployment name takes precedence over machine set name",
			machineScope: MachineScope{

				ClusterScoper: &ClusterScope{
					Cluster: &clusterv1.Cluster{
						ObjectMeta: metav1.ObjectMeta{
							Name: "cluster",
						},
					},
					AzureCluster: &infrav1.AzureCluster{
						Status: infrav1.AzureClusterStatus{},
					},
				},
				Machine: &clusterv1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							clusterv1.MachineDeploymentLabelName: "foo-machine-deployment",
							clusterv1.MachineSetLabelName:        "foo-machine-set",
						},
					},
				},
			},
			wantAvailabilitySetName:      "cluster_foo-machine-deployment-as",
			wantAvailabilitySetExistence: true,
		},
		{
			name: "returns empty and false if AvailabilitySet is enabled but worker machine is not part of machine deployment or machine set",
			machineScope: MachineScope{

				ClusterScoper: &ClusterScope{
					Cluster: &clusterv1.Cluster{
						ObjectMeta: metav1.ObjectMeta{
							Name: "cluster",
						},
					},
					AzureCluster: &infrav1.AzureCluster{
						Status: infrav1.AzureClusterStatus{},
					},
				},
				Machine: &clusterv1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{},
					},
				},
			},
			wantAvailabilitySetName:      "",
			wantAvailabilitySetExistence: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotAvailabilitySetName, gotAvailabilitySetExistence := tt.machineScope.AvailabilitySet()
			if gotAvailabilitySetName != tt.wantAvailabilitySetName {
				t.Errorf("AvailabilitySet() gotAvailabilitySetName = %v, wantAvailabilitySetName %v", gotAvailabilitySetName, tt.wantAvailabilitySetName)
			}
			if gotAvailabilitySetExistence != tt.wantAvailabilitySetExistence {
				t.Errorf("AvailabilitySet() gotAvailabilitySetExistence = %v, wantAvailabilitySetExistence %v", gotAvailabilitySetExistence, tt.wantAvailabilitySetExistence)
			}
		})
	}
}

func TestMachineScope_VMState(t *testing.T) {
	tests := []struct {
		name         string
		machineScope MachineScope
		want         infrav1.ProvisioningState
	}{
		{
			name: "returns the VMState if present in AzureMachine status",
			machineScope: MachineScope{
				AzureMachine: &infrav1.AzureMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine-name",
					},
					Status: infrav1.AzureMachineStatus{
						VMState: func() *infrav1.ProvisioningState {
							provisioningState := infrav1.Creating
							return &provisioningState
						}(),
					},
				},
			},
			want: infrav1.Creating,
		},
		{
			name: "returns empty if VMState is not present in AzureMachine status",
			machineScope: MachineScope{
				AzureMachine: &infrav1.AzureMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine-name",
					},
					Status: infrav1.AzureMachineStatus{},
				},
			},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.machineScope.VMState(); got != tt.want {
				t.Errorf("VMState() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMachineScope_GetVMImage(t *testing.T) {
	tests := []struct {
		name         string
		machineScope MachineScope
		want         *infrav1.Image
		wantErr      bool
	}{
		{
			name: "returns AzureMachine image is found if present in the AzureMachine spec",
			machineScope: MachineScope{
				AzureMachine: &infrav1.AzureMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine-name",
					},
					Spec: infrav1.AzureMachineSpec{
						Image: &infrav1.Image{
							ID: pointer.StringPtr("1"),
						},
					},
				},
			},
			want: &infrav1.Image{
				ID: pointer.StringPtr("1"),
			},
			wantErr: false,
		},
		{
			name: "if no image is specified and os specified is windows with version below 1.22, returns windows dockershim image",
			machineScope: MachineScope{
				Machine: &clusterv1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine-name",
					},
					Spec: clusterv1.MachineSpec{
						Version: pointer.String("1.20.1"),
					},
				},
				AzureMachine: &infrav1.AzureMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine-name",
					},
					Spec: infrav1.AzureMachineSpec{
						OSDisk: infrav1.OSDisk{
							OSType: azure.WindowsOS,
						},
					},
				},
			},
			want: func() *infrav1.Image {
				image, _ := azure.GetDefaultWindowsImage("1.20.1", "dockershim")
				return image
			}(),
			wantErr: false,
		},
		{
			name: "if no image is specified and os specified is windows with version is 1.22+ with no annotation, returns windows containerd image",
			machineScope: MachineScope{
				Machine: &clusterv1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine-name",
					},
					Spec: clusterv1.MachineSpec{
						Version: pointer.String("1.22.1"),
					},
				},
				AzureMachine: &infrav1.AzureMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine-name",
					},
					Spec: infrav1.AzureMachineSpec{
						OSDisk: infrav1.OSDisk{
							OSType: azure.WindowsOS,
						},
					},
				},
			},
			want: func() *infrav1.Image {
				image, _ := azure.GetDefaultWindowsImage("1.22.1", "containerd")
				return image
			}(),
			wantErr: false,
		},
		{
			name: "if no image is specified and os specified is windows with version is 1.22+ with annotation dockershim, returns windows dockershim image",
			machineScope: MachineScope{
				Machine: &clusterv1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine-name",
					},
					Spec: clusterv1.MachineSpec{
						Version: pointer.String("1.22.1"),
					},
				},
				AzureMachine: &infrav1.AzureMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine-name",
						Annotations: map[string]string{
							"runtime": "dockershim",
						},
					},
					Spec: infrav1.AzureMachineSpec{
						OSDisk: infrav1.OSDisk{
							OSType: azure.WindowsOS,
						},
					},
				},
			},
			want: func() *infrav1.Image {
				image, _ := azure.GetDefaultWindowsImage("1.22.1", "dockershim")
				return image
			}(),
			wantErr: false,
		},
		{
			name: "if no image is specified and os specified is windows with version is less and 1.22 with annotation dockershim, returns windows dockershim image",
			machineScope: MachineScope{
				Machine: &clusterv1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine-name",
					},
					Spec: clusterv1.MachineSpec{
						Version: pointer.String("1.21.1"),
					},
				},
				AzureMachine: &infrav1.AzureMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine-name",
						Annotations: map[string]string{
							"runtime": "dockershim",
						},
					},
					Spec: infrav1.AzureMachineSpec{
						OSDisk: infrav1.OSDisk{
							OSType: azure.WindowsOS,
						},
					},
				},
			},
			want: func() *infrav1.Image {
				image, _ := azure.GetDefaultWindowsImage("1.21.1", "dockershim")
				return image
			}(),
			wantErr: false,
		},
		{
			name: "if no image is specified and os specified is windows with version is less and 1.22 with annotation containerd, returns error",
			machineScope: MachineScope{
				Machine: &clusterv1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine-name",
					},
					Spec: clusterv1.MachineSpec{
						Version: pointer.String("1.21.1"),
					},
				},
				AzureMachine: &infrav1.AzureMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine-name",
						Annotations: map[string]string{
							"runtime": "containerd",
						},
					},
					Spec: infrav1.AzureMachineSpec{
						OSDisk: infrav1.OSDisk{
							OSType: azure.WindowsOS,
						},
					},
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "if no image and OS is specified, returns linux image",
			machineScope: MachineScope{
				Machine: &clusterv1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine-name",
					},
					Spec: clusterv1.MachineSpec{
						Version: pointer.String("1.20.1"),
					},
				},
				AzureMachine: &infrav1.AzureMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine-name",
					},
				},
			},
			want: func() *infrav1.Image {
				image, _ := azure.GetDefaultUbuntuImage("1.20.1")
				return image
			}(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotImage, err := tt.machineScope.GetVMImage(context.TODO())
			gotError := false
			if err != nil {
				gotError = true
			}

			if gotError != tt.wantErr {
				t.Errorf("GetVMImage(), gotError = %v, wantError %v", gotError, tt.wantErr)
			}
			if !reflect.DeepEqual(gotImage, tt.want) {
				t.Errorf("GetVMImage(), gotImage = %v, wantImage %v", gotImage, tt.want)
			}
		})
	}
}

func TestMachineScope_NICSpecs(t *testing.T) {
	tests := []struct {
		name         string
		machineScope MachineScope
		want         []azure.NICSpec
	}{
		{
			name: "Node Machine with no NAT gateway and no public IP address",
			machineScope: MachineScope{
				ClusterScoper: &ClusterScope{
					Cluster: &clusterv1.Cluster{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "cluster",
							Namespace: "default",
						},
					},
					AzureCluster: &infrav1.AzureCluster{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "cluster",
							Namespace: "default",
							OwnerReferences: []metav1.OwnerReference{
								{
									APIVersion: "cluster.x-k8s.io/v1beta1",
									Kind:       "Cluster",
									Name:       "cluster",
								},
							},
						},
						Spec: infrav1.AzureClusterSpec{
							NetworkSpec: infrav1.NetworkSpec{
								Vnet: infrav1.VnetSpec{
									Name:          "vnet1",
									ResourceGroup: "rg1",
								},
								Subnets: []infrav1.SubnetSpec{
									{
										Role: infrav1.SubnetNode,
										Name: "subnet1",
									},
								},
								NodeOutboundLB: &infrav1.LoadBalancerSpec{
									Name: "outbound-lb",
								},
							},
						},
					},
				},
				AzureMachine: &infrav1.AzureMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine",
					},
					Spec: infrav1.AzureMachineSpec{
						ProviderID: to.StringPtr("azure://compute/virtual-machines/machine-name"),
						SubnetName: "subnet1",
					},
				},
				Machine: &clusterv1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "machine",
						Labels: map[string]string{
							//clusterv1.MachineControlPlaneLabelName: "true",
						},
					},
				},
			},
			want: []azure.NICSpec{
				{
					Name:                      "machine-name-nic",
					MachineName:               "machine-name",
					SubnetName:                "subnet1",
					VNetName:                  "vnet1",
					VNetResourceGroup:         "rg1",
					PublicLBName:              "outbound-lb",
					PublicLBAddressPoolName:   "outbound-lb-outboundBackendPool",
					PublicLBNATRuleName:       "",
					InternalLBName:            "",
					InternalLBAddressPoolName: "",
					PublicIPName:              "",
					VMSize:                    "",
					AcceleratedNetworking:     nil,
					IPv6Enabled:               false,
					EnableIPForwarding:        false,
				},
			},
		},
		{
			name: "Node Machine with NAT gateway",
			machineScope: MachineScope{
				ClusterScoper: &ClusterScope{
					Cluster: &clusterv1.Cluster{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "cluster",
							Namespace: "default",
						},
					},
					AzureCluster: &infrav1.AzureCluster{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "cluster",
							Namespace: "default",
							OwnerReferences: []metav1.OwnerReference{
								{
									APIVersion: "cluster.x-k8s.io/v1beta1",
									Kind:       "Cluster",
									Name:       "cluster",
								},
							},
						},
						Spec: infrav1.AzureClusterSpec{
							NetworkSpec: infrav1.NetworkSpec{
								Vnet: infrav1.VnetSpec{
									Name:          "vnet1",
									ResourceGroup: "rg1",
								},
								Subnets: []infrav1.SubnetSpec{
									{
										Role: infrav1.SubnetNode,
										Name: "subnet1",
										NatGateway: infrav1.NatGateway{
											Name: "natgw",
										},
									},
								},
								NodeOutboundLB: &infrav1.LoadBalancerSpec{
									Name: "outbound-lb",
								},
							},
						},
					},
				},
				AzureMachine: &infrav1.AzureMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine",
					},
					Spec: infrav1.AzureMachineSpec{
						ProviderID: to.StringPtr("azure://compute/virtual-machines/machine-name"),
						SubnetName: "subnet1",
					},
				},
				Machine: &clusterv1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "machine",
						Labels: map[string]string{
							//clusterv1.MachineControlPlaneLabelName: "true",
						},
					},
				},
			},
			want: []azure.NICSpec{
				{
					Name:                      "machine-name-nic",
					MachineName:               "machine-name",
					SubnetName:                "subnet1",
					VNetName:                  "vnet1",
					VNetResourceGroup:         "rg1",
					PublicLBName:              "",
					PublicLBAddressPoolName:   "",
					PublicLBNATRuleName:       "",
					InternalLBName:            "",
					InternalLBAddressPoolName: "",
					PublicIPName:              "",
					VMSize:                    "",
					AcceleratedNetworking:     nil,
					IPv6Enabled:               false,
					EnableIPForwarding:        false,
				},
			},
		},
		{
			name: "Node Machine with public IP address",
			machineScope: MachineScope{
				ClusterScoper: &ClusterScope{
					Cluster: &clusterv1.Cluster{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "cluster",
							Namespace: "default",
						},
					},
					AzureCluster: &infrav1.AzureCluster{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "cluster",
							Namespace: "default",
							OwnerReferences: []metav1.OwnerReference{
								{
									APIVersion: "cluster.x-k8s.io/v1beta1",
									Kind:       "Cluster",
									Name:       "cluster",
								},
							},
						},
						Spec: infrav1.AzureClusterSpec{
							NetworkSpec: infrav1.NetworkSpec{
								Vnet: infrav1.VnetSpec{
									Name:          "vnet1",
									ResourceGroup: "rg1",
								},
								Subnets: []infrav1.SubnetSpec{
									{
										Role: infrav1.SubnetNode,
										Name: "subnet1",
									},
								},
								NodeOutboundLB: &infrav1.LoadBalancerSpec{
									Name: "outbound-lb",
								},
							},
						},
					},
				},
				AzureMachine: &infrav1.AzureMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine",
					},
					Spec: infrav1.AzureMachineSpec{
						ProviderID:       to.StringPtr("azure://compute/virtual-machines/machine-name"),
						SubnetName:       "subnet1",
						AllocatePublicIP: true,
					},
				},
				Machine: &clusterv1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Name:   "machine",
						Labels: map[string]string{
							//clusterv1.MachineControlPlaneLabelName: "true",
						},
					},
				},
			},
			want: []azure.NICSpec{
				{
					Name:                      "machine-name-nic",
					MachineName:               "machine-name",
					SubnetName:                "subnet1",
					VNetName:                  "vnet1",
					VNetResourceGroup:         "rg1",
					PublicLBName:              "",
					PublicLBAddressPoolName:   "",
					PublicLBNATRuleName:       "",
					InternalLBName:            "",
					InternalLBAddressPoolName: "",
					PublicIPName:              "pip-machine-name",
					VMSize:                    "",
					AcceleratedNetworking:     nil,
					IPv6Enabled:               false,
					EnableIPForwarding:        false,
				},
			},
		},
		{
			name: "Control Plane Machine with private LB",
			machineScope: MachineScope{
				ClusterScoper: &ClusterScope{
					Cluster: &clusterv1.Cluster{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "cluster",
							Namespace: "default",
						},
					},
					AzureCluster: &infrav1.AzureCluster{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "cluster",
							Namespace: "default",
							OwnerReferences: []metav1.OwnerReference{
								{
									APIVersion: "cluster.x-k8s.io/v1beta1",
									Kind:       "Cluster",
									Name:       "cluster",
								},
							},
						},
						Spec: infrav1.AzureClusterSpec{
							NetworkSpec: infrav1.NetworkSpec{
								Vnet: infrav1.VnetSpec{
									Name:          "vnet1",
									ResourceGroup: "rg1",
								},
								Subnets: []infrav1.SubnetSpec{
									{
										Role: infrav1.SubnetNode,
										Name: "subnet1",
									},
								},
								APIServerLB: infrav1.LoadBalancerSpec{
									Name: "api-lb",
									Type: infrav1.Internal,
								},
								NodeOutboundLB: &infrav1.LoadBalancerSpec{
									Name: "outbound-lb",
								},
							},
						},
					},
				},
				AzureMachine: &infrav1.AzureMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine",
					},
					Spec: infrav1.AzureMachineSpec{
						ProviderID: to.StringPtr("azure://compute/virtual-machines/machine-name"),
						SubnetName: "subnet1",
					},
				},
				Machine: &clusterv1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine",
						Labels: map[string]string{
							clusterv1.MachineControlPlaneLabelName: "true",
						},
					},
				},
			},
			want: []azure.NICSpec{
				{
					Name:                      "machine-name-nic",
					MachineName:               "machine-name",
					SubnetName:                "subnet1",
					VNetName:                  "vnet1",
					VNetResourceGroup:         "rg1",
					PublicLBName:              "",
					PublicLBAddressPoolName:   "",
					PublicLBNATRuleName:       "",
					InternalLBName:            "api-lb",
					InternalLBAddressPoolName: "api-lb-backendPool",
					PublicIPName:              "",
					VMSize:                    "",
					AcceleratedNetworking:     nil,
					IPv6Enabled:               false,
					EnableIPForwarding:        false,
				},
			},
		},
		{
			name: "Control Plane Machine with public LB",
			machineScope: MachineScope{
				ClusterScoper: &ClusterScope{
					Cluster: &clusterv1.Cluster{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "cluster",
							Namespace: "default",
						},
					},
					AzureCluster: &infrav1.AzureCluster{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "cluster",
							Namespace: "default",
							OwnerReferences: []metav1.OwnerReference{
								{
									APIVersion: "cluster.x-k8s.io/v1beta1",
									Kind:       "Cluster",
									Name:       "cluster",
								},
							},
						},
						Spec: infrav1.AzureClusterSpec{
							NetworkSpec: infrav1.NetworkSpec{
								Vnet: infrav1.VnetSpec{
									Name:          "vnet1",
									ResourceGroup: "rg1",
								},
								Subnets: []infrav1.SubnetSpec{
									{
										Role: infrav1.SubnetNode,
										Name: "subnet1",
									},
								},
								APIServerLB: infrav1.LoadBalancerSpec{
									Name: "api-lb",
								},
								NodeOutboundLB: &infrav1.LoadBalancerSpec{
									Name: "outbound-lb",
								},
							},
						},
					},
				},
				AzureMachine: &infrav1.AzureMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine",
					},
					Spec: infrav1.AzureMachineSpec{
						ProviderID: to.StringPtr("azure://compute/virtual-machines/machine-name"),
						SubnetName: "subnet1",
					},
				},
				Machine: &clusterv1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine",
						Labels: map[string]string{
							clusterv1.MachineControlPlaneLabelName: "true",
						},
					},
				},
			},
			want: []azure.NICSpec{
				{
					Name:                      "machine-name-nic",
					MachineName:               "machine-name",
					SubnetName:                "subnet1",
					VNetName:                  "vnet1",
					VNetResourceGroup:         "rg1",
					PublicLBName:              "api-lb",
					PublicLBAddressPoolName:   "api-lb-backendPool",
					PublicLBNATRuleName:       "machine-name",
					InternalLBName:            "",
					InternalLBAddressPoolName: "",
					PublicIPName:              "",
					VMSize:                    "",
					AcceleratedNetworking:     nil,
					IPv6Enabled:               false,
					EnableIPForwarding:        false,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotNicSpecs := tt.machineScope.NICSpecs()
			if !reflect.DeepEqual(gotNicSpecs, tt.want) {
				t.Errorf("NICSpecs(), gotNicSpecs = %v, want %v", gotNicSpecs, tt.want)
			}
		})
	}
}

func TestDiskSpecs(t *testing.T) {
	testcases := []struct {
		name         string
		machineScope MachineScope
		want         []azure.ResourceSpecGetter
	}{
		{
			name: "only os disk",
			machineScope: MachineScope{
				ClusterScoper: &ClusterScope{
					Cluster: &clusterv1.Cluster{
						ObjectMeta: metav1.ObjectMeta{
							Name: "cluster",
						},
					},
					AzureCluster: &infrav1.AzureCluster{
						ObjectMeta: metav1.ObjectMeta{
							Name: "cluster",
						},
						Spec: infrav1.AzureClusterSpec{
							ResourceGroup: "my-rg",
						},
					},
				},
				AzureMachine: &infrav1.AzureMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-azure-machine",
					},
					Spec: infrav1.AzureMachineSpec{
						OSDisk: infrav1.OSDisk{
							DiskSizeGB: to.Int32Ptr(30),
							OSType:     "Linux",
						},
					},
				},
				Machine: &clusterv1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine",
					},
				},
			},
			want: []azure.ResourceSpecGetter{
				&disks.DiskSpec{
					Name:          "my-azure-machine_OSDisk",
					ResourceGroup: "my-rg",
				},
			},
		},
		{
			name: "os and data disks",
			machineScope: MachineScope{
				ClusterScoper: &ClusterScope{
					Cluster: &clusterv1.Cluster{
						ObjectMeta: metav1.ObjectMeta{
							Name: "cluster",
						},
					},
					AzureCluster: &infrav1.AzureCluster{
						ObjectMeta: metav1.ObjectMeta{
							Name: "cluster",
						},
						Spec: infrav1.AzureClusterSpec{
							ResourceGroup: "my-rg",
						},
					},
				},
				AzureMachine: &infrav1.AzureMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-azure-machine",
					},
					Spec: infrav1.AzureMachineSpec{
						OSDisk: infrav1.OSDisk{
							DiskSizeGB: to.Int32Ptr(30),
							OSType:     "Linux",
						},
						DataDisks: []infrav1.DataDisk{
							{
								NameSuffix: "etcddisk",
							},
						},
					},
				},
				Machine: &clusterv1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine",
					},
				},
			},
			want: []azure.ResourceSpecGetter{
				&disks.DiskSpec{
					Name:          "my-azure-machine_OSDisk",
					ResourceGroup: "my-rg",
				},
				&disks.DiskSpec{
					Name:          "my-azure-machine_etcddisk",
					ResourceGroup: "my-rg",
				},
			},
		}, {
			name: "os and multiple data disks",
			machineScope: MachineScope{
				ClusterScoper: &ClusterScope{
					Cluster: &clusterv1.Cluster{
						ObjectMeta: metav1.ObjectMeta{
							Name: "cluster",
						},
					},
					AzureCluster: &infrav1.AzureCluster{
						ObjectMeta: metav1.ObjectMeta{
							Name: "cluster",
						},
						Spec: infrav1.AzureClusterSpec{
							ResourceGroup: "my-rg",
						},
					},
				},
				AzureMachine: &infrav1.AzureMachine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "my-azure-machine",
					},
					Spec: infrav1.AzureMachineSpec{
						OSDisk: infrav1.OSDisk{
							DiskSizeGB: to.Int32Ptr(30),
							OSType:     "Linux",
						},
						DataDisks: []infrav1.DataDisk{
							{
								NameSuffix: "etcddisk",
							},
							{
								NameSuffix: "otherdisk",
							},
						},
					},
				},
				Machine: &clusterv1.Machine{
					ObjectMeta: metav1.ObjectMeta{
						Name: "machine",
					},
				},
			},
			want: []azure.ResourceSpecGetter{
				&disks.DiskSpec{
					Name:          "my-azure-machine_OSDisk",
					ResourceGroup: "my-rg",
				},
				&disks.DiskSpec{
					Name:          "my-azure-machine_etcddisk",
					ResourceGroup: "my-rg",
				},
				&disks.DiskSpec{
					Name:          "my-azure-machine_otherdisk",
					ResourceGroup: "my-rg",
				},
			},
		},
	}

	for _, tt := range testcases {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			t.Parallel()
			result := tt.machineScope.DiskSpecs()
			g.Expect(result).To(BeEquivalentTo(tt.want))
		})
	}
}
