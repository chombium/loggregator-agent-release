package bindings_test

import (
	"errors"

	"code.cloudfoundry.org/loggregator-agent-release/src/pkg/binding"
	"code.cloudfoundry.org/loggregator-agent-release/src/pkg/egress/syslog"
	"code.cloudfoundry.org/loggregator-agent-release/src/pkg/ingress/bindings"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Aggregate Drain Binding Fetcher", func() {
	var ()

	BeforeEach(func() {
	})

	Context("cache fetcher is nil", func() {
		It("returns drain bindings for the drain urls", func() {
			bs := []string{
				"syslog://aggregate-drain1.url.com",
				"syslog://aggregate-drain2.url.com",
			}
			fetcher := bindings.NewAggregateDrainFetcher(bs, nil)

			b, err := fetcher.FetchBindings()
			Expect(err).ToNot(HaveOccurred())

			Expect(b).To(ConsistOf(
				syslog.Binding{
					AppId: "",
					Drain: syslog.Drain{Url: "syslog://aggregate-drain1.url.com"},
				},
				syslog.Binding{
					AppId: "",
					Drain: syslog.Drain{Url: "syslog://aggregate-drain2.url.com"},
				},
			))
		})
	})
	Context("cache fetcher exists", func() {
		It("ignores fetcher if both are available", func() {
			bs := []string{
				"syslog://aggregate-drain1.url.com",
				"syslog://aggregate-drain2.url.com",
			}
			cacheFetcher := mockCacheFetcher{}
			fetcher := bindings.NewAggregateDrainFetcher(bs, &cacheFetcher)

			b, err := fetcher.FetchBindings()
			Expect(err).ToNot(HaveOccurred())

			Expect(b).To(ConsistOf(
				syslog.Binding{
					AppId: "",
					Drain: syslog.Drain{Url: "syslog://aggregate-drain1.url.com"},
				},
				syslog.Binding{
					AppId: "",
					Drain: syslog.Drain{Url: "syslog://aggregate-drain2.url.com"},
				},
			))
		})
		It("returns results from cache", func() {
			bs := []string{""}
			cacheFetcher := mockCacheFetcher{bindings: []binding.Binding{
				{
					Url: "syslog://aggregate-drain1.url.com",
					Credentials: []binding.Credentials{
						{
							Cert: "cert",
							Key:  "key",
							CA:   "ca",
						},
					},
				},
				{
					Url: "syslog://aggregate-drain2.url.com",
					Credentials: []binding.Credentials{
						{
							Cert: "cert2",
							Key:  "key2",
							CA:   "ca2",
						},
					},
				},
			}}
			fetcher := bindings.NewAggregateDrainFetcher(bs, &cacheFetcher)

			b, err := fetcher.FetchBindings()
			Expect(err).ToNot(HaveOccurred())

			Expect(b).To(ConsistOf(
				syslog.Binding{
					AppId: "",
					Drain: syslog.Drain{
						Url: "syslog://aggregate-drain1.url.com",
						Credentials: syslog.Credentials{
							Cert: "cert",
							Key:  "key",
							CA:   "ca",
						},
					},
				},
				syslog.Binding{
					AppId: "",
					Drain: syslog.Drain{
						Url: "syslog://aggregate-drain2.url.com",
						Credentials: syslog.Credentials{
							Cert: "cert2",
							Key:  "key2",
							CA:   "ca2",
						},
					},
				},
			))
		})
		It("ignores empty urls", func() {
			bs := []string{""}
			cacheFetcher := mockCacheFetcher{bindings: []binding.Binding{
				{
					Url: "syslog://aggregate-drain1.url.com",
					Credentials: []binding.Credentials{
						{
							Cert: "cert",
							Key:  "key",
							CA:   "ca",
						},
					},
				},
				{
					Url: "",
				},
			}}
			fetcher := bindings.NewAggregateDrainFetcher(bs, &cacheFetcher)

			b, err := fetcher.FetchBindings()
			Expect(err).ToNot(HaveOccurred())

			Expect(b).To(ConsistOf(
				syslog.Binding{
					AppId: "",
					Drain: syslog.Drain{
						Url: "syslog://aggregate-drain1.url.com",
						Credentials: syslog.Credentials{
							Cert: "cert",
							Key:  "key",
							CA:   "ca",
						},
					},
				},
			))
		})
		It("returns error if fetching fails", func() {
			bs := []string{""}
			cacheFetcher := mockCacheFetcher{err: errors.New("error")}
			fetcher := bindings.NewAggregateDrainFetcher(bs, &cacheFetcher)

			_, err := fetcher.FetchBindings()
			Expect(err).To(MatchError("error"))
		})
	})
})

type mockCacheFetcher struct {
	bindings []binding.Binding
	err      error
}

func (m *mockCacheFetcher) GetAggregate() ([]binding.Binding, error) {
	return m.bindings, m.err
}
