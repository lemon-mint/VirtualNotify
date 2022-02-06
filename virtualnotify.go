package virtualnotify

import (
	"crypto/sha256"
	"encoding/base32"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gofrs/flock"
)

func hashstr(s string) string {
	v := sha256.Sum256([]byte(s))
	return base32.StdEncoding.EncodeToString(v[:])
}

type Event struct {
	Name     string
	filepath string
}

var TempDir = os.TempDir()

type VirtualNotify struct {
	nameSpace string

	lockfile string
	fl       *flock.Flock

	events   chan Event
	stopChan chan struct{}

	mu   sync.Mutex
	subs map[eventFile]struct{}
}

type eventFile struct {
	EventName string
	FilePath  string
}

func New(ns string) *VirtualNotify {
	f := "vn_" + hashstr(ns) + ".lock"
	f = filepath.Join(TempDir, f)

	fl := flock.New(f)

	return &VirtualNotify{
		subs:      make(map[eventFile]struct{}),
		nameSpace: hashstr(ns),
		lockfile:  f,
		fl:        fl,
		events:    make(chan Event),
		stopChan:  make(chan struct{}),
	}
}

func (p *VirtualNotify) Subscribe(eventName string) error {
	p.fl.Lock()
	defer p.fl.Unlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	f := "vn_" + p.nameSpace + "_" + hashstr(eventName) + ".virtualnotify"
	f = filepath.Join(TempDir, f)

	if _, ok := p.subs[eventFile{eventName, f}]; ok {
		return nil
	}

	file, err := os.Create(f)
	if err != nil {
		return err
	}
	file.Close()

	p.subs[eventFile{eventName, f}] = struct{}{}
	return nil
}

func (p *VirtualNotify) Unsubscribe(eventName string) {
	p.fl.Lock()
	defer p.fl.Unlock()

	p.mu.Lock()
	defer p.mu.Unlock()
	f := "vn_" + p.nameSpace + "_" + hashstr(eventName) + ".virtualnotify"
	f = filepath.Join(TempDir, f)

	delete(p.subs, eventFile{eventName, f})
	err := os.Remove(f)
	if err != nil {
		return
	}
}

func (p *VirtualNotify) Run(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-p.stopChan:
			return
		case <-ticker.C:
			p.mu.Lock()
			for k := range p.subs {
				// check if file exists
				if _, err := os.Stat(k.FilePath); os.IsNotExist(err) {
					p.events <- Event{k.EventName, k.FilePath}
					f, err := os.Create(k.FilePath)
					if err != nil {
						return
					}
					f.Close()
				}
			}
			p.mu.Unlock()
		}
	}
}

func (p *VirtualNotify) Cleanup() {
	err := p.fl.Lock()
	if err != nil {
		return
	}
	defer p.fl.Unlock()

	p.mu.Lock()
	defer p.mu.Unlock()

	for k := range p.subs {
		p.Unsubscribe(k.EventName)
	}
}

func (p *VirtualNotify) Publish(eventName string) error {
	err := p.fl.Lock()
	if err != nil {
		return err
	}
	defer p.fl.Unlock()

	f := "vn_" + p.nameSpace + "_" + hashstr(eventName) + ".virtualnotify"
	f = filepath.Join(TempDir, f)

	err = os.Remove(f)
	if err != nil {
		return err
	}

	return nil
}

func (p *VirtualNotify) EventsChan() <-chan Event {
	return p.events
}

func (p *VirtualNotify) Close() {
	p.stopChan <- struct{}{}
	p.Cleanup()
	close(p.events)
}

var ErrClosed = os.ErrClosed

func (p *VirtualNotify) Next() (Event, error) {
	v, ok := <-p.events
	if !ok {
		return Event{}, ErrClosed
	}
	return v, nil
}
