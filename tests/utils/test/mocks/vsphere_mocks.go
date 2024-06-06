package mocks

import (
	"context"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vapi/tags"
	"github.com/vmware/govmomi/vim25/mo"
	"strings"

	"github.com/validator-labs/validator-plugin-vsphere/pkg/vsphere"
)

type VsphereDriver interface {
	GetVSphereVMFolders(ctx context.Context, datacenter string) ([]string, error)
	GetVSphereDatacenters(ctx context.Context) ([]string, error)
	GetVSphereClusters(ctx context.Context, datacenter string) ([]string, error)
	GetVSphereHostSystems(ctx context.Context, datacenter, cluster string) ([]vsphere.VSphereHostSystem, error)
	IsValidVSphereCredentials(ctx context.Context) (bool, error)
	ValidateVsphereVersion(constraint string) error
	GetHostClusterMapping(ctx context.Context) (map[string]string, error)
	GetVSphereVms(ctx context.Context, dcName string) ([]vsphere.VSphereVM, error)
	GetResourcePools(ctx context.Context, datacenter string, cluster string) ([]*object.ResourcePool, error)
	GetVapps(ctx context.Context) ([]mo.VirtualApp, error)
	GetResourceTags(ctx context.Context, resourceType string) (map[string]tags.AttachedTags, error)
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

func (d MockVsphereDriver) GetVSphereVMFolders(ctx context.Context, datacenter string) ([]string, error) {
	return d.VMFolders, nil
}

func (d MockVsphereDriver) GetVSphereDatacenters(ctx context.Context) ([]string, error) {
	return d.Datacenters, nil
}

func (d MockVsphereDriver) GetVSphereClusters(ctx context.Context, datacenter string) ([]string, error) {
	return d.Clusters, nil
}

func (d MockVsphereDriver) GetVSphereHostSystems(ctx context.Context, datacenter, cluster string) ([]vsphere.VSphereHostSystem, error) {
	return d.HostSystems[concat(datacenter, cluster)], nil
}

func (d MockVsphereDriver) IsValidVSphereCredentials(ctx context.Context) (bool, error) {
	return true, nil
}

func (d MockVsphereDriver) ValidateVsphereVersion(constraint string) error {
	return nil
}

func (d MockVsphereDriver) GetHostClusterMapping(ctx context.Context) (map[string]string, error) {
	return d.HostClusterMapping, nil
}

func (d MockVsphereDriver) GetVSphereVms(ctx context.Context, dcName string) ([]vsphere.VSphereVM, error) {
	return d.VMs, nil
}

func (d MockVsphereDriver) GetResourcePools(ctx context.Context, datacenter string, cluster string) ([]*object.ResourcePool, error) {
	return d.ResourcePools, nil
}

func (d MockVsphereDriver) GetVapps(ctx context.Context) ([]mo.VirtualApp, error) {
	return d.VApps, nil
}

func (d MockVsphereDriver) GetResourceTags(ctx context.Context, resourceType string) (map[string]tags.AttachedTags, error) {
	return d.ResourceTags, nil
}

func concat(ss ...string) string {
	return strings.Join(ss, "_")
}
