/*
Copyright 2025.

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

package parallel

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	argov1beta1api "github.com/argoproj-labs/argocd-operator/api/v1beta1"
	"github.com/argoproj-labs/argocd-operator/common"
	"github.com/argoproj-labs/argocd-operator/tests/ginkgo/fixture"
	argocdFixture "github.com/argoproj-labs/argocd-operator/tests/ginkgo/fixture/argocd"
	fixtureUtils "github.com/argoproj-labs/argocd-operator/tests/ginkgo/fixture/utils"
)

// The e2e operator runs as a local process (make start-e2e), so it has no operator
// namespace and reconcileImagePullSecrets skips the operator-NS->instance-NS copy.
// The in-namespace path is what is exercised here: a Secret labeled
// propagate-image-pull-secret=true in the instance namespace is resolved by
// getImagePullSecretRefs and set as imagePullSecrets on the component ServiceAccounts.
var _ = Describe("GitOps Operator Parallel E2E Tests", func() {

	Context("1-142_validate_image_pull_secret_propagation", func() {

		var (
			k8sClient client.Client
			ctx       context.Context
		)

		BeforeEach(func() {
			fixture.EnsureParallelCleanSlate()
			var err error
			k8sClient, _, err = fixtureUtils.GetE2ETestKubeClientWithError()
			Expect(err).NotTo(HaveOccurred())
			ctx = context.Background()
		})

		// dockerCfg is a minimal valid .dockerconfigjson payload.
		dockerCfg := map[string][]byte{".dockerconfigjson": []byte(`{"auths":{}}`)}

		newLabeledPullSecret := func(name, ns string) *corev1.Secret {
			return &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: ns,
					Labels:    map[string]string{common.ArgoCDImagePullSecretPropagateLabel: "true"},
				},
				Type: corev1.SecretTypeDockerConfigJson,
				Data: dockerCfg,
			}
		}

		// expectSAPullSecret asserts (eventually) whether the named ServiceAccount's
		// imagePullSecrets contains secretName. There is no ServiceAccount fixture matcher.
		expectSAPullSecret := func(saName, ns, secretName string, present bool) {
			Eventually(func(g Gomega) {
				sa := &corev1.ServiceAccount{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: saName, Namespace: ns}, sa)).To(Succeed())
				names := make([]string, 0, len(sa.ImagePullSecrets))
				for _, r := range sa.ImagePullSecrets {
					names = append(names, r.Name)
				}
				if present {
					g.Expect(names).To(ContainElement(secretName))
				} else {
					g.Expect(names).NotTo(ContainElement(secretName))
				}
			}, "2m", "3s").Should(Succeed())
		}

		It("sets imagePullSecrets on component ServiceAccounts from an in-namespace labeled Secret", func() {
			ns, cleanupFunc := fixture.CreateRandomE2ETestNamespaceWithCleanupFunc()
			defer cleanupFunc()

			By("creating a labeled pull Secret before the ArgoCD instance")
			Expect(k8sClient.Create(ctx, newLabeledPullSecret("my-pull-secret", ns.Name))).To(Succeed())

			By("creating the ArgoCD instance")
			argoCD := &argov1beta1api.ArgoCD{
				ObjectMeta: metav1.ObjectMeta{Name: "example-argocd", Namespace: ns.Name},
			}
			Expect(k8sClient.Create(ctx, argoCD)).To(Succeed())
			Eventually(argoCD, "5m", "5s").Should(argocdFixture.BeAvailable())

			By("verifying server and application-controller ServiceAccounts carry the pull secret")
			serverSA := "example-argocd-" + common.ArgoCDServerComponent
			appCtrlSA := "example-argocd-" + common.ArgoCDApplicationControllerComponent
			expectSAPullSecret(serverSA, ns.Name, "my-pull-secret", true)
			expectSAPullSecret(appCtrlSA, ns.Name, "my-pull-secret", true)

			By("verifying the reference is stable")
			Consistently(func(g Gomega) {
				sa := &corev1.ServiceAccount{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: serverSA, Namespace: ns.Name}, sa)).To(Succeed())
				names := make([]string, 0, len(sa.ImagePullSecrets))
				for _, r := range sa.ImagePullSecrets {
					names = append(names, r.Name)
				}
				g.Expect(names).To(ContainElement("my-pull-secret"))
			}, "30s", "5s").Should(Succeed())
		})

		It("skips propagation when multiple labeled pull Secrets exist in the namespace", func() {
			ns, cleanupFunc := fixture.CreateRandomE2ETestNamespaceWithCleanupFunc()
			defer cleanupFunc()

			By("creating two labeled pull Secrets before the ArgoCD instance")
			Expect(k8sClient.Create(ctx, newLabeledPullSecret("pull-secret-1", ns.Name))).To(Succeed())
			Expect(k8sClient.Create(ctx, newLabeledPullSecret("pull-secret-2", ns.Name))).To(Succeed())

			By("creating the ArgoCD instance")
			argoCD := &argov1beta1api.ArgoCD{
				ObjectMeta: metav1.ObjectMeta{Name: "example-argocd", Namespace: ns.Name},
			}
			Expect(k8sClient.Create(ctx, argoCD)).To(Succeed())
			Eventually(argoCD, "5m", "5s").Should(argocdFixture.BeAvailable())

			By("verifying the server ServiceAccount carries neither pull secret")
			serverSA := "example-argocd-" + common.ArgoCDServerComponent
			Consistently(func(g Gomega) {
				sa := &corev1.ServiceAccount{}
				g.Expect(k8sClient.Get(ctx, types.NamespacedName{Name: serverSA, Namespace: ns.Name}, sa)).To(Succeed())
				names := make([]string, 0, len(sa.ImagePullSecrets))
				for _, r := range sa.ImagePullSecrets {
					names = append(names, r.Name)
				}
				g.Expect(names).NotTo(ContainElement("pull-secret-1"))
				g.Expect(names).NotTo(ContainElement("pull-secret-2"))
			}, "1m", "5s").Should(Succeed())
		})

		It("removes imagePullSecrets from ServiceAccounts when the labeled Secret is deleted", func() {
			ns, cleanupFunc := fixture.CreateRandomE2ETestNamespaceWithCleanupFunc()
			defer cleanupFunc()

			By("creating a labeled pull Secret before the ArgoCD instance")
			pullSecret := newLabeledPullSecret("my-pull-secret", ns.Name)
			Expect(k8sClient.Create(ctx, pullSecret)).To(Succeed())

			By("creating the ArgoCD instance")
			argoCD := &argov1beta1api.ArgoCD{
				ObjectMeta: metav1.ObjectMeta{Name: "example-argocd", Namespace: ns.Name},
			}
			Expect(k8sClient.Create(ctx, argoCD)).To(Succeed())
			Eventually(argoCD, "5m", "5s").Should(argocdFixture.BeAvailable())

			serverSA := "example-argocd-" + common.ArgoCDServerComponent
			expectSAPullSecret(serverSA, ns.Name, "my-pull-secret", true)

			By("deleting the labeled pull Secret")
			Expect(k8sClient.Delete(ctx, pullSecret)).To(Succeed())

			By("forcing a reconcile by annotating the ArgoCD CR")
			argocdFixture.Update(argoCD, func(ac *argov1beta1api.ArgoCD) {
				if ac.Annotations == nil {
					ac.Annotations = map[string]string{}
				}
				ac.Annotations["e2e.argoproj.io/trigger"] = "1"
			})

			By("verifying the pull secret is removed from the server ServiceAccount")
			expectSAPullSecret(serverSA, ns.Name, "my-pull-secret", false)
		})
	})
})
