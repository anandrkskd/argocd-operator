package serviceaccount

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	matcher "github.com/onsi/gomega/types"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/argoproj-labs/argocd-operator/tests/ginkgo/fixture/utils"
)

func HaveImagePullSecret(secretName string) matcher.GomegaMatcher {
	return WithTransform(func(sa *corev1.ServiceAccount) bool {
		k8sClient, _ := utils.GetE2ETestKubeClient()

		err := k8sClient.Get(context.Background(), client.ObjectKeyFromObject(sa), sa)
		if err != nil {
			GinkgoWriter.Println("HaveImagePullSecret:", err)
			return false
		}

		for _, ref := range sa.ImagePullSecrets {
			if ref.Name == secretName {
				return true
			}
		}
		GinkgoWriter.Println("HaveImagePullSecret: secret", secretName, "not found in", sa.ImagePullSecrets)
		return false
	}, BeTrue())
}

func Exist() matcher.GomegaMatcher {
	return WithTransform(func(sa *corev1.ServiceAccount) bool {
		k8sClient, _ := utils.GetE2ETestKubeClient()

		err := k8sClient.Get(context.Background(), client.ObjectKeyFromObject(sa), sa)
		if err != nil {
			GinkgoWriter.Println("Exist:", err)
			return false
		}
		return true
	}, BeTrue())
}
