package resources

import (
	"github.com/klothoplatform/klotho/pkg/core"
)

const (
	EFS_ACCESS_POINT_TYPE = "efs_access_point"
	EFS_FILE_SYSTEM_TYPE  = "efs_file_system"
	EFS_MOUNT_TARGET_TYPE = "efs_mount_target"

	EFS_MOUNT_PATH_IAC_VALUE string = "efs_mount_path"
)

type (
	EfsToIaPolicy      string
	EfsToPrimaryPolicy string
	EfsLifecyclePolicy struct {
		// TransitionToIA is the transition to IA of the EfsLifecyclePolicy
		TransitionToIA EfsToIaPolicy
		// TransitionToPrimaryStorageClass is the transition to primary storage class of the EfsLifecyclePolicy
		TransitionToPrimaryStorageClass EfsToPrimaryPolicy
	}

	EfsFileSystem struct {
		// Name is the name of the EfsFileSystem
		Name          string
		ConstructRefs core.BaseConstructSet `yaml:"-"`
		// PerformanceMode is the performance mode of the EfsFileSystem
		PerformanceMode string
		// ThroughputMode is the throughput mode of the EfsFileSystem
		ThroughputMode string
		// ProvisionedThroughputInMibps is the provisioned throughput of the EfsFileSystem
		ProvisionedThroughputInMibps int
		// Encrypted is the encryption status of the EfsFileSystem
		Encrypted bool
		// KmsKey is the kms key of the EfsFileSystem
		KmsKey *KmsKey
		// LifecyclePolicies is the list of lifecycle policies of the EfsFileSystem
		LifecyclePolicies []*EfsLifecyclePolicy
		// AvailabilityZoneName is the availability zone name of the EfsFileSystem when running in a single AZ
		AvailabilityZoneName *core.IaCValue
		// CreationToken is the creation token of the EfsFileSystem
		CreationToken string
	}

	EfsMountTarget struct {
		// Name is the name of the EFS core.Resource
		Name          string
		ConstructRefs core.BaseConstructSet `yaml:"-"`
		// FileSystem is the file system associated with the EfsMountTarget
		FileSystem *EfsFileSystem
		// Subnets is the list of subnets associated with the EfsMountTarget
		Subnet *Subnet
		// SecurityGroups is the list of security groups associated with the EfsMountTarget
		SecurityGroups []*SecurityGroup
		// IpAddress is the ip address of the EfsMountTarget
		IpAddress string
	}

	EfsPosixUser struct {
		// Uid is the user id
		Uid int
		// Gid is the group id
		Gid int
	}

	EfsRootDirectoryCreationInfo struct {
		// OwnerGid is the owner group id
		OwnerGid int
		// OwnerUid is the owner user id
		OwnerUid int
		// Permissions is the permissions of the root directory
		Permissions string
	}

	EfsRootDirectory struct {
		// Path is the path of the root directory
		Path string
		// CreationInfo is the creation info of the root directory
		CreationInfo *EfsRootDirectoryCreationInfo
	}

	EfsAccessPoint struct {
		// Name is the name of the EfsAccessPoint
		Name string
		// ConstructRefs is the list of references to the EfsAccessPoint
		ConstructRefs core.BaseConstructSet `yaml:"-"`
		// FileSystem is the file system associated with the EfsAccessPoint
		FileSystem *EfsFileSystem
		// PosixUser is the posix user associated with the EfsAccessPoint
		PosixUser *EfsPosixUser
		// RootDirectory is the root directory associated with the EfsAccessPoint
		RootDirectory *EfsRootDirectory
	}
)

var (
	EfsToIaAfter1Day   = EfsToIaPolicy("AFTER_1_DAY")
	EfsToIaAfter7Days  = EfsToIaPolicy("AFTER_7_DAYS")
	EfsToIaAfter14Days = EfsToIaPolicy("AFTER_14_DAYS")
	EfsToIaAfter30Days = EfsToIaPolicy("AFTER_30_DAYS")
	EfsToIaAfter60Days = EfsToIaPolicy("AFTER_60_DAYS")
	EfsToIaAfter90Days = EfsToIaPolicy("AFTER_90_DAYS")

	EfsToPrimaryAfter1Access = EfsToPrimaryPolicy("AFTER_1_ACCESS")
)

func (efs *EfsFileSystem) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     EFS_FILE_SYSTEM_TYPE,
		Name:     efs.Name,
	}
}

func (efs *EfsFileSystem) BaseConstructRefs() core.BaseConstructSet {
	return efs.ConstructRefs
}

func (efs *EfsFileSystem) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream:     true,
		RequiresNoDownstream:   true,
		RequiresExplicitDelete: true,
	}
}

func (eap *EfsAccessPoint) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     EFS_ACCESS_POINT_TYPE,
		Name:     eap.Name,
	}
}

func (eap *EfsAccessPoint) BaseConstructRefs() core.BaseConstructSet {
	return eap.ConstructRefs
}

func (eap *EfsAccessPoint) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream:     true,
		RequiresNoDownstream:   false,
		RequiresExplicitDelete: false,
	}
}

func (emt *EfsMountTarget) Id() core.ResourceId {
	return core.ResourceId{
		Provider: AWS_PROVIDER,
		Type:     EFS_MOUNT_TARGET_TYPE,
		Name:     emt.Name,
	}
}

func (emt *EfsMountTarget) BaseConstructRefs() core.BaseConstructSet {
	return emt.ConstructRefs
}

func (emt *EfsMountTarget) DeleteContext() core.DeleteContext {
	return core.DeleteContext{
		RequiresNoUpstream:     true,
		RequiresNoDownstream:   false,
		RequiresExplicitDelete: false,
	}
}

type EfsMountTargetCreateParams struct {
	Name          string
	ConstructRefs core.BaseConstructSet
}

func (emt *EfsMountTarget) Create(dag *core.ResourceGraph, params EfsMountTargetCreateParams) error {
	emt.Name = params.Name
	emt.ConstructRefs = params.ConstructRefs.Clone()
	return nil
}
