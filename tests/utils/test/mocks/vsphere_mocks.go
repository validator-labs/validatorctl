package mocks

import (
	"context"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vapi/tags"
	"github.com/vmware/govmomi/vim25/mo"
	"strings"

	vsphere "github.com/validator-labs/validator-plugin-vsphere/clouddriver"
	msg "github.com/validator-labs/validatorctl/pkg/utils/extra"
)

type VsphereDriver interface {
	GetVSphereVMFolders(ctx context.Context, datacenter string) ([]string, *msg.MsgError)
	GetVSphereDatacenters(ctx context.Context) ([]string, *msg.MsgError)
	GetVSphereClusters(ctx context.Context, datacenter string) ([]string, *msg.MsgError)
	GetVSphereHostSystems(ctx context.Context, datacenter, cluster string) ([]vsphere.VSphereHostSystem, *msg.MsgError)
	IsValidVSphereCredentials(ctx context.Context) (bool, *msg.MsgError)
	ValidateVsphereVersion(constraint string) error
	GetHostClusterMapping(ctx context.Context) (map[string]string, *msg.MsgError)
	GetVSphereVms(ctx context.Context, dcName string) ([]vsphere.VSphereVM, *msg.MsgError)
	GetResourcePools(ctx context.Context, datacenter string, cluster string) ([]*object.ResourcePool, *msg.MsgError)
	GetVapps(ctx context.Context) ([]mo.VirtualApp, *msg.MsgError)
	GetResourceTags(ctx context.Context, resourceType string) (map[string]tags.AttachedTags, *msg.MsgError)
}

type MockVsphereDriver struct {
	Datacenters        []string
	Clusters           []string
	VMs                []vsphere.VSphereVM
	VMFolders          []string
	HostSystems        map[string][]vsphere.VSphereHostSystem
	VApps              []mo.VirtualApp
	ResourcePools      []*object.ResourcePool
	HostClusterMapping map[string]string
	ResourceTags       map[string]tags.AttachedTags
}

func (d MockVsphereDriver) GetVSphereVMFolders(ctx context.Context, datacenter string) ([]string, *msg.MsgError) {
	return d.VMFolders, nil
}

func (d MockVsphereDriver) GetVSphereDatacenters(ctx context.Context) ([]string, *msg.MsgError) {
	return d.Datacenters, nil
}

func (d MockVsphereDriver) GetVSphereClusters(ctx context.Context, datacenter string) ([]string, *msg.MsgError) {
	return d.Clusters, nil
}

func (d MockVsphereDriver) GetVSphereHostSystems(ctx context.Context, datacenter, cluster string) ([]vsphere.VSphereHostSystem, *msg.MsgError) {
	return d.HostSystems[concat(datacenter, cluster)], nil
}

func (d MockVsphereDriver) IsValidVSphereCredentials(ctx context.Context) (bool, *msg.MsgError) {
	return true, nil
}

func (d MockVsphereDriver) ValidateVsphereVersion(constraint string) error {
	return nil
}

func (d MockVsphereDriver) GetHostClusterMapping(ctx context.Context) (map[string]string, *msg.MsgError) {
	return d.HostClusterMapping, nil
}

func (d MockVsphereDriver) GetVSphereVms(ctx context.Context, dcName string) ([]vsphere.VSphereVM, *msg.MsgError) {
	return d.VMs, nil
}

func (d MockVsphereDriver) GetResourcePools(ctx context.Context, datacenter string, cluster string) ([]*object.ResourcePool, *msg.MsgError) {
	return d.ResourcePools, nil
}

func (d MockVsphereDriver) GetVapps(ctx context.Context) ([]mo.VirtualApp, *msg.MsgError) {
	return d.VApps, nil
}

func (d MockVsphereDriver) GetResourceTags(ctx context.Context, resourceType string) (map[string]tags.AttachedTags, *msg.MsgError) {
	return d.ResourceTags, nil
}

func concat(ss ...string) string {
	return strings.Join(ss, "_")
}
