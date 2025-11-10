package util_test

import (
	"com/realworld/ginrxgogorm/util"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gorm.io/gorm"
)

var _ = Describe("Testing End Points", Ordered, func() {
	Context("FormatTimestamp", func() {
		DescribeTable("testing ...",
			func(in gorm.Model, oExpect string) {
				Expect(util.FormatTimestamp(in.CreatedAt)).To(Equal(oExpect))
			},
			Entry("empty timestamp", gorm.Model{}, ""),
			Entry("2 timestamp", gorm.Model{}, ""),
		)
	})
})
