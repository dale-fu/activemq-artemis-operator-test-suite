package basic_test

import (
	"github.com/onsi/gomega"
	"github.com/rh-messaging/shipshape/pkg/framework/ginkgowrapper"
	"gitlab.cee.redhat.com/msgqe/openshift-broker-suite-golang/test"
	"testing"
)

func TestBasic(t *testing.T) {

	gomega.RegisterFailHandler(ginkgowrapper.Fail)
	test.Initialize(t, "basic", "Basic Suite")
}
