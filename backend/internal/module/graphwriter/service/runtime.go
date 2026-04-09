package service

import "time"

// RuntimeStatus 表示 graphwriter 的运行态快照。
type RuntimeStatus struct {
	Enabled         bool      `json:"enabled"`
	Started         bool      `json:"started"`
	Stopped         bool      `json:"stopped"`
	QueueLength     int       `json:"queueLength"`
	QueueCapacity   int       `json:"queueCapacity"`
	TopologyVersion string    `json:"topologyVersion"`
	TopologyUpdatedAt string  `json:"topologyUpdatedAt"`
	SubscriberActive bool     `json:"subscriberActive"`
	WorkerActive     bool     `json:"workerActive"`
	LastFlushAt      string   `json:"lastFlushAt"`
	LastFlushError   string   `json:"lastFlushError"`
	LastErrorAt      string   `json:"lastErrorAt"`
}

func (s *Service) RuntimeStatus() RuntimeStatus {
	status := RuntimeStatus{}
	if s == nil {
		return status
	}
	topology := s.CurrentTopology()
	status.Enabled = s.Enabled()
	status.Started = s.started.Load()
	status.Stopped = s.stopped.Load()
	status.QueueLength = len(s.queue)
	status.QueueCapacity = cap(s.queue)
	status.TopologyVersion = topology.Version
	if !topology.UpdatedAt.IsZero() {
		status.TopologyUpdatedAt = topology.UpdatedAt.UTC().Format(time.RFC3339)
	}
	status.SubscriberActive = s.subscriberActive.Load()
	status.WorkerActive = s.workerActive.Load()
	if unix := s.lastFlushAt.Load(); unix > 0 {
		status.LastFlushAt = time.Unix(unix, 0).UTC().Format(time.RFC3339)
	}
	if value, ok := s.lastFlushError.Load().(string); ok {
		status.LastFlushError = value
	}
	if unix := s.lastErrorAt.Load(); unix > 0 {
		status.LastErrorAt = time.Unix(unix, 0).UTC().Format(time.RFC3339)
	}
	return status
}
