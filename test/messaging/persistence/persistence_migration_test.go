package persistence

import (
	"github.com/artemiscloud/activemq-artemis-operator-test-suite/pkg/bdw"
	"github.com/artemiscloud/activemq-artemis-operator-test-suite/test"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"github.com/rh-messaging/shipshape/pkg/api/client/amqp"
	"github.com/rh-messaging/shipshape/pkg/framework"
)

var _ = ginkgo.Describe("MessagingMigrationTests", func() {

	var (
		ctx1           *framework.ContextData
		brokerDeployer *bdw.BrokerDeploymentWrapper
		srw            *test.SenderReceiverWrapper
		sender         amqp.Client
		receiver       amqp.Client
	)

	const (
		MessageBody   = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		MessageCount  = 10
		Port          = "5672"
		Domain        = "svc.cluster.local"
		SubdomainName = "-hdls-svc"
		AddressBit    = "someQueue"
		Protocol      = test.AMQP
	)

	// PrepareNamespace after framework has been created.
	ginkgo.JustBeforeEach(func() {
		ctx1 = sw.Framework.GetFirstContext()
		brokerDeployer = &bdw.BrokerDeploymentWrapper{}
		setEnv(ctx1, brokerDeployer)
		srw = &test.SenderReceiverWrapper{}
		srw.WithContext(ctx1).
			WithMessageBody(MessageBody).
			WithMessageCount(MessageCount)
	})

	ginkgo.JustAfterEach(func() {
		sw.Framework.GetFirstContext().EventHandler.ClearCallbacks()
	})

	// This test might fail due to ENTMQBR-3597
	ginkgo.It("Deploy double broker instance, migrate to single", func() {
		err := brokerDeployer.DeployBrokers(2)
		gomega.Expect(err).To(gomega.BeNil(), "Broker deployment failed: %s", err)

		sendUrl := test.FormUrl(Protocol, DeployName, "0", SubdomainName, ctx1.Namespace, Domain, AddressBit, Port)
		receiveUrl := test.FormUrl(Protocol, DeployName, "0", SubdomainName, ctx1.Namespace, Domain, AddressBit, Port)
		sender, receiver := srw.
			WithReceiveUrl(receiveUrl).
			WithSendUrl(sendUrl).
			PrepareSenderReceiverWithProtocol(test.AMQP)

		callback := func() (interface{}, error) {
			senderResult := sender.Result()
			gomega.Expect(senderResult.Delivered).To(gomega.Equal(MessageCount), "Delivered %d messages, expected %d", senderResult.Delivered, MessageCount)
			_ = brokerDeployer.Scale(1)
			drainerCompleted := test.WaitForDrainerRemoval(sw, 1)
			gomega.Expect(drainerCompleted).To(gomega.BeTrue(), "Drainer completion not detected")
			return drainerCompleted, nil
		}
		_, err = test.SendReceiveMessages(sender, receiver, callback)
		gomega.Expect(err).To(gomega.BeNil(), "Sending/receiving messages failed: %s", err)

		receiverResult := receiver.Result()

		for _, msg := range receiverResult.Messages {
			gomega.Expect(msg.Content).To(gomega.Equal(MessageBody), "MessageBody corrupted: expected %s, received %s", MessageBody, msg.Content)
		}

	})

	// This test might fail due to ENTMQBR-3597
	ginkgo.It("Deploy 4 brokers, migrate everything to single", func() {
		podNumbers := []string{"3", "2", "1"}

		err := brokerDeployer.DeployBrokers(4)
		gomega.Expect(err).To(gomega.BeNil(), "Broker deployment failed: %s", err)
		for _, number := range podNumbers {
			url := test.FormUrl(Protocol, DeployName, number, SubdomainName, ctx1.Namespace, Domain, AddressBit, Port)
			sender = srw.WithSendUrl(url).PrepareNamedSender("sender-" + string(number))
			_ = sender.Deploy()
			sender.Wait()
		}

		err = brokerDeployer.Scale(1)
		gomega.Expect(err).To(gomega.BeNil(), "Broker scaling to single instance failed")
		drainerCompleted := test.WaitForDrainerRemoval(sw, 3)
		gomega.Expect(drainerCompleted).To(gomega.BeTrue(), "Drainers have not been completed")
		receiveUrl := test.FormUrl(Protocol, DeployName, "0", SubdomainName, ctx1.Namespace, Domain, AddressBit, Port)
		receiver = srw.
			WithReceiveUrl(receiveUrl).
			WithReceiverCount(len(podNumbers) * MessageCount).
			PrepareReceiver()

		err = receiver.Deploy()
		gomega.Expect(err).To(gomega.BeNil(), "Receiver deployment failed: %s", err)
		receiver.Wait()
		receiverResult := receiver.Result()
		expectedCount := MessageCount * len(podNumbers)
		gomega.Expect(receiverResult.Delivered).To(gomega.Equal(expectedCount), "Message migration not completed. Expected %d messages, received %d.", receiverResult.Delivered, expectedCount)
		for _, msg := range receiverResult.Messages {
			gomega.Expect(msg.Content).To(gomega.Equal(MessageBody), "Message body corrupted: expected: %s, real: %s", MessageBody, msg.Content)
		}
	})

	// This test might fail due to ENTMQBR-3597
	ginkgo.It("Deploy 4 brokers, migrate last one", func() {
		sendUrl := test.FormUrl(Protocol, DeployName, "3", SubdomainName, ctx1.Namespace, Domain, AddressBit, Port)
		receiveUrl := test.FormUrl(Protocol, DeployName, "0", SubdomainName, ctx1.Namespace, Domain, AddressBit, Port)

		sender, receiver = srw.
			WithReceiveUrl(receiveUrl).
			WithSendUrl(sendUrl).
			PrepareSenderReceiver()
		err := brokerDeployer.DeployBrokers(4)

		gomega.Expect(err).To(gomega.BeNil(), "Broker deployment failed: %s", err)
		err = sender.Deploy()
		gomega.Expect(err).To(gomega.BeNil(), "Sender deployment failed: %s", err)
		sender.Wait()
		err = brokerDeployer.Scale(3)
		gomega.Expect(err).To(gomega.BeNil(), "Brokre upscaling failed: %s", err)
		drainerCompleted := test.WaitForDrainerRemoval(sw, 1)
		gomega.Expect(drainerCompleted).To(gomega.BeTrue(), "Drainer completion not detected")

		err = receiver.Deploy()
		gomega.Expect(err).To(gomega.BeNil(), "Receiver deployment failed :%s", err)
		receiver.Wait()
		receiverResult := receiver.Result()
		gomega.Expect(receiverResult.Delivered).To(gomega.Equal(MessageCount), "MessageCount: expected %d, actual %d", MessageCount, receiverResult.Delivered)

		for _, msg := range receiverResult.Messages {
			gomega.Expect(msg.Content).To(gomega.Equal(MessageBody), "MessageBody corrupted: expected %s, real %s", msg.Content, MessageBody)
		}
	})

	// TODO: redesign mass migration test to be actually able to run it with giant message sizes and message quantities
})
