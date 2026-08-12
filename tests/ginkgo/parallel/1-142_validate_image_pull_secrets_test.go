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
	"os"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	argov1beta1api "github.com/argoproj-labs/argocd-operator/api/v1beta1"
	"github.com/argoproj-labs/argocd-operator/tests/ginkgo/fixture"
	argocdFixture "github.com/argoproj-labs/argocd-operator/tests/ginkgo/fixture/argocd"
	secretFixture "github.com/argoproj-labs/argocd-operator/tests/ginkgo/fixture/secret"
	saFixture "github.com/argoproj-labs/argocd-operator/tests/ginkgo/fixture/serviceaccount"
	fixtureUtils "github.com/argoproj-labs/argocd-operator/tests/ginkgo/fixture/utils"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("GitOps Operator Parallel E2E Tests", func() {

	Context("1-142_validate_image_pull_secrets", func() {

		var (
			ctx       context.Context
			k8sClient client.Client
		)

		BeforeEach(func() {
			fixture.EnsureParallelCleanSlate()
			k8sClient, _ = fixtureUtils.GetE2ETestKubeClient()
			ctx = context.Background()
		})

		It("validates that all ArgoCD SAs have imagePullSecrets when env var is set", func() {
			secretName := os.Getenv("IMAGE_PULL_SECRETS")
			if secretName == "" {
				Skip("IMAGE_PULL_SECRETS not set")
			}

			By("creating new namespace-scoped Argo CD instance")
			randomNS, cleanupFunc := fixture.CreateRandomE2ETestNamespaceWithCleanupFunc()
			defer cleanupFunc()

			argoCD := &argov1beta1api.ArgoCD{
				ObjectMeta: metav1.ObjectMeta{Name: "argocd", Namespace: randomNS.Name},
			}
			Expect(k8sClient.Create(ctx, argoCD)).To(Succeed())

			By("waiting for ArgoCD to be available")
			Eventually(argoCD, "5m", "5s").Should(argocdFixture.HavePhase("Available"))

			By("verifying imagePullSecrets on all component SAs")
			componentSAs := []string{
				"argocd-argocd-application-controller",
				"argocd-argocd-server",
				"argocd-argocd-dex-server",
				"argocd-argocd-redis",
				"argocd-argocd-repo-server",
			}

			for _, saName := range componentSAs {
				sa := &corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{Name: saName, Namespace: randomNS.Name},
				}
				Eventually(sa, "2m", "5s").Should(saFixture.HaveImagePullSecret(secretName))
			}
		})

		It("validates that image pull secrets are copied to ArgoCD namespace", func() {
			secretNames := os.Getenv("IMAGE_PULL_SECRETS")
			if secretNames == "" {
				Skip("IMAGE_PULL_SECRETS not set")
			}

			By("creating new namespace-scoped Argo CD instance")
			randomNS, cleanupFunc := fixture.CreateRandomE2ETestNamespaceWithCleanupFunc()
			defer cleanupFunc()

			argoCD := &argov1beta1api.ArgoCD{
				ObjectMeta: metav1.ObjectMeta{Name: "argocd", Namespace: randomNS.Name},
			}
			Expect(k8sClient.Create(ctx, argoCD)).To(Succeed())

			By("waiting for ArgoCD to be available")
			Eventually(argoCD, "5m", "5s").Should(argocdFixture.HavePhase("Available"))

			By("verifying image pull secrets are copied to ArgoCD namespace")
			for _, name := range strings.Split(secretNames, ",") {
				name = strings.TrimSpace(name)
				if name == "" {
					continue
				}
				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: randomNS.Name},
				}
				Eventually(secret, "2m", "5s").Should(secretFixture.Exist())
			}
		})

		It("validates repo-server uses dedicated SA", func() {

			By("creating new namespace-scoped Argo CD instance")
			randomNS, cleanupFunc := fixture.CreateRandomE2ETestNamespaceWithCleanupFunc()
			defer cleanupFunc()

			argoCD := &argov1beta1api.ArgoCD{
				ObjectMeta: metav1.ObjectMeta{Name: "argocd", Namespace: randomNS.Name},
			}
			Expect(k8sClient.Create(ctx, argoCD)).To(Succeed())

			By("waiting for ArgoCD to be available")
			Eventually(argoCD, "5m", "5s").Should(argocdFixture.HavePhase("Available"))

			By("verifying repo-server SA exists")
			repoSA := &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{Name: "argocd-argocd-repo-server", Namespace: randomNS.Name},
			}
			Eventually(repoSA, "2m", "5s").Should(saFixture.Exist())
		})
	})
})
