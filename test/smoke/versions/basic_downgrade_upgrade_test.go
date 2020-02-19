package versions

import (
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/rh-messaging/shipshape/pkg/framework"
	"gitlab.cee.redhat.com/msgqe/openshift-broker-suite-golang/test"
)

var _ = ginkgo.Describe("DeploymentScalingBroker", func() {

	var (
		ctx1 *framework.ContextData
		dw = test.DeploymentWrapper{}.WithWait(true).WithBrokerClient(brokerClient).WithContext(ctx1).WithCustomImage(test.BrokerImageName)
	)

	// Initialize after framework has been created
	ginkgo.JustBeforeEach(func() {
		ctx1 = Framework.GetFirstContext()
		dw = test.DeploymentWrapper{}.WithWait(true).WithBrokerClient(brokerClient).WithContext(ctx1).WithCustomImage(test.BrokerImageName)
	})

	ginkgo.It("Deploy broker and downgrade it to another version", func() {
		gomega.Expect(dw.DeployBrokers(1)).To(gomega.BeNil())
		// Check for jolokia call for version, curl from pod
		gomega.Expect(dw.WithCustomImage(test.BrokerImageNameOld).ChangeImage())
	})

	ginkgo.It("Deploy broker and upgrade it to another version", func() {
		gomega.Expect(dw.WithCustomImage(test.BrokerImageNameOld).DeployBrokers( 1)).To(gomega.BeNil())
		// Check for jolokia call for version, curl from pod
		gomega.Expect(dw.WithCustomImage(test.BrokerImageName).ChangeImage())
	})

	ginkgo.It("Deploy broker and upgrade it to another version", func() {

	})
})
