package v1_test

import (
	"log"
	"testing"

	"google.golang.org/grpc/grpclog"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestClientpool(t *testing.T) {
	log.SetOutput(GinkgoWriter)
	grpclog.SetLoggerV2(grpclog.NewLoggerV2(GinkgoWriter, GinkgoWriter, GinkgoWriter))

	RegisterFailHandler(Fail)
	RunSpecs(t, "Clientpool V1 Suite")
}
