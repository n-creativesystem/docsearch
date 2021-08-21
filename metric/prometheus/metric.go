package prometheus

import (
	"context"

	grpcprometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
)

const (
	namespace     = "docsearch"
	raftSubSystem = "raft"
)

type strings []string

func (s strings) Add(value string) strings {
	s = append(s, value)
	return s
}

func (s strings) copy() []string {
	c := make([]string, len(s))
	copy(c, s)
	return c
}

var (
	baseLabelNames = strings([]string{"id"})
	Registry       = prometheus.NewRegistry()
	gRPCMetrics    = grpcprometheus.NewServerMetrics(func(co *prometheus.CounterOpts) {
		co.Namespace = namespace
	})
	RaftStateMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: raftSubSystem,
		Name:      "state",
		Help:      "Node state. 0: Follower, 1:Candidate, 2:Leader, 3:Shutdown",
	}, baseLabelNames.copy())
	RaftTermMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: raftSubSystem,
		Name:      "term",
		Help:      "Term",
	}, baseLabelNames.copy())
	RaftLastLogIndexMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: raftSubSystem,
		Name:      "last_log_index",
		Help:      "Last log index.",
	}, baseLabelNames.copy())

	RaftLastLogTermMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: raftSubSystem,
		Name:      "last_log_term",
		Help:      "Last log term.",
	}, baseLabelNames.copy())

	RaftCommitIndexMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: raftSubSystem,
		Name:      "commit_index",
		Help:      "Commit index.",
	}, baseLabelNames.copy())

	RaftAppliedIndexMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: raftSubSystem,
		Name:      "applied_index",
		Help:      "Applied index.",
	}, baseLabelNames.copy())

	RaftFsmPendingMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: raftSubSystem,
		Name:      "fsm_pending",
		Help:      "FSM pending.",
	}, baseLabelNames.copy())

	RaftLastSnapshotIndexMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: raftSubSystem,
		Name:      "last_snapshot_index",
		Help:      "Last snapshot index.",
	}, baseLabelNames.copy())

	RaftLastSnapshotTermMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: raftSubSystem,
		Name:      "last_snapshot_term",
		Help:      "Last snapshot term.",
	}, baseLabelNames.copy())

	RaftLatestConfigurationIndexMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: raftSubSystem,
		Name:      "latest_configuration_index",
		Help:      "Latest configuration index.",
	}, baseLabelNames.copy())

	RaftNumPeersMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: raftSubSystem,
		Name:      "num_peers",
		Help:      "Number of peers.",
	}, baseLabelNames.copy())

	RaftLastContactMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: raftSubSystem,
		Name:      "last_copntact",
		Help:      "Last contact.",
	}, baseLabelNames.copy())

	RaftNumNodesMetric = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: raftSubSystem,
		Name:      "num_nodes",
		Help:      "Number of nodes.",
	}, baseLabelNames.copy())
)

func init() {
	Registry.MustRegister(
		gRPCMetrics,
		RaftStateMetric,
		RaftTermMetric,
		RaftLastLogIndexMetric,
		RaftLastLogTermMetric,
		RaftCommitIndexMetric,
		RaftAppliedIndexMetric,
		RaftFsmPendingMetric,
		RaftLastSnapshotIndexMetric,
		RaftLastSnapshotTermMetric,
		RaftLatestConfigurationIndexMetric,
		RaftNumPeersMetric,
		RaftLastContactMetric,
		RaftNumNodesMetric,
	)
	gRPCMetrics.EnableHandlingTimeHistogram(func(ho *prometheus.HistogramOpts) {
		ho.Namespace = namespace
	})
}

func StreamServerInterceptor() func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	return gRPCMetrics.StreamServerInterceptor()
}

func UnaryServerInterceptor() func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return gRPCMetrics.UnaryServerInterceptor()
}
