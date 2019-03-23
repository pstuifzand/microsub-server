package main

import (
	"p83.nl/go/ekster/pkg/microsub"
)

// TimelineBackend specifies the interface for Timeline. It supports everything that is needed
// for Ekster to implement the channel protocol for Microsub
type TimelineBackend interface {
	Items(before, after string) (microsub.Timeline, error)
	Count() (int, error)

	AddItem(item microsub.Item) error
	MarkRead(uids []string) error

	// Not used at the moment
	// MarkUnread(uids []string) error
}

func (b *memoryBackend) getTimeline(channel string) TimelineBackend {
	// TODO: fetch timeline type from channel
	timelineType := "sorted-set"
	if channel == "notifications" {
		timelineType = "stream"
	} else {
		if setting, ok := b.Settings[channel]; ok {
			if setting.ChannelType != "" {
				timelineType = setting.ChannelType
			}
		}
	}

	if timelineType == "sorted-set" {
		timeline := &redisSortedSetTimeline{channel: channel, pool: b.pool}
		err := timeline.Init()
		if err != nil {
			return nil
		}
		return timeline
	}
	if timelineType == "stream" {
		timeline := &redisStreamTimeline{channel: channel, pool: b.pool}
		err := timeline.Init()
		if err != nil {
			return nil
		}
		return timeline
	}
	if timelineType == "null" {
		timeline := &nullTimeline{channel: channel}
		err := timeline.Init()
		if err != nil {
			return nil
		}
		return timeline
	}
	return nil
}
