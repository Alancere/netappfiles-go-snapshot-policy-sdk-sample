// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

// This package centralizes any function that directly
// using any of the Azure's (with exception of authentication related ones)
// available SDK packages.

package sdkutils

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Azure-Samples/netappfiles-go-snapshot-policy-sdk-sample/netappfiles-go-snapshot-policy-sdk-sample/internal/iam"
	"github.com/Azure-Samples/netappfiles-go-snapshot-policy-sdk-sample/netappfiles-go-snapshot-policy-sdk-sample/internal/uri"
	"github.com/Azure-Samples/netappfiles-go-snapshot-policy-sdk-sample/netappfiles-go-snapshot-policy-sdk-sample/internal/utils"

	"github.com/Azure/azure-sdk-for-go/services/netapp/mgmt/2020-06-01/netapp"
	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-10-01/resources"
	"github.com/Azure/go-autorest/autorest/to"
)

const (
	userAgent = "anf-go-sdk-sample-agent"
	nfsv3     = "NFSv3"
	nfsv41    = "NFSv4.1"
	cifs      = "CIFS"
)

var (
	validProtocols = []string{nfsv3, nfsv41, cifs}
)

func validateANFServiceLevel(serviceLevel string) (validatedServiceLevel netapp.ServiceLevel, err error) {

	var svcLevel netapp.ServiceLevel

	switch strings.ToLower(serviceLevel) {
	case "ultra":
		svcLevel = netapp.Ultra
	case "premium":
		svcLevel = netapp.Premium
	case "standard":
		svcLevel = netapp.Standard
	default:
		return "", fmt.Errorf("invalid service level, supported service levels are: %v", netapp.PossibleServiceLevelValues())
	}

	return svcLevel, nil
}

func getResourcesClient() (resources.Client, error) {

	authorizer, subscriptionID, err := iam.GetAuthorizer()
	if err != nil {
		return resources.Client{}, err
	}

	client := resources.NewClient(subscriptionID)
	client.Authorizer = authorizer
	client.AddToUserAgent(userAgent)

	return client, nil
}

func getAccountsClient() (netapp.AccountsClient, error) {

	authorizer, subscriptionID, err := iam.GetAuthorizer()
	if err != nil {
		return netapp.AccountsClient{}, err
	}

	client := netapp.NewAccountsClient(subscriptionID)
	client.Authorizer = authorizer
	client.AddToUserAgent(userAgent)

	return client, nil
}

func getPoolsClient() (netapp.PoolsClient, error) {

	authorizer, subscriptionID, err := iam.GetAuthorizer()
	if err != nil {
		return netapp.PoolsClient{}, err
	}

	client := netapp.NewPoolsClient(subscriptionID)
	client.Authorizer = authorizer
	client.AddToUserAgent(userAgent)

	return client, nil
}

func getVolumesClient() (netapp.VolumesClient, error) {

	authorizer, subscriptionID, err := iam.GetAuthorizer()
	if err != nil {
		return netapp.VolumesClient{}, err
	}

	client := netapp.NewVolumesClient(subscriptionID)
	client.Authorizer = authorizer
	client.AddToUserAgent(userAgent)

	return client, nil
}

func getSnapshotsClient() (netapp.SnapshotsClient, error) {

	authorizer, subscriptionID, err := iam.GetAuthorizer()
	if err != nil {
		return netapp.SnapshotsClient{}, err
	}

	client := netapp.NewSnapshotsClient(subscriptionID)
	client.Authorizer = authorizer
	client.AddToUserAgent(userAgent)

	return client, nil
}

func getSnapshotPoliciesClient() (netapp.SnapshotPoliciesClient, error) {

	authorizer, subscriptionID, err := iam.GetAuthorizer()
	if err != nil {
		return netapp.SnapshotPoliciesClient{}, err
	}

	client := netapp.NewSnapshotPoliciesClient(subscriptionID)
	client.Authorizer = authorizer
	client.AddToUserAgent(userAgent)

	return client, nil
}

