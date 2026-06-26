package proxy

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/packetmind/packetmind/internal/storage"
)

func (p *Proxy) ensureActiveSession() {
	session, err := storage.Default.GetActiveSession()
	if err != nil || session == nil {
		session = &storage.Session{
			ID:       "default",
			Name:     "Default Session",
			IsActive: true,
		}
		if err := storage.Default.CreateSession(session); err != nil {
			fmt.Printf("[Proxy] Failed to create default session: %v\n", err)
			return
		}
	}
}

func (p *Proxy) saveRequestStart(record *storage.Request) {
	if !p.recordingEnabled() {
		return
	}
	p.ensureActiveSession()
	_, existingErr := storage.Default.GetRequest("", record.ID)
	isNewRecord := existingErr != nil

	if err := storage.Default.SaveRequest(record); err != nil {
		fmt.Printf("[Proxy] Failed to save request: %v\n", err)
		return
	}

	shouldEmitStart := isNewRecord && p.onRequest != nil && !(record.IsWebSocket && p.onComplete == nil)
	if shouldEmitStart {
		if saved, err := storage.Default.GetRequest("", record.ID); err == nil {
			p.onRequest(saved)
			return
		}
		p.onRequest(record)
	}
}

func (p *Proxy) saveRequestComplete(record *storage.Request) {
	if !p.recordingEnabled() {
		return
	}
	p.ensureActiveSession()

	if err := storage.Default.SaveRequest(record); err != nil {
		fmt.Printf("[Proxy] Failed to save completed request: %v\n", err)
		return
	}

	if p.onComplete != nil {
		if saved, err := storage.Default.GetRequest("", record.ID); err == nil {
			p.onComplete(saved)
			return
		}
		p.onComplete(record)
		return
	}

	if record.IsWebSocket && p.onRequest != nil {
		if saved, err := storage.Default.GetRequest("", record.ID); err == nil {
			p.onRequest(saved)
			return
		}
		p.onRequest(record)
	}
}

// recordMinimalError creates and saves a minimal request record for failed connections
func (p *Proxy) recordMinimalError(method, scheme, host string, port int, url string,
	statusCode int, errMsg string, remoteAddr, serverAddr string, startTime time.Time) {
	record := &storage.Request{
		ID:                uuid.New().String(),
		CreatedAt:         startTime,
		Method:            method,
		Scheme:            scheme,
		Host:              host,
		Port:              port,
		URL:               url,
		StatusCode:        statusCode,
		Error:             errMsg,
		RemoteAddr:        remoteAddr,
		ClientAddr:        remoteAddr,
		ServerAddr:        serverAddr,
		Headers:           make(storage.Headers),
		RespHeaders:       make(storage.Headers),
		RequestStartTime:  startTime,
		RequestEndTime:    time.Now(),
		ResponseStartTime: time.Now(),
		ResponseEndTime:   time.Now(),
		Duration:          time.Since(startTime).Milliseconds(),
	}
	p.saveRequestStart(record)
}
