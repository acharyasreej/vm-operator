// Copyright (c) 2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package common_test

import (
	"net/http"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	vmopv1 "github.com/acharyasreej/vm-operator-api/api/v1alpha1"

	"github.com/acharyasreej/vm-operator/pkg/context"
	"github.com/acharyasreej/vm-operator/webhooks/common"
)

var _ = Describe("Validation Response", func() {

	var (
		gr  = schema.GroupResource{Group: vmopv1.SchemeGroupVersion.Group, Resource: "VirtualMachine"}
		ctx *context.WebhookRequestContext
	)

	BeforeEach(func() {
		ctx = &context.WebhookRequestContext{
			Logger: ctrllog.Log.WithName("validate-response"),
		}
	})

	When("No errors occur", func() {
		It("Returns allowed", func() {
			response := common.BuildValidationResponse(ctx, nil, nil)
			Expect(response.Allowed).To(BeTrue())
		})
	})

	When("Validation errors occur", func() {
		It("Returns denied", func() {
			validationErrs := []string{"this is required"}
			response := common.BuildValidationResponse(ctx, validationErrs, nil)
			Expect(response.Allowed).To(BeFalse())
			Expect(response.Result).ToNot(BeNil())
			Expect(response.Result.Code).To(Equal(int32(http.StatusUnprocessableEntity)))
			Expect(string(response.Result.Reason)).To(ContainSubstring(validationErrs[0]))
		})
	})

	Context("Returns denied for expected well-known errors", func() {

		wellKnownError := func(err error, expectedCode int) {
			response := common.BuildValidationResponse(ctx, nil, err)
			Expect(response.Allowed).To(BeFalse())
			Expect(response.Result).ToNot(BeNil())
			Expect(response.Result.Code).To(Equal(int32(expectedCode)))
		}

		DescribeTable("", wellKnownError,
			Entry("NotFound", apierrors.NewNotFound(gr, ""), http.StatusNotFound),
			Entry("Gone", apierrors.NewGone("gone"), http.StatusGone),
			Entry("ResourceExpired", apierrors.NewResourceExpired("expired"), http.StatusGone),
			Entry("ServiceUnavailable", apierrors.NewServiceUnavailable("unavailable"), http.StatusServiceUnavailable),
			Entry("ServiceUnavailable", apierrors.NewServiceUnavailable("unavailable"), http.StatusServiceUnavailable),
			Entry("Timeout", apierrors.NewTimeoutError("timeout", 42), http.StatusGatewayTimeout),
			Entry("Server Timeout", apierrors.NewServerTimeout(gr, "op", 42), http.StatusGatewayTimeout),
			Entry("Generic", apierrors.NewMethodNotSupported(gr, "op"), http.StatusInternalServerError),
		)
	})
})