// GetResourceByID gets a generic resource
func GetResourceByID(ctx context.Context, resourceID, APIVersion string) (resources.GenericResource, error) {

	resourcesClient, err := getResourcesClient()
	if err != nil {
		return resources.GenericResource{}, err
	}

	parentResource := ""
	resourceGroup := uri.GetResourceGroup(resourceID)
	resourceProvider := uri.GetResourceValue(resourceID, "providers")
	resourceName := uri.GetResourceName(resourceID)
	resourceType := uri.GetResourceValue(resourceID, resourceProvider)

	if strings.Contains(resourceID, "/subnets/") {
		parentResourceName := uri.GetResourceValue(resourceID, resourceType)
		parentResource = fmt.Sprintf("%v/%v", resourceType, parentResourceName)
		resourceType = "subnets"
	}

	return resourcesClient.Get(
		ctx,
		resourceGroup,
		resourceProvider,
		parentResource,
		resourceType,
		resourceName,
		APIVersion,
	)
}

// CreateANFAccount creates an ANF Account resource
func CreateANFAccount(ctx context.Context, location, resourceGroupName, accountName string, activeDirectories []netapp.ActiveDirectory, tags map[string]*string) (netapp.Account, error) {

	accountClient, err := getAccountsClient()
	if err != nil {
		return netapp.Account{}, err
	}

	accountProperties := netapp.AccountProperties{}

	if activeDirectories != nil {
		accountProperties = netapp.AccountProperties{
			ActiveDirectories: &activeDirectories,
		}
	}

	future, err := accountClient.CreateOrUpdate(
		ctx,
		netapp.Account{
			Location:          to.StringPtr(location),
			Tags:              tags,
			AccountProperties: &accountProperties,
		},
		resourceGroupName,
		accountName,
	)
	if err != nil {
		return netapp.Account{}, fmt.Errorf("cannot create account: %v", err)
	}

	err = future.WaitForCompletionRef(ctx, accountClient.Client)
	if err != nil {
		return netapp.Account{}, fmt.Errorf("cannot get the account create or update future response: %v", err)
	}

	return future.Result(accountClient)
}

// CreateANFCapacityPool creates an ANF Capacity Pool within ANF Account
func CreateANFCapacityPool(ctx context.Context, location, resourceGroupName, accountName, poolName, serviceLevel string, sizeBytes int64, tags map[string]*string) (netapp.CapacityPool, error) {

	poolClient, err := getPoolsClient()
	if err != nil {
		return netapp.CapacityPool{}, err
	}

	svcLevel, err := validateANFServiceLevel(serviceLevel)
	if err != nil {
		return netapp.CapacityPool{}, err
	}

	future, err := poolClient.CreateOrUpdate(
		ctx,
		netapp.CapacityPool{
			Location: to.StringPtr(location),
			Tags:     tags,
			PoolProperties: &netapp.PoolProperties{
				ServiceLevel: svcLevel,
				Size:         to.Int64Ptr(sizeBytes),
			},
		},
		resourceGroupName,
		accountName,
		poolName,
	)

	if err != nil {
		return netapp.CapacityPool{}, fmt.Errorf("cannot create pool: %v", err)
	}

	err = future.WaitForCompletionRef(ctx, poolClient.Client)
	if err != nil {
		return netapp.CapacityPool{}, fmt.Errorf("cannot get the pool create or update future response: %v", err)
	}

	return future.Result(poolClient)
}

