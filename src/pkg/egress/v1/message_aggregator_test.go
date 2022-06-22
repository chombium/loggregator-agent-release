package v1_test

import (
	"time"

	egress "code.cloudfoundry.org/loggregator-agent-release/src/pkg/egress/v1"
	"github.com/cloudfoundry/sonde-go/events"
	"google.golang.org/protobuf/proto"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("MessageAggregator", func() {
	var (
		mockWriter        *mockEnvelopeWriter
		messageAggregator *egress.MessageAggregator
		originalTTL       time.Duration
	)

	BeforeEach(func() {
		mockWriter = newMockEnvelopeWriter()
		messageAggregator = egress.NewAggregator(
			mockWriter,
		)
		originalTTL = egress.MaxTTL
	})

	AfterEach(func() {
		egress.MaxTTL = originalTTL
	})

	It("passes value messages through", func() {
		inputMessage := createValueMessage()
		messageAggregator.Write(inputMessage)

		Expect(mockWriter.WriteInput.Event).To(HaveLen(1))
		Expect(<-mockWriter.WriteInput.Event).To(Equal(inputMessage))
	})

	It("handles concurrent writes without data race", func() {
		inputMessage := createValueMessage()
		done := make(chan struct{})
		go func() {
			defer close(done)
			for i := 0; i < 20; i++ {
				messageAggregator.Write(inputMessage)
			}
		}()
		for i := 0; i < 20; i++ {
			messageAggregator.Write(inputMessage)
		}
		<-done
	})

	Describe("counter processing", func() {
		It("sets the Total field on a CounterEvent ", func() {
			messageAggregator.Write(createCounterMessage("total", "fake-origin-4", nil))

			Expect(mockWriter.WriteInput.Event).To(HaveLen(1))
			outputMessage := <-mockWriter.WriteInput.Event
			Expect(outputMessage.GetEventType()).To(Equal(events.Envelope_CounterEvent))
			expectCorrectCounterNameDeltaAndTotal(outputMessage, "total", 4, 4)
		})

		It("accumulates Deltas for CounterEvents with the same name, origin, and tags", func() {
			messageAggregator.Write(createCounterMessage(
				"total",
				"fake-origin-4",
				map[string]string{
					"protocol": "tcp",
				},
			))
			messageAggregator.Write(createCounterMessage(
				"total",
				"fake-origin-4",
				map[string]string{
					"protocol": "tcp",
				},
			))
			messageAggregator.Write(createCounterMessage(
				"total",
				"fake-origin-4",
				map[string]string{
					"protocol": "tcp",
				},
			))

			Expect(mockWriter.WriteInput.Event).To(HaveLen(3))
			e := <-mockWriter.WriteInput.Event
			expectCorrectCounterNameDeltaAndTotal(e, "total", 4, 4)
			e = <-mockWriter.WriteInput.Event
			expectCorrectCounterNameDeltaAndTotal(e, "total", 4, 8)
			e = <-mockWriter.WriteInput.Event
			expectCorrectCounterNameDeltaAndTotal(e, "total", 4, 12)
		})

		It("overwrites aggregated total when total is set", func() {
			messageAggregator.Write(createCounterMessage(
				"total",
				"fake-origin-4",
				map[string]string{
					"protocol": "tcp",
				},
			))
			messageAggregator.Write(createCounterMessageWithTotal(
				"total",
				"fake-origin-4",
				map[string]string{
					"protocol": "tcp",
				},
			))
			messageAggregator.Write(createCounterMessage(
				"total",
				"fake-origin-4",
				map[string]string{
					"protocol": "tcp",
				},
			))

			Expect(mockWriter.WriteInput.Event).To(HaveLen(3))
			e := <-mockWriter.WriteInput.Event
			expectCorrectCounterNameDeltaAndTotal(e, "total", 4, 4)
			e = <-mockWriter.WriteInput.Event
			expectCorrectCounterNameDeltaAndTotal(e, "total", 0, 101)
			e = <-mockWriter.WriteInput.Event
			expectCorrectCounterNameDeltaAndTotal(e, "total", 4, 105)
		})

		It("accumulates differently-named counters separately", func() {
			messageAggregator.Write(createCounterMessage("total1", "fake-origin-4", nil))
			messageAggregator.Write(createCounterMessage("total2", "fake-origin-4", nil))

			Expect(mockWriter.WriteInput.Event).To(HaveLen(2))
			e := <-mockWriter.WriteInput.Event
			expectCorrectCounterNameDeltaAndTotal(e, "total1", 4, 4)
			e = <-mockWriter.WriteInput.Event
			expectCorrectCounterNameDeltaAndTotal(e, "total2", 4, 4)
		})

		It("accumulates differently-tagged counters separately", func() {
			By("writing protocol tagged counters")
			messageAggregator.Write(createCounterMessage(
				"total",
				"fake-origin-4",
				map[string]string{
					"protocol": "grpc",
				},
			))
			messageAggregator.Write(createCounterMessage(
				"total",
				"fake-origin-4",
				map[string]string{
					"protocol": "tcp",
				},
			))
			messageAggregator.Write(createCounterMessage(
				"total",
				"fake-origin-4",
				map[string]string{
					"protocol": "grpc",
				},
			))

			By("writing counters tagged with key/value strings split differently")
			messageAggregator.Write(createCounterMessage(
				"total",
				"fake-origin-4",
				map[string]string{
					"proto": "other",
				},
			))

			Expect(mockWriter.WriteInput.Event).To(HaveLen(4))
			expectCorrectCounterNameDeltaAndTotal(<-mockWriter.WriteInput.Event, "total", 4, 4)
			expectCorrectCounterNameDeltaAndTotal(<-mockWriter.WriteInput.Event, "total", 4, 4)
			expectCorrectCounterNameDeltaAndTotal(<-mockWriter.WriteInput.Event, "total", 4, 8)
			expectCorrectCounterNameDeltaAndTotal(<-mockWriter.WriteInput.Event, "total", 4, 4)
		})

		It("does not accumulate for counters when receiving a non-counter event", func() {
			messageAggregator.Write(createValueMessage())
			messageAggregator.Write(createCounterMessage("counter1", "fake-origin-4", nil))

			Expect(mockWriter.WriteInput.Event).To(HaveLen(2))
			e := <-mockWriter.WriteInput.Event
			Expect(e.GetEventType()).To(Equal(events.Envelope_ValueMetric))
			e = <-mockWriter.WriteInput.Event
			Expect(e.GetEventType()).To(Equal(events.Envelope_CounterEvent))
			expectCorrectCounterNameDeltaAndTotal(e, "counter1", 4, 4)
		})

		It("accumulates independently for different origins", func() {
			messageAggregator.Write(createCounterMessage("counter1", "fake-origin-4", nil))
			messageAggregator.Write(createCounterMessage("counter1", "fake-origin-5", nil))
			messageAggregator.Write(createCounterMessage("counter1", "fake-origin-4", nil))

			Expect(mockWriter.WriteInput.Event).To(HaveLen(3))

			e := <-mockWriter.WriteInput.Event
			Expect(e.GetOrigin()).To(Equal("fake-origin-4"))
			expectCorrectCounterNameDeltaAndTotal(e, "counter1", 4, 4)

			e = <-mockWriter.WriteInput.Event
			Expect(e.GetOrigin()).To(Equal("fake-origin-5"))
			expectCorrectCounterNameDeltaAndTotal(e, "counter1", 4, 4)

			e = <-mockWriter.WriteInput.Event
			Expect(e.GetOrigin()).To(Equal("fake-origin-4"))
			expectCorrectCounterNameDeltaAndTotal(e, "counter1", 4, 8)
		})
	})
})

func createValueMessage() *events.Envelope {
	return &events.Envelope{
		Origin:    proto.String("fake-origin-2"),
		EventType: events.Envelope_ValueMetric.Enum(),
		ValueMetric: &events.ValueMetric{
			Name:  proto.String("fake-metric-name"),
			Value: proto.Float64(42),
			Unit:  proto.String("fake-unit"),
		},
	}
}

func createCounterMessage(name, origin string, tags map[string]string) *events.Envelope {
	return &events.Envelope{
		Origin:    proto.String(origin),
		EventType: events.Envelope_CounterEvent.Enum(),
		CounterEvent: &events.CounterEvent{
			Name:  proto.String(name),
			Delta: proto.Uint64(4),
		},
		Tags: tags,
	}
}

func createCounterMessageWithTotal(name, origin string, tags map[string]string) *events.Envelope {
	return &events.Envelope{
		Origin:    proto.String(origin),
		EventType: events.Envelope_CounterEvent.Enum(),
		CounterEvent: &events.CounterEvent{
			Name:  proto.String(name),
			Total: proto.Uint64(101),
		},
		Tags: tags,
	}
}

func expectCorrectCounterNameDeltaAndTotal(outputMessage *events.Envelope, name string, delta uint64, total uint64) {
	Expect(outputMessage.GetCounterEvent().GetName()).To(Equal(name))
	Expect(outputMessage.GetCounterEvent().GetDelta()).To(Equal(delta))
	Expect(outputMessage.GetCounterEvent().GetTotal()).To(Equal(total))
}
