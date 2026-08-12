// Copyright 2025 ArgoCD Operator Developers
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// 	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package argoutil

import (
	"fmt"
	"os"
	"strings"
)

func IsNamespaceClusterConfigNamespace(ns string) bool {
	return allowedNamespace(ns, os.Getenv("ARGOCD_CLUSTER_CONFIG_NAMESPACES"))
}

func allowedNamespace(current string, namespaces string) bool {
	clusterConfigNamespaces := splitList(namespaces)
	if len(clusterConfigNamespaces) > 0 {
		if clusterConfigNamespaces[0] == "*" {
			return true
		}

		for _, n := range clusterConfigNamespaces {
			if n == current {
				return true
			}
		}
	}
	return false
}

// OperatorNamespaceFile is the path to the in-cluster service account namespace file.
var OperatorNamespaceFile = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

func GetOperatorNamespace() (string, error) {
	data, err := os.ReadFile(OperatorNamespaceFile)
	if err != nil {
		return "", fmt.Errorf("failed to read operator namespace: %w", err)
	}
	ns := strings.TrimSpace(string(data))
	if ns == "" {
		return "", fmt.Errorf("operator namespace file is empty")
	}
	return ns, nil
}

func splitList(s string) []string {
	elems := strings.Split(s, ",")
	for i := range elems {
		elems[i] = strings.TrimSpace(elems[i])
	}
	return elems
}