// CreateANFVolume creates an ANF volume within a Capacity Pool
func CreateANFVolume(ctx context.Context, location, resourceGroupName, accountName, poolName, volumeName, serviceLevel, subnetID, snapshotID string, protocolTypes []string, volumeUsageQuota int64, unixReadOnly, unixReadWrite bool, tags map[string]*string, dataProtectionObject netapp.VolumePropertiesDataProtection) (netapp.Volume, error) {

	if len(protocolTypes) > 2 {
		return netapp.Volume{}, fmt.Errorf("maximum of two protocol types are supported")
	}

	if len(protocolTypes) > 1 && utils.Contains(protocolTypes, "NFSv4.1") {
		return netapp.Volume{}, fmt.Errorf("only cifs/nfsv3 protocol types are supported as dual protocol")
	}

	_, found := utils.FindInSlice(validProtocols, protocolTypes[0])
	if !found {
		return netapp.Volume{}, fmt.Errorf("invalid protocol type, valid protocol types are: %v", validProtocols)
	}

	svcLevel, err := validateANFServiceLevel(serviceLevel)
	if err != nil {
		return netapp.Volume{}, err
	}

	volumeClient, err := getVolumesClient()
	if err != nil {
		return netapp.Volume{}, err
	}

	exportPolicy := netapp.VolumePropertiesExportPolicy{}

	if _, found := utils.FindInSlice(protocolTypes, cifs); !found {
		exportPolicy = netapp.VolumePropertiesExportPolicy{
			Rules: &[]netapp.ExportPolicyRule{
				{
					AllowedClients: to.StringPtr("0.0.0.0/0"),
					Cifs:           to.BoolPtr(map[bool]bool{true: true, false: false}[protocolTypes[0] == cifs]),
					Nfsv3:          to.BoolPtr(map[bool]bool{true: true, false: false}[protocolTypes[0] == nfsv3]),
					Nfsv41:         to.BoolPtr(map[bool]bool{true: true, false: false}[protocolTypes[0] == nfsv41]),
					RuleIndex:      to.Int32Ptr(1),
					UnixReadOnly:   to.BoolPtr(unixReadOnly),
					UnixReadWrite:  to.BoolPtr(unixReadWrite),
				},
			},
		}
	}

	volumeProperties := netapp.VolumeProperties{
		SnapshotID:     map[bool]*string{true: to.StringPtr(snapshotID), false: nil}[snapshotID != ""],
		ExportPolicy:   map[bool]*netapp.VolumePropertiesExportPolicy{true: &exportPolicy, false: nil}[protocolTypes[0] != cifs],
		ProtocolTypes:  &protocolTypes,
		ServiceLevel:   svcLevel,
		SubnetID:       to.StringPtr(subnetID),
		UsageThreshold: to.Int64Ptr(volumeUsageQuota),
		CreationToken:  to.StringPtr(volumeName),
		DataProtection: &dataProtectionObject,
	}

	future, err := volumeClient.CreateOrUpdate(
		ctx,
		netapp.Volume{
			Location:         to.StringPtr(location),
			Tags:             tags,
			VolumeProperties: &volumeProperties,
		},
		resourceGroupName,
		accountName,
		poolName,
		volumeName,
	)

	if err != nil {
		return netapp.Volume{}, fmt.Errorf("cannot create volume: %v", err)
	}

	err = future.WaitForCompletionRef(ctx, volumeClient.Client)
	if err != nil {
		return netapp.Volume{}, fmt.Errorf("cannot get the volume create or update future response: %v", err)
	}

	return future.Result(volumeClient)
}

// UpdateANFVolume update an ANF volume
func UpdateANFVolume(ctx context.Context, location, resourceGroupName, accountName, poolName, volumeName string, volumePropertiesPatch netapp.VolumePatchProperties, tags map[string]*string) (netapp.VolumesUpdateFuture, error) {

	volumeClient, err := getVolumesClient()
	if err != nil {
		return netapp.VolumesUpdateFuture{}, err
	}

	volume, err := volumeClient.Update(
		ctx,
		netapp.VolumePatch{
			Location:              to.StringPtr(location),
			Tags:                  tags,
			VolumePatchProperties: &volumePropertiesPatch,
		},
		resourceGroupName,
		accountName,
		poolName,
		volumeName,
	)

	if err != nil {
		return netapp.VolumesUpdateFuture{}, fmt.Errorf("cannot update volume: %v", err)
	}

	return volume, nil
}

