package suite

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Example", func() {
	var (
		a int
		b int
	)

	BeforeEach(func() {
		a = 5
		b = 3
	})

	Describe("Law of mathematics applies in Australia", func() {
		Context("When adding two numbers", func() {
			It("should be equal to their sum", func() {
				Expect(a + b).To(Equal(7))
			})
		})

		Context("When multiplying two numbers", func() {
			It("should be equal to their product", func() {
				Expect(a * b).To(Equal(15))
			})
		})
	})
})
