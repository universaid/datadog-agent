package flowaggregator

import (
	"sync"
	"time"

	"github.com/DataDog/datadog-agent/pkg/util/log"

	"github.com/DataDog/datadog-agent/pkg/netflow/common"
)

const flowFlushInterval = 60 // TODO: make it configurable
const flowContextTTL = flowFlushInterval * 2

type flowWrapper struct {
	flow                *common.Flow
	nextFlush           time.Time
	lastSuccessfulFlush time.Time
}

// flowAccumulator is used to accumulate aggregated flows
type flowAccumulator struct {
	flows map[string]flowWrapper
	mu    sync.Mutex
}

func newFlowWrapper(flow *common.Flow) flowWrapper {
	now := time.Now()
	return flowWrapper{
		flow:                flow,
		nextFlush:           now,
		lastSuccessfulFlush: now,
	}
}

func newFlowAccumulator() *flowAccumulator {
	return &flowAccumulator{
		flows: make(map[string]flowWrapper),
	}
}

func (f *flowAccumulator) flush() []*common.Flow {
	f.mu.Lock()
	defer f.mu.Unlock()

	var flows []*common.Flow // TODO: init with optimal size
	for key, flow := range f.flows {
		now := time.Now()
		if flow.nextFlush.After(now) {
			continue
		}
		if flow.flow != nil {
			flows = append(flows, flow.flow)
			flow.lastSuccessfulFlush = now
			flow.flow = nil
		} else if time.Since(flow.lastSuccessfulFlush).Seconds() > flowContextTTL {
			delete(f.flows, key)
		}
		flow.nextFlush = flow.nextFlush.Add(flowFlushInterval * time.Second)
		f.flows[key] = flow
	}
	return flows
}

func (f *flowAccumulator) add(flowToAdd *common.Flow) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// TODO: handle port direction (see network-http-logger)
	// TODO: ignore ephemeral ports

	aggFlow, ok := f.flows[flowToAdd.AggregationHash()]
	log.Tracef("New Flow (digest=%s): %+v", flowToAdd.AggregationHash(), flowToAdd)
	aggHash := flowToAdd.AggregationHash()
	if !ok {
		f.flows[aggHash] = newFlowWrapper(flowToAdd)
	} else {
		if aggFlow.flow == nil {
			aggFlow.flow = flowToAdd
		} else {
			aggFlow.flow.Bytes += flowToAdd.Bytes
			aggFlow.flow.Packets += flowToAdd.Packets
			aggFlow.flow.ReceivedTimestamp = minUint64(aggFlow.flow.ReceivedTimestamp, flowToAdd.ReceivedTimestamp)
			aggFlow.flow.StartTimestamp = minUint64(aggFlow.flow.StartTimestamp, flowToAdd.StartTimestamp)
			aggFlow.flow.EndTimestamp = maxUint64(aggFlow.flow.EndTimestamp, flowToAdd.EndTimestamp)

			// TODO: Cumulate TCPFlags (Cumulative of all the TCP flags seen for this flow)

			log.Tracef("Existing Aggregated Flow (digest=%s): %+v", flowToAdd.AggregationHash(), aggFlow)
			log.Tracef("New Aggregated Flow (digest=%s): %+v", flowToAdd.AggregationHash(), aggFlow)
		}
		f.flows[aggHash] = aggFlow
	}
}

func minUint64(a uint64, b uint64) uint64 {
	// TODO: TESTME
	if a < b {
		return a
	}
	return b
}

func maxUint64(a uint64, b uint64) uint64 {
	// TODO: TESTME
	if a > b {
		return a
	}
	return b
}