// AuthorizeReplication - authorizes volume replication
func AuthorizeReplication(ctx context.Context, resourceGroupName, accountName, poolName, volumeName, remoteVolumeResourceID string) error {

	volumeClient, err := getVolumesClient()
	if err != nil {
		return err
	}

	future, err := volumeClient.AuthorizeReplication(
		ctx,
		resourceGroupName,
		accountName,
		poolName,
		volumeName,
		netapp.AuthorizeRequest{
			RemoteVolumeResourceID: to.StringPtr(remoteVolumeResourceID),
		},
	)

	if err != nil {
		return fmt.Errorf("cannot authorize volume replication: %v", err)
	}

	err = future.WaitForCompletionRef(ctx, volumeClient.Client)
	if err != nil {
		return fmt.Errorf("cannot get authorize volume replication future response: %v", err)
	}

	return nil
}

// DeleteANFVolumeReplication - authorizes volume replication
func DeleteANFVolumeReplication(ctx context.Context, resourceGroupName, accountName, poolName, volumeName string) error {

	volumeClient, err := getVolumesClient()
	if err != nil {
		return err
	}

	future, err := volumeClient.DeleteReplication(
		ctx,
		resourceGroupName,
		accountName,
		poolName,
		volumeName,
	)

	if err != nil {
		return fmt.Errorf("cannot delete volume replication: %v", err)
	}

	err = future.WaitForCompletionRef(ctx, volumeClient.Client)
	if err != nil {
		return fmt.Errorf("cannot get delete volume replication future response: %v", err)
	}

	return nil
}

// CreateANFSnapshot creates a Snapshot from an ANF volume
func CreateANFSnapshot(ctx context.Context, location, resourceGroupName, accountName, poolName, volumeName, snapshotName string, tags map[string]*string) (netapp.Snapshot, error) {

	snapshotClient, err := getSnapshotsClient()
	if err != nil {
		return netapp.Snapshot{}, err
	}

	future, err := snapshotClient.Create(
		ctx,
		netapp.Snapshot{
			Location: to.StringPtr(location),
		},
		resourceGroupName,
		accountName,
		poolName,
		volumeName,
		snapshotName,
	)

	if err != nil {
		return netapp.Snapshot{}, fmt.Errorf("cannot create snapshot: %v", err)
	}

	err = future.WaitForCompletionRef(ctx, snapshotClient.Client)
	if err != nil {
		return netapp.Snapshot{}, fmt.Errorf("cannot get the snapshot create or update future response: %v", err)
	}

	return future.Result(snapshotClient)
}

// DeleteANFSnapshot deletes a Snapshot from an ANF volume
func DeleteANFSnapshot(ctx context.Context, resourceGroupName, accountName, poolName, volumeName, snapshotName string) error {

	snapshotClient, err := getSnapshotsClient()
	if err != nil {
		return err
	}

	future, err := snapshotClient.Delete(
		ctx,
		resourceGroupName,
		accountName,
		poolName,
		volumeName,
		snapshotName,
	)

	if err != nil {
		return fmt.Errorf("cannot delete snapshot: %v", err)
	}

	err = future.WaitForCompletionRef(ctx, snapshotClient.Client)
	if err != nil {
		return fmt.Errorf("cannot get the snapshot delete future response: %v", err)
	}

	return nil
}

// CreateANFSnapshotPolicy creates a Snapshot Policy to be used on volumes
func CreateANFSnapshotPolicy(ctx context.Context, resourceGroupName, accountName, policyName string, policy netapp.SnapshotPolicy) (netapp.SnapshotPolicy, error) {

	snapshotPolicyClient, err := getSnapshotPoliciesClient()
	if err != nil {
		return netapp.SnapshotPolicy{}, err
	}

	snapshotPolicy, err := snapshotPolicyClient.Create(
		ctx,
		policy,
		resourceGroupName,
		accountName,
		policyName,
	)

	if err != nil {
		return netapp.SnapshotPolicy{}, fmt.Errorf("cannot create snapshot policy: %v", err)
	}

	return snapshotPolicy, nil
}

