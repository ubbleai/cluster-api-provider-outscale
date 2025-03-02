/*
Copyright 2022 The Kubernetes Authors.

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

package test

import (
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"golang.org/x/net/context"
	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type CapoKubeAdmControlPlaneListInput struct {
	Lister      client.Client
	ListOptions *client.ListOptions
}

type CapoKubeAdmControlPlaneListDeleteInput struct {
	Deleter     client.Client
	ListOptions *client.ListOptions
}

// GetCapoKubeAdmControlPlaneList get kubeadmcontrolplane.
func GetCapoKubeAdmControlPlaneList(ctx context.Context, input CapoKubeAdmControlPlaneListInput) bool {
	capoKubeAdmControlPlaneList := &controlplanev1.KubeadmControlPlaneList{}
	if err := input.Lister.List(ctx, capoKubeAdmControlPlaneList, input.ListOptions); err != nil {
		By(fmt.Sprintf("Can not list CapoKubeAdmControlPlane %s", err))
		return false
	}
	for _, capoKubeAdmControlPlane := range capoKubeAdmControlPlaneList.Items {
		By(fmt.Sprintf("Find capoKubeAdmControlPlane %s in namespace %s \n", capoKubeAdmControlPlane.Name, capoKubeAdmControlPlane.Namespace))
	}
	return true
}

// DeleteCapoKubeAdmControlPlaneList delete kubeadmcontrolplane.
func DeleteCapoKubeAdmControlPlaneList(ctx context.Context, input CapoKubeAdmControlPlaneListDeleteInput) bool {
	capoKubeAdmControlPlaneList := &controlplanev1.KubeadmControlPlaneList{}
	if err := input.Deleter.List(ctx, capoKubeAdmControlPlaneList, input.ListOptions); err != nil {
		By(fmt.Sprintf("Can not list CapoKubeAdmControlPlaneLisr %s", err))
		return false
	}
	var key client.ObjectKey
	var capoKubeAdmControlPlaneGet *controlplanev1.KubeadmControlPlane
	for _, capoKubeAdmControlPlane := range capoKubeAdmControlPlaneList.Items {
		By(fmt.Sprintf("Find capoKubeAdmControlPlane %s in namespace %s to be deleted \n", capoKubeAdmControlPlane.Name, capoKubeAdmControlPlane.Namespace))
		capoKubeAdmControlPlaneGet = &controlplanev1.KubeadmControlPlane{}
		key = client.ObjectKey{
			Namespace: capoKubeAdmControlPlane.Namespace,
			Name:      capoKubeAdmControlPlane.Name,
		}
		if err := input.Deleter.Get(ctx, key, capoKubeAdmControlPlaneGet); err != nil {
			By(fmt.Sprintf("Can not find %s\n", err))
			return false
		}
		Eventually(func() error {
			return input.Deleter.Delete(ctx, capoKubeAdmControlPlaneGet)
		}, 30*time.Second, 10*time.Second).Should(Succeed())
		fmt.Fprintf(GinkgoWriter, "Delete KubeAdmControlPlane pending \n")
		time.Sleep(20 * time.Second)
		if err := input.Deleter.Get(ctx, key, capoKubeAdmControlPlaneGet); err != nil {
			By(fmt.Sprintf("Can not find %s, continue\n", err))
		} else {
			capoKubeAdmControlPlaneGet.ObjectMeta.Finalizers = nil
			Expect(input.Deleter.Update(ctx, capoKubeAdmControlPlaneGet)).Should(Succeed())
			fmt.Fprintf(GinkgoWriter, "Patch machine \n")
		}

		capoKubeAdmControlPlaneGet = &controlplanev1.KubeadmControlPlane{}
		EventuallyWithOffset(1, func() error {
			fmt.Fprintf(GinkgoWriter, "Wait %s in namespace %s to be deleted \n", capoKubeAdmControlPlane.Name, capoKubeAdmControlPlane.Namespace)
			return input.Deleter.Get(ctx, key, capoKubeAdmControlPlaneGet)
		}, 1*time.Minute, 5*time.Second).ShouldNot(Succeed())

	}
	return true
}

// WaitForCapoKubeAdmControlPLaneListAvailable wait kubeadmcontolplane.
func WaitForCapoKubeAdmControlPLaneListAvailable(ctx context.Context, input CapoKubeAdmControlPlaneListInput) bool {
	By("Waiting for kubeAdmControlPlane selected by options to be ready")
	Eventually(func() bool {
		isCapoKubeAdmControlPlaneListAvailable := GetCapoKubeAdmControlPlaneList(ctx, input)
		return isCapoKubeAdmControlPlaneListAvailable
	}, 15*time.Second, 3*time.Second).Should(BeTrue(), "Failed to find capoKubeAdmControlPlaneList")
	return false
}

// WaitForCapoKubeAdmControlPlaneListDelete wait kubeadmcontrolplane to be deleted.
func WaitForCapoKubeAdmControlPlaneListDelete(ctx context.Context, input CapoKubeAdmControlPlaneListDeleteInput) bool {
	By("Waiting for capoMachineList selected by options to be deleted")
	Eventually(func() bool {
		isCapoKubeAdmControlPlaneListDelete := DeleteCapoKubeAdmControlPlaneList(ctx, input)
		return isCapoKubeAdmControlPlaneListDelete
	}, 15*time.Second, 3*time.Second).Should(BeTrue(), "Failed to find capoKubeAdmControlPlaneList")
	return false
}
