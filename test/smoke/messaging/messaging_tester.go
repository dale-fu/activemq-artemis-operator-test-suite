package messaging

import (
	"github.com/onsi/gomega"
	"github.com/rh-messaging/shipshape/pkg/framework/log"
	"gitlab.cee.redhat.com/msgqe/openshift-broker-suite-golang/test"
)

func testBaseSendReceiveMessages(bdw *test.BrokerDeploymentWrapper,
	srw *test.SenderReceiverWrapper,
	MessageCount int, MessageBody string,
	acceptorType test.AcceptorType, BrokerCount int, protocol string) {
	testBaseSendReceiveMessagesWithCallback(bdw, srw, MessageCount, MessageBody, acceptorType, BrokerCount, protocol, nil)
}

func testBaseSendReceiveMessagesWithCallback(bdw *test.BrokerDeploymentWrapper,
	srw *test.SenderReceiverWrapper,
	MessageCount int, MessageBody string,
	acceptorType test.AcceptorType, BrokerCount int, protocol string, callback test.SenderReceiverCallback) {
	err := bdw.DeployBrokersWithAcceptor(BrokerCount, acceptorType)
	gomega.Expect(err).To(gomega.BeNil())
	sender, receiver := srw.PrepareSenderReceiverWithProtocol(protocol)
	_, err = test.SendReceiveMessages(sender, receiver, callback)
	gomega.Expect(err).To(gomega.BeNil())
	senderResult := sender.Result()
	receiverResult := receiver.Result()
	gomega.Expect(senderResult.Delivered).To(gomega.Equal(MessageCount))
	gomega.Expect(receiverResult.Delivered).To(gomega.Equal(MessageCount))

	log.Logf("MessageCount is fine")
	for _, msg := range receiverResult.Messages {
		gomega.Expect(msg.Content).To(gomega.Equal(MessageBody))
	}
}

func testBaseSendMessages(bdw *test.BrokerDeploymentWrapper,
	srw *test.SenderReceiverWrapper,
	MessageCount int, MessageBody string,
	acceptorType test.AcceptorType, BrokerCount int, protocol, senderName string, callback test.SenderReceiverCallback) {
	err := bdw.DeployBrokersWithAcceptor(BrokerCount, acceptorType)
	gomega.Expect(err).To(gomega.BeNil())
	sender := srw.PrepareNamedSenderWithProtocol(senderName, protocol)
	_, err = test.SendMessages(sender, callback)
	gomega.Expect(err).To(gomega.BeNil())
	senderResult := sender.Result()
	gomega.Expect(senderResult.Delivered).To(gomega.Equal(MessageCount))
	log.Logf("MessageCount is fine")
}

func testBaseReceiveMessages(bdw *test.BrokerDeploymentWrapper,
	srw *test.SenderReceiverWrapper,
	MessageCount int, MessageBody string,
	protocol string) {
	receiver := srw.PrepareReceiverWithProtocol(protocol)
	err := test.ReceiveMessages(receiver)
	gomega.Expect(err).To(gomega.BeNil())
	receiverResult := receiver.Result()
	gomega.Expect(receiverResult.Delivered).To(gomega.Equal(MessageCount))
	log.Logf("MessageCount is fine")
	for _, msg := range receiverResult.Messages {
		gomega.Expect(msg.Content).To(gomega.Equal(MessageBody))
	}
}