// UpdateANFSnapshotPolicy update an ANF volume
func UpdateANFSnapshotPolicy(ctx context.Context, resourceGroupName, accountName, policyName string, snapshotPolicyPatch netapp.SnapshotPolicyPatch) (netapp.SnapshotPolicy, error) {

	snapshotPolicyClient, err := getSnapshotPoliciesClient()
	if err != nil {
		return netapp.SnapshotPolicy{}, err
	}

	snapshotPolicy, err := snapshotPolicyClient.Update(
		ctx,
		snapshotPolicyPatch,
		resourceGroupName,
		accountName,
		policyName,
	)

	if err != nil {
		return netapp.SnapshotPolicy{}, fmt.Errorf("cannot update snapshot policy: %v", err)
	}

	return snapshotPolicy, nil
}

// DeleteANFVolume deletes a volume
func DeleteANFVolume(ctx context.Context, resourceGroupName, accountName, poolName, volumeName string) error {

	volumesClient, err := getVolumesClient()
	if err != nil {
		return err
	}

	future, err := volumesClient.Delete(
		ctx,
		resourceGroupName,
		accountName,
		poolName,
		volumeName,
	)

	if err != nil {
		return fmt.Errorf("cannot delete volume: %v", err)
	}

	err = future.WaitForCompletionRef(ctx, volumesClient.Client)
	if err != nil {
		return fmt.Errorf("cannot get the volume delete future response: %v", err)
	}

	return nil
}

// DeleteANFCapacityPool deletes a capacity pool
func DeleteANFCapacityPool(ctx context.Context, resourceGroupName, accountName, poolName string) error {

	poolsClient, err := getPoolsClient()
	if err != nil {
		return err
	}

	future, err := poolsClient.Delete(
		ctx,
		resourceGroupName,
		accountName,
		poolName,
	)

	if err != nil {
		return fmt.Errorf("cannot delete capacity pool: %v", err)
	}

	err = future.WaitForCompletionRef(ctx, poolsClient.Client)
	if err != nil {
		return fmt.Errorf("cannot get the capacity pool delete future response: %v", err)
	}

	return nil
}

// DeleteANFSnapshotPolicy deletes a snapshot policy
func DeleteANFSnapshotPolicy(ctx context.Context, resourceGroupName, accountName, policyName string) error {

	snapshotPolicyClient, err := getSnapshotPoliciesClient()
	if err != nil {
		return err
	}

	future, err := snapshotPolicyClient.Delete(
		ctx,
		resourceGroupName,
		accountName,
		policyName,
	)

	if err != nil {
		return fmt.Errorf("cannot delete snapshot policy: %v", err)
	}

	err = future.WaitForCompletionRef(ctx, snapshotPolicyClient.Client)
	if err != nil {
		return fmt.Errorf("cannot get the snapshot policy delete future response: %v", err)
	}

	return nil
}

// DeleteANFAccount deletes an account
func DeleteANFAccount(ctx context.Context, resourceGroupName, accountName string) error {

	accountsClient, err := getAccountsClient()
	if err != nil {
		return err
	}

	future, err := accountsClient.Delete(
		ctx,
		resourceGroupName,
		accountName,
	)

	if err != nil {
		return fmt.Errorf("cannot delete account: %v", err)
	}

	err = future.WaitForCompletionRef(ctx, accountsClient.Client)
	if err != nil {
		return fmt.Errorf("cannot get the account delete future response: %v", err)
	}

	return nil
}

