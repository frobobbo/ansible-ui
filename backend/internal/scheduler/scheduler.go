package scheduler

import (
	"log"
	"sync"
	"time"

	"github.com/brettjrea/ansible-frontend/internal/models"
	"github.com/robfig/cron/v3"
)

// TriggerFunc is called on each cron tick with the form and its default variables.
type TriggerFunc func(form *models.Form, variables map[string]interface{})

// Scheduler wraps robfig/cron and maintains a registry of formID → cron entry
// so schedules can be updated or removed when forms change.
type Scheduler struct {
	c       *cron.Cron
	mu      sync.Mutex
	entries map[string]cron.EntryID
	trigger TriggerFunc
}

// New creates a Scheduler and starts the cron loop immediately.
func New(trigger TriggerFunc) *Scheduler {
	s := &Scheduler{
		c:       cron.New(),
		entries: make(map[string]cron.EntryID),
		trigger: trigger,
	}
	s.c.Start()
	return s
}

// Upsert registers or replaces the schedule for a form.
// If the form's schedule is disabled or the cron expression is empty, any
// existing entry is removed and nothing new is registered.
func (s *Scheduler) Upsert(form *models.Form) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove any existing entry for this form.
	if id, ok := s.entries[form.ID]; ok {
		s.c.Remove(id)
		delete(s.entries, form.ID)
	}

	if !form.ScheduleEnabled || form.ScheduleCron == "" {
		return
	}

	// Snapshot the fields slice so the closure captures stable data.
	fields := make([]models.FormField, len(form.Fields))
	copy(fields, form.Fields)
	formCopy := *form
	formCopy.Fields = fields

	eid, err := s.c.AddFunc(form.ScheduleCron, func() {
		vars := make(map[string]interface{})
		for _, f := range fields {
			if f.Name != "" {
				vars[f.Name] = f.DefaultValue
			}
		}
		log.Printf("[scheduler] triggering form %s (%s)", formCopy.ID, formCopy.Name)
		s.trigger(&formCopy, vars)
	})
	if err != nil {
		// Should not happen if ValidateCron was called at save time.
		log.Printf("[scheduler] failed to register cron %q for form %s: %v", form.ScheduleCron, form.ID, err)
		return
	}
	s.entries[form.ID] = eid
	log.Printf("[scheduler] registered form %q (%s) with schedule %q", form.Name, form.ID, form.ScheduleCron)
}

// Remove cancels any scheduled entry for the given form ID.
func (s *Scheduler) Remove(formID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if id, ok := s.entries[formID]; ok {
		s.c.Remove(id)
		delete(s.entries, formID)
	}
}

// NextRunAt returns the next scheduled time for a form, or nil if not scheduled.
func (s *Scheduler) NextRunAt(formID string) *time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	id, ok := s.entries[formID]
	if !ok {
		return nil
	}
	t := s.c.Entry(id).Next
	if t.IsZero() {
		return nil
	}
	return &t
}

// Stop shuts down the cron loop gracefully (blocks until running jobs finish).
func (s *Scheduler) Stop() {
	s.c.Stop()
}
