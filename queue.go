package main

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"sync"
)

type ProcessQueue struct {
	MaxConcurrent int
	running       map[string]bool
	queue         []QueueItem
	mu            sync.Mutex
	cpuCores      int
}

type QueueItem struct {
	Task    func() (interface{}, error)
	Result  chan TaskResult
	ID      string
}

type TaskResult struct {
	Data  interface{}
	Error error
}

type QueueStatus struct {
	MaxConcurrent int `json:"maxConcurrent"`
	Running       int `json:"running"`
	Queued        int `json:"queued"`
	CPUCores      int `json:"cpuCores"`
}

func NewProcessQueue(maxConcurrent int) *ProcessQueue {
	if maxConcurrent < 1 {
		maxConcurrent = 1
	}
	if maxConcurrent > 32 {
		maxConcurrent = 32
	}

	pq := &ProcessQueue{
		MaxConcurrent: maxConcurrent,
		running:       make(map[string]bool),
		queue:         []QueueItem{},
		cpuCores:      cpuCores,
	}

	log.Printf("[QUEUE] Initialized with max %d concurrent processes (CPU cores: %d)", maxConcurrent, cpuCores)
	return pq
}

func (pq *ProcessQueue) SetMaxConcurrent(max int) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if max < 1 {
		max = 1
	}
	if max > 32 {
		max = 32
	}

	pq.MaxConcurrent = max
	log.Printf("[QUEUE] Max concurrent processes updated to %d", pq.MaxConcurrent)
	
	// Process queue after updating limit
	go pq.processQueue()
}

func (pq *ProcessQueue) Add(task func() (interface{}, error)) (interface{}, error) {
	resultChan := make(chan TaskResult, 1)
	
	id := generateRandomID()
	
	queueItem := QueueItem{
		Task:   task,
		Result: resultChan,
		ID:     id,
	}

	pq.mu.Lock()
	pq.queue = append(pq.queue, queueItem)
	queueSize := len(pq.queue)
	runningSize := len(pq.running)
	pq.mu.Unlock()

	log.Printf("[QUEUE] Added task %s to queue. Queue size: %d, Running: %d", id, queueSize, runningSize)

	// Try to process queue
	go pq.processQueue()

	// Wait for result
	result := <-resultChan
	return result.Data, result.Error
}

func (pq *ProcessQueue) processQueue() {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	for len(pq.queue) > 0 && len(pq.running) < pq.MaxConcurrent {
		queueItem := pq.queue[0]
		pq.queue = pq.queue[1:]
		
		pq.running[queueItem.ID] = true

		log.Printf("[QUEUE] Starting task %s. Running: %d/%d", queueItem.ID, len(pq.running), pq.MaxConcurrent)

		go func(item QueueItem) {
			// Execute task
			data, err := item.Task()

			// Send result
			item.Result <- TaskResult{Data: data, Error: err}
			close(item.Result)

			// Remove from running
			pq.mu.Lock()
			delete(pq.running, item.ID)
			runningSize := len(pq.running)
			pq.mu.Unlock()

			if err != nil {
				log.Printf("[QUEUE] Failed task %s. Running: %d/%d", item.ID, runningSize, pq.MaxConcurrent)
			} else {
				log.Printf("[QUEUE] Completed task %s. Running: %d/%d", item.ID, runningSize, pq.MaxConcurrent)
			}

			// Process next item in queue
			go pq.processQueue()
		}(queueItem)
	}
}

func (pq *ProcessQueue) GetStatus() QueueStatus {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	return QueueStatus{
		MaxConcurrent: pq.MaxConcurrent,
		Running:       len(pq.running),
		Queued:        len(pq.queue),
		CPUCores:      pq.cpuCores,
	}
}

func generateRandomID() string {
	bytes := make([]byte, 4)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

func generateRandomString(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}