// WaitForNoANFResource waits for a specified resource to don't exist anymore following a deletion.
// This is due to a known issue related to ARM Cache where the state of the resource is still cached within ARM infrastructure
// reporting that it still exists so looping into a get process will return 404 as soon as the cached state expires
func WaitForNoANFResource(ctx context.Context, resourceID string, intervalInSec int, retries int, checkForReplication bool) error {

	var err error

	for i := 0; i < retries; i++ {
		time.Sleep(time.Duration(intervalInSec) * time.Second)
		if uri.IsANFSnapshot(resourceID) {
			client, _ := getSnapshotsClient()
			_, err = client.Get(
				ctx,
				uri.GetResourceGroup(resourceID),
				uri.GetANFAccount(resourceID),
				uri.GetANFCapacityPool(resourceID),
				uri.GetANFVolume(resourceID),
				uri.GetANFSnapshot(resourceID),
			)
		} else if uri.IsANFVolume(resourceID) {
			client, _ := getVolumesClient()
			if checkForReplication == false {
				_, err = client.Get(
					ctx,
					uri.GetResourceGroup(resourceID),
					uri.GetANFAccount(resourceID),
					uri.GetANFCapacityPool(resourceID),
					uri.GetANFVolume(resourceID),
				)
			} else {
				_, err = client.ReplicationStatusMethod(
					ctx,
					uri.GetResourceGroup(resourceID),
					uri.GetANFAccount(resourceID),
					uri.GetANFCapacityPool(resourceID),
					uri.GetANFVolume(resourceID),
				)
			}
		} else if uri.IsANFCapacityPool(resourceID) {
			client, _ := getPoolsClient()
			_, err = client.Get(
				ctx,
				uri.GetResourceGroup(resourceID),
				uri.GetANFAccount(resourceID),
				uri.GetANFCapacityPool(resourceID),
			)
		} else if uri.IsANFSnapshotPolicy(resourceID) {
			client, _ := getSnapshotPoliciesClient()
			_, err = client.Get(
				ctx,
				uri.GetResourceGroup(resourceID),
				uri.GetANFAccount(resourceID),
				uri.GetANFSnapshotPolicy(resourceID),
			)
		} else if uri.IsANFAccount(resourceID) {
			client, _ := getAccountsClient()
			_, err = client.Get(
				ctx,
				uri.GetResourceGroup(resourceID),
				uri.GetANFAccount(resourceID),
			)
		}

		// In this case error is expected
		if err != nil {
			return nil
		}
	}

	return fmt.Errorf("exceeded number of retries: %v", retries)
}

// WaitForANFResource waits for a specified resource to be fully ready following a creation operation.
func WaitForANFResource(ctx context.Context, resourceID string, intervalInSec int, retries int, checkForReplication bool) error {

	var err error

	for i := 0; i < retries; i++ {
		time.Sleep(time.Duration(intervalInSec) * time.Second)
		if uri.IsANFSnapshot(resourceID) {
			client, _ := getSnapshotsClient()
			_, err = client.Get(
				ctx,
				uri.GetResourceGroup(resourceID),
				uri.GetANFAccount(resourceID),
				uri.GetANFCapacityPool(resourceID),
				uri.GetANFVolume(resourceID),
				uri.GetANFSnapshot(resourceID),
			)
		} else if uri.IsANFVolume(resourceID) {
			client, _ := getVolumesClient()
			if checkForReplication == false {
				_, err = client.Get(
					ctx,
					uri.GetResourceGroup(resourceID),
					uri.GetANFAccount(resourceID),
					uri.GetANFCapacityPool(resourceID),
					uri.GetANFVolume(resourceID),
				)
			} else {
				_, err = client.ReplicationStatusMethod(
					ctx,
					uri.GetResourceGroup(resourceID),
					uri.GetANFAccount(resourceID),
					uri.GetANFCapacityPool(resourceID),
					uri.GetANFVolume(resourceID),
				)
			}
		} else if uri.IsANFCapacityPool(resourceID) {
			client, _ := getPoolsClient()
			_, err = client.Get(
				ctx,
				uri.GetResourceGroup(resourceID),
				uri.GetANFAccount(resourceID),
				uri.GetANFCapacityPool(resourceID),
			)
		} else if uri.IsANFSnapshotPolicy(resourceID) {
			client, _ := getSnapshotPoliciesClient()
			_, err = client.Get(
				ctx,
				uri.GetResourceGroup(resourceID),
				uri.GetANFAccount(resourceID),
				uri.GetANFSnapshotPolicy(resourceID),
			)
		} else if uri.IsANFAccount(resourceID) {
			client, _ := getAccountsClient()
			_, err = client.Get(
				ctx,
				uri.GetResourceGroup(resourceID),
				uri.GetANFAccount(resourceID),
			)
		}

		// In this case, we exit when there is no error
		if err == nil {
			return nil
		}
	}

	return fmt.Errorf("resource still not found after number of retries: %v, error: %v", retries, err)
}
