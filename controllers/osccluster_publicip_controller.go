package controllers

import (
	"context"
	"fmt"

	infrastructurev1beta1 "github.com/outscale-dev/cluster-api-provider-outscale.git/api/v1beta1"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/scope"
	"github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/services/security"
	tag "github.com/outscale-dev/cluster-api-provider-outscale.git/cloud/tag"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// getPublicIpResourceId return the resourceId from the resourceMap base on PublicIpName (tag name + cluster object uid)
func getPublicIpResourceId(resourceName string, clusterScope *scope.ClusterScope) (string, error) {
	publicIpRef := clusterScope.GetPublicIpRef()
	if publicIpId, ok := publicIpRef.ResourceMap[resourceName]; ok {
		return publicIpId, nil
	} else {
		return "", fmt.Errorf("%s is not exist", resourceName)
	}
}

// checkPublicIpFormatParameters check PublicIp parameters format (Tag format, cidr format, ..)
func checkPublicIpFormatParameters(clusterScope *scope.ClusterScope) (string, error) {
	clusterScope.Info("Check Public Ip parameters")
	var publicIpsSpec []*infrastructurev1beta1.OscPublicIp
	networkSpec := clusterScope.GetNetwork()
	if networkSpec.PublicIps == nil {
		networkSpec.SetPublicIpDefaultValue()
		publicIpsSpec = networkSpec.PublicIps
	} else {
		publicIpsSpec = clusterScope.GetPublicIp()
	}
	for _, publicIpSpec := range publicIpsSpec {
		publicIpName := publicIpSpec.Name + "-" + clusterScope.GetUID()
		publicIpTagName, err := tag.ValidateTagNameValue(publicIpName)
		if err != nil {
			return publicIpTagName, err
		}
	}
	return "", nil
}

// checkPublicIpOscAssociateResourceName check that PublicIp dependancies tag name in both resource configuration are the same.
func checkPublicIpOscAssociateResourceName(clusterScope *scope.ClusterScope) error {
	clusterScope.Info("check match public ip with nat service")
	var resourceNameList []string
	natServiceSpec := clusterScope.GetNatService()
	natServiceSpec.SetDefaultValue()
	natPublicIpName := natServiceSpec.PublicIpName + "-" + clusterScope.GetUID()
	var publicIpsSpec []*infrastructurev1beta1.OscPublicIp
	networkSpec := clusterScope.GetNetwork()
	publicIpsSpec = networkSpec.PublicIps
	for _, publicIpSpec := range publicIpsSpec {
		publicIpName := publicIpSpec.Name + "-" + clusterScope.GetUID()
		resourceNameList = append(resourceNameList, publicIpName)
	}
	checkOscAssociate := contains(resourceNameList, natPublicIpName)
	if checkOscAssociate {
		return nil
	} else {
		return fmt.Errorf("publicIp %s does not exist in natService ", natPublicIpName)
	}
}

// checkPublicIpOscDuplicateName check that there are not the same name for PublicIp resource.
func checkPublicIpOscDuplicateName(clusterScope *scope.ClusterScope) error {
	clusterScope.Info("Check unique name publicIp")
	var publicIpsSpec []*infrastructurev1beta1.OscPublicIp
	var resourceNameList []string
	publicIpsSpec = clusterScope.GetPublicIp()
	for _, publicIpSpec := range publicIpsSpec {
		resourceNameList = append(resourceNameList, publicIpSpec.Name)
	}
	duplicateResourceErr := alertDuplicate(resourceNameList)
	if duplicateResourceErr != nil {
		return duplicateResourceErr
	} else {
		return nil
	}
}

// reconcilePublicIp reconcile the PublicIp of the cluster.
func reconcilePublicIp(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	securitysvc := security.NewService(ctx, clusterScope)

	clusterScope.Info("Create PublicIp")
	var publicIpsSpec []*infrastructurev1beta1.OscPublicIp
	publicIpsSpec = clusterScope.GetPublicIp()
	var publicIpId string
	publicIpRef := clusterScope.GetPublicIpRef()
	var publicIpIds []string
	for _, publicIpSpec := range publicIpsSpec {
		publicIpId = publicIpSpec.ResourceId
		publicIpIds = append(publicIpIds, publicIpId)
	}
	validPublicIpIds, err := securitysvc.ValidatePublicIpIds(publicIpIds)
	if err != nil {
		return reconcile.Result{}, err
	}
	clusterScope.Info("### Check Id  ###", "publicip", publicIpIds)
	for _, publicIpSpec := range publicIpsSpec {
		publicIpName := publicIpSpec.Name + "-" + clusterScope.GetUID()
		publicIpId := publicIpRef.ResourceMap[publicIpName]
		clusterScope.Info("### Get publicIp Id ###", "publicip", publicIpRef.ResourceMap)
		if len(publicIpRef.ResourceMap) == 0 {
			publicIpRef.ResourceMap = make(map[string]string)
		}
		if publicIpSpec.ResourceId != "" {
			publicIpRef.ResourceMap[publicIpName] = publicIpSpec.ResourceId
		}
		if !contains(validPublicIpIds, publicIpId) {
			publicIp, err := securitysvc.CreatePublicIp(publicIpName)
			if err != nil {
				return reconcile.Result{}, fmt.Errorf("%w Can not create publicIp for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
			}
			clusterScope.Info("### Get publicIp  ###", "publicip", publicIp)
			publicIpRef.ResourceMap[publicIpName] = publicIp.GetPublicIpId()
			publicIpSpec.ResourceId = publicIp.GetPublicIpId()
		}
	}
	return reconcile.Result{}, nil
}

// reconcileDeletePublicIp reconcile the destruction of the PublicIp of the cluster.
func reconcileDeletePublicIp(ctx context.Context, clusterScope *scope.ClusterScope) (reconcile.Result, error) {
	securitysvc := security.NewService(ctx, clusterScope)
	osccluster := clusterScope.OscCluster

	clusterScope.Info("Delete PublicIp")
	var publicIpsSpec []*infrastructurev1beta1.OscPublicIp
	networkSpec := clusterScope.GetNetwork()
	if networkSpec.PublicIps == nil {
		networkSpec.SetPublicIpDefaultValue()
		publicIpsSpec = networkSpec.PublicIps
	} else {
		publicIpsSpec = clusterScope.GetPublicIp()
	}
	var publicIpIds []string
	var publicIpId string
	for _, publicIpSpec := range publicIpsSpec {
		publicIpId = publicIpSpec.ResourceId
		publicIpIds = append(publicIpIds, publicIpId)
	}
	validPublicIpIds, err := securitysvc.ValidatePublicIpIds(publicIpIds)
	if err != nil {
		return reconcile.Result{}, err
	}
	clusterScope.Info("### Check Id  ###", "publicip", publicIpIds)
	for _, publicIpSpec := range publicIpsSpec {
		//		publicIpName := publicIpSpec.Name + "-" + clusterScope.GetUID()
		publicIpId := publicIpSpec.ResourceId
		if !contains(validPublicIpIds, publicIpId) {
			controllerutil.RemoveFinalizer(osccluster, "oscclusters.infrastructure.cluster.x-k8s.io")
			return reconcile.Result{}, nil
		}
		clusterScope.Info("Remove publicip")
		err = securitysvc.DeletePublicIp(publicIpId)
		if err != nil {
			return reconcile.Result{}, fmt.Errorf("%w Can not delete publicIp for Osccluster %s/%s", err, clusterScope.GetNamespace(), clusterScope.GetName())
		}

	}
	return reconcile.Result{}, nil
}