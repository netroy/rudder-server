package warehouseutils_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	testutils "github.com/rudderlabs/rudder-server/utils/tests"
)

func TestUtils(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecsWithDefaultAndCustomReporters(t, "Warehouse Utils Suite", []Reporter{testutils.NewJUnitReporter()})
}
