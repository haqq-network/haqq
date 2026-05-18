package keeper_test

import (
	. "github.com/onsi/ginkgo/v2"
)

var _ = Describe("Integration test", Label("Ethiq module"), Ordered, func() {
	var s *IntegrationTestSuite
	BeforeEach(func() {
		s = new(IntegrationTestSuite)
		s.SetupTest()
	})

	Context("context", func() {
		It("test case", func() {
			Skip("TODO implement if necessary")
		})
	})
})
