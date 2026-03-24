package services

import (
	"Backend/domain/repository"
	"Backend/internal/models"
	"errors"
	"fmt"
	"strings"
	"time"
)

type ScheduleService struct {
	repo repository.ScheduleRepository
}

func NewScheduleService(repo repository.ScheduleRepository) *ScheduleService {
	return &ScheduleService{repo: repo}
}

func (s *ScheduleService) Create(userID uint, companyName, stage, title string, scheduledAt time.Time, notes string) (*models.ScheduleEvent, error) {
	if strings.TrimSpace(companyName) == "" {
		return nil, errors.New("company_name is required")
	}
	if scheduledAt.IsZero() {
		return nil, errors.New("scheduled_at is required")
	}
	event := &models.ScheduleEvent{
		UserID:      userID,
		CompanyName: companyName,
		Stage:       models.ScheduleStage(stage),
		Title:       title,
		ScheduledAt: scheduledAt,
		Notes:       notes,
	}
	if err := s.repo.Create(event); err != nil {
		return nil, err
	}
	return event, nil
}

func (s *ScheduleService) Get(userID, eventID uint) (*models.ScheduleEvent, error) {
	event, err := s.repo.FindByID(eventID)
	if err != nil {
		return nil, err
	}
	if event.UserID != userID {
		return nil, errors.New("forbidden")
	}
	return event, nil
}

func (s *ScheduleService) Update(userID, eventID uint, companyName, stage, title string, scheduledAt time.Time, notes string) (*models.ScheduleEvent, error) {
	event, err := s.repo.FindByID(eventID)
	if err != nil {
		return nil, err
	}
	if event.UserID != userID {
		return nil, errors.New("forbidden")
	}
	if strings.TrimSpace(companyName) != "" {
		event.CompanyName = companyName
	}
	if stage != "" {
		event.Stage = models.ScheduleStage(stage)
	}
	event.Title = title
	if !scheduledAt.IsZero() {
		event.ScheduledAt = scheduledAt
	}
	event.Notes = notes
	if err := s.repo.Update(event); err != nil {
		return nil, err
	}
	return event, nil
}

func (s *ScheduleService) Delete(userID, eventID uint) error {
	event, err := s.repo.FindByID(eventID)
	if err != nil {
		return err
	}
	if event.UserID != userID {
		return errors.New("forbidden")
	}
	return s.repo.Delete(eventID)
}

func (s *ScheduleService) List(userID uint) ([]models.ScheduleEvent, error) {
	return s.repo.ListByUser(userID)
}

func (s *ScheduleService) ListByRange(userID uint, from, to time.Time) ([]models.ScheduleEvent, error) {
	return s.repo.ListByUserAndRange(userID, from, to)
}

// ExportICS は指定ユーザーの全スケジュールを iCalendar 形式で返す。
func (s *ScheduleService) ExportICS(userID uint) (string, error) {
	events, err := s.repo.ListByUser(userID)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString("BEGIN:VCALENDAR\r\n")
	b.WriteString("VERSION:2.0\r\n")
	b.WriteString("PRODID:-//soc-ai-agent//Schedule//JA\r\n")
	b.WriteString("CALSCALE:GREGORIAN\r\n")
	b.WriteString("METHOD:PUBLISH\r\n")

	for _, ev := range events {
		title := ev.Title
		if title == "" {
			title = fmt.Sprintf("%s - %s", ev.CompanyName, ev.Stage)
		}
		dtStart := ev.ScheduledAt.UTC().Format("20060102T150405Z")
		dtEnd := ev.ScheduledAt.UTC().Add(time.Hour).Format("20060102T150405Z")
		uid := fmt.Sprintf("schedule-%d@soc-ai-agent", ev.ID)

		b.WriteString("BEGIN:VEVENT\r\n")
		fmt.Fprintf(&b, "UID:%s\r\n", uid)
		fmt.Fprintf(&b, "DTSTART:%s\r\n", dtStart)
		fmt.Fprintf(&b, "DTEND:%s\r\n", dtEnd)
		fmt.Fprintf(&b, "SUMMARY:%s\r\n", escapeICS(title))
		if ev.Notes != "" {
			fmt.Fprintf(&b, "DESCRIPTION:%s\r\n", escapeICS(ev.Notes))
		}
		fmt.Fprintf(&b, "CATEGORIES:%s\r\n", escapeICS(string(ev.Stage)))
		b.WriteString("END:VEVENT\r\n")
	}

	b.WriteString("END:VCALENDAR\r\n")
	return b.String(), nil
}

func escapeICS(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, ";", "\\;")
	s = strings.ReplaceAll(s, ",", "\\,")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}
