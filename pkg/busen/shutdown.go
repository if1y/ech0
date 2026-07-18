// SPDX-License-Identifier: AGPL-3.0-or-later
// Copyright (C) 2025-2026 lin-snow

package busen

import (
	"context"
	"fmt"
)

// ShutdownMode controls how bus shutdown handles queued async events.
type ShutdownMode int

const (
	// ShutdownDrain waits for async queues to drain.
	ShutdownDrain ShutdownMode = iota
	// ShutdownBestEffort stops accepting work and waits until context ends.
	ShutdownBestEffort
	// ShutdownAbort stops accepting work and drops queued async events.
	ShutdownAbort
)

// ShutdownResult reports structured shutdown outcomes.
type ShutdownResult struct {
	Mode ShutdownMode
	// Processed is the number of handler executions observed during shutdown.
	Processed int64
	// Dropped is the number of dropped events observed during shutdown.
	// It includes backpressure drops and abort-mode queue drops.
	Dropped int64
	// Rejected is the number of rejected events observed during shutdown.
	Rejected int64
	// TimedOutSubscribers contains subscriber IDs that did not stop before ctx ended.
	TimedOutSubscribers []uint64
	// Completed reports whether shutdown fully completed before context cancellation.
	Completed bool
}

// Shutdown stops accepting new publishes and subscriptions according to mode.
func (b *Bus) Shutdown(ctx context.Context, mode ShutdownMode) (ShutdownResult, error) {
	result := ShutdownResult{Mode: mode}

	if b == nil {
		return result, fmt.Errorf("%w: nil bus", ErrInvalidOption)
	}
	if !mode.valid() {
		return result, fmt.Errorf("%w: unknown shutdown mode", ErrInvalidOption)
	}
	if ctx == nil {
		ctx = context.Background()
	}

	// before 快照必须先于 gate.Close()：这样发布方一旦观察到 ErrClosed，快照必然已拍下，
	// 此后完成的 handler 执行都会计入增量（反过来的顺序会漏计 Close 与快照之间完成的执行，
	// 外部也没有任何时点可以确认快照已生效）。权威订阅列表 subs 仍在 Close 之后取才完备；
	// 夹缝中新增的订阅不在 before 里，applyStatsDelta 对其按零基线计入，语义无损。
	before := snapshotSubscriptionStats(b.allSubscriptions())
	b.gate.Close()
	subs := b.allSubscriptions()

	if err := b.gate.Wait(ctx); err != nil {
		result.Completed = false
		if mode == ShutdownDrain {
			return result, closeWaitError(err)
		}
		result.TimedOutSubscribers = allSubscriptionIDs(subs)
		applyStatsDelta(&result, before, snapshotSubscriptionStats(subs))
		return result, nil
	}

	for _, sub := range subs {
		sub.stopAccepting()
	}
	if mode == ShutdownAbort {
		for _, sub := range subs {
			sub.abortPending()
		}
	}
	for _, sub := range subs {
		sub.scheduleStop()
	}

	for _, sub := range subs {
		if err := sub.waitStopped(ctx); err != nil {
			result.TimedOutSubscribers = append(result.TimedOutSubscribers, sub.id)
			if mode == ShutdownDrain {
				applyStatsDelta(&result, before, snapshotSubscriptionStats(subs))
				result.Completed = false
				return result, closeWaitError(err)
			}
		}
	}

	applyStatsDelta(&result, before, snapshotSubscriptionStats(subs))
	result.Completed = len(result.TimedOutSubscribers) == 0
	return result, nil
}

func snapshotSubscriptionStats(subs []*subscription) map[uint64]subscriptionStats {
	stats := make(map[uint64]subscriptionStats, len(subs))
	for _, sub := range subs {
		stats[sub.id] = sub.snapshotStats()
	}
	return stats
}

func applyStatsDelta(result *ShutdownResult, before, after map[uint64]subscriptionStats) {
	if result == nil {
		return
	}
	for id, end := range after {
		start, ok := before[id]
		if !ok {
			start = subscriptionStats{}
		}
		result.Processed += end.processed - start.processed
		result.Dropped += (end.dropped - start.dropped) + (end.shutdownDrop - start.shutdownDrop)
		result.Rejected += end.rejected - start.rejected
	}
}

func allSubscriptionIDs(subs []*subscription) []uint64 {
	if len(subs) == 0 {
		return nil
	}
	ids := make([]uint64, 0, len(subs))
	for _, sub := range subs {
		ids = append(ids, sub.id)
	}
	return ids
}

func (m ShutdownMode) valid() bool {
	return m >= ShutdownDrain && m <= ShutdownAbort
}
