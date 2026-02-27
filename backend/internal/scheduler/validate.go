package scheduler

import "github.com/robfig/cron/v3"

// ValidateCron returns nil for an empty string (meaning "no schedule") or any
// valid 5-field cron expression or predefined schedule (@hourly, @daily, etc.).
func ValidateCron(expr string) error {
	if expr == "" {
		return nil
	}
	p := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	_, err := p.Parse(expr)
	return err
}
