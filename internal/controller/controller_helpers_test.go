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

package controller

import (
	"errors"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestControllerHelpers(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Helpers Unit Tests")
}

var _ = Describe("classifyError", func() {
	It("should return empty string for nil error", func() {
		result := classifyError(nil)
		Expect(result).To(Equal(""))
	})

	It("should classify server timeout as Transient", func() {
		err := k8serrors.NewServerTimeout(corev1.Resource("pod"), "get", 0)
		result := classifyError(err)
		Expect(result).To(Equal("Transient"))
	})

	It("should classify timeout as Transient", func() {
		err := k8serrors.NewTimeoutError("timeout", 0)
		result := classifyError(err)
		Expect(result).To(Equal("Transient"))
	})

	It("should classify service unavailable as Transient", func() {
		err := k8serrors.NewServiceUnavailable("service unavailable")
		result := classifyError(err)
		Expect(result).To(Equal("Transient"))
	})

	It("should classify invalid error as Terminal", func() {
		err := k8serrors.NewInvalid(corev1.SchemeGroupVersion.WithKind("Pod").GroupKind(), "test", nil)
		result := classifyError(err)
		Expect(result).To(Equal("Terminal"))
	})

	It("should classify bad request as Terminal", func() {
		err := k8serrors.NewBadRequest("bad request")
		result := classifyError(err)
		Expect(result).To(Equal("Terminal"))
	})

	It("should classify forbidden as Terminal", func() {
		err := k8serrors.NewForbidden(corev1.Resource("pod"), "test", errors.New("forbidden"))
		result := classifyError(err)
		Expect(result).To(Equal("Terminal"))
	})

	It("should classify unknown error as Retriable", func() {
		err := k8serrors.NewInternalError(errors.New("internal error"))
		result := classifyError(err)
		Expect(result).To(Equal("Retriable"))
	})

	It("should classify not found as Retriable", func() {
		err := k8serrors.NewNotFound(corev1.Resource("pod"), "test")
		result := classifyError(err)
		Expect(result).To(Equal("Retriable"))
	})
})

var _ = Describe("updateCondition", func() {
	It("should append condition when list is empty", func() {
		conditions := []metav1.Condition{}
		newCondition := metav1.Condition{
			Type:   ConditionTypeReady,
			Status: metav1.ConditionTrue,
		}

		result := updateCondition(conditions, newCondition)
		Expect(len(result)).To(Equal(1))
		Expect(result[0].Type).To(Equal(ConditionTypeReady))
	})

	It("should update existing condition", func() {
		conditions := []metav1.Condition{
			{
				Type:   ConditionTypeReady,
				Status: metav1.ConditionFalse,
			},
		}
		newCondition := metav1.Condition{
			Type:   ConditionTypeReady,
			Status: metav1.ConditionTrue,
		}

		result := updateCondition(conditions, newCondition)
		Expect(len(result)).To(Equal(1))
		Expect(result[0].Type).To(Equal(ConditionTypeReady))
		Expect(result[0].Status).To(Equal(metav1.ConditionTrue))
	})

	It("should append condition when type doesn't exist", func() {
		conditions := []metav1.Condition{
			{
				Type:   ConditionTypeReady,
				Status: metav1.ConditionTrue,
			},
		}
		newCondition := metav1.Condition{
			Type:   ConditionTypeGitCloned,
			Status: metav1.ConditionTrue,
		}

		result := updateCondition(conditions, newCondition)
		Expect(len(result)).To(Equal(2))
		Expect(result[1].Type).To(Equal(ConditionTypeGitCloned))
	})

	It("should preserve other conditions when updating one", func() {
		conditions := []metav1.Condition{
			{
				Type:   ConditionTypeReady,
				Status: metav1.ConditionFalse,
			},
			{
				Type:   ConditionTypeGitCloned,
				Status: metav1.ConditionTrue,
			},
		}
		newCondition := metav1.Condition{
			Type:   ConditionTypeReady,
			Status: metav1.ConditionTrue,
		}

		result := updateCondition(conditions, newCondition)
		Expect(len(result)).To(Equal(2))
		Expect(result[0].Type).To(Equal(ConditionTypeReady))
		Expect(result[0].Status).To(Equal(metav1.ConditionTrue))
		Expect(result[1].Type).To(Equal(ConditionTypeGitCloned))
		Expect(result[1].Status).To(Equal(metav1.ConditionTrue))
	})
})

var _ = Describe("parseQuantity", func() {
	It("should parse valid CPU quantity", func() {
		result := parseQuantity("100m")
		Expect(result.String()).To(Equal("100m"))
	})

	It("should parse valid memory quantity", func() {
		result := parseQuantity("1Gi")
		Expect(result.String()).To(Equal("1Gi"))
	})

	It("should parse valid integer quantity", func() {
		result := parseQuantity("2")
		Expect(result.String()).To(Equal("2"))
	})

	It("should handle invalid quantity gracefully", func() {
		// parseQuantity ignores errors, so invalid input returns zero
		result := parseQuantity("invalid")
		Expect(result.IsZero()).To(BeTrue())
	})

	It("should parse decimal quantities", func() {
		result := parseQuantity("1.5")
		Expect(result.String()).To(Equal("1500m"))
	})
})
