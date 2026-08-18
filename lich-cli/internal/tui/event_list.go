package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"lich-cli/internal/cache"

	"github.com/charmbracelet/lipgloss"
)

var (
	selectedEventStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FFFFFF")).
				Background(lipgloss.Color("#45475A")).
				Padding(0, 1)

	cursorArrowStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#74C0FC"))

	conflictBadgeStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#FCD34D")).
				Background(lipgloss.Color("#3E2D12")).
				Padding(0, 1)

	ganttHourLabelStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#7982A9")).
				Width(6)

	ganttTimeLineStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#313244"))

	ganttBoxNormal = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#45475A")).
			Padding(0, 1)

	ganttBoxSelected = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#74C0FC")).
				Background(lipgloss.Color("#1E2030")).
				Padding(0, 1)

	ganttBoxConflict = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#FCD34D")).
				Padding(0, 1)
)

func formatEventTime(startStr, endStr string, loc *time.Location) string {
	tStart, err1 := time.Parse(time.RFC3339, startStr)
	tEnd, err2 := time.Parse(time.RFC3339, endStr)
	if err1 != nil || err2 != nil {
		return fmt.Sprintf("%s - %s", startStr, endStr)
	}

	startLocal := tStart.In(loc)
	endLocal := tEnd.In(loc)

	return fmt.Sprintf("%02d:%02d - %02d:%02d", startLocal.Hour(), startLocal.Minute(), endLocal.Hour(), endLocal.Minute())
}

func detectConflicts(events []cache.LocalEvent, loc *time.Location) map[int]bool {
	conflicts := make(map[int]bool)
	type timeSpan struct {
		start time.Time
		end   time.Time
	}
	spans := make([]timeSpan, len(events))
	for i, e := range events {
		t1, _ := time.Parse(time.RFC3339, e.StartAt)
		t2, _ := time.Parse(time.RFC3339, e.EndAt)
		spans[i] = timeSpan{start: t1.In(loc), end: t2.In(loc)}
	}

	for i := 0; i < len(events); i++ {
		for j := i + 1; j < len(events); j++ {
			if spans[i].start.Before(spans[j].end) && spans[j].start.Before(spans[i].end) {
				conflicts[i] = true
				conflicts[j] = true
			}
		}
	}
	return conflicts
}

func RenderAgenda(mode string, selectedDate time.Time, events []cache.LocalEvent, loc *time.Location, selectedIdx int, isFocused bool, width int) string {
	switch strings.ToLower(mode) {
	case "gantt", "timeline":
		return RenderAgendaGantt(selectedDate, events, loc, selectedIdx, isFocused, width)
	case "ascii":
		return RenderAgendaASCII(selectedDate, events, loc, selectedIdx, isFocused)
	default:
		return RenderAgendaList(selectedDate, events, loc, selectedIdx, isFocused)
	}
}

// 1. Mode LIST (Danh sách thẻ hiện đại + Cảnh báo trùng)
func RenderAgendaList(selectedDate time.Time, events []cache.LocalEvent, loc *time.Location, selectedIdx int, isFocused bool) string {
	var sb strings.Builder

	header := fmt.Sprintf("Agenda: %s (%d sự kiện)", selectedDate.Format("Mon, 02/01/2006"), len(events))
	if isFocused {
		header += " [TIÊU ĐIỂM]"
	}
	sb.WriteString(agendaHeaderStyle.Render(header))
	sb.WriteString("\n\n")

	if len(events) == 0 {
		sb.WriteString(eventLocStyle.Render("Không có sự kiện nào cho ngày này.\n\nNhấn 'a' để tạo sự kiện mới."))
		return sb.String()
	}

	conflicts := detectConflicts(events, loc)

	for i, ev := range events {
		isSelected := isFocused && i == selectedIdx
		cursor := "  "
		if isSelected {
			cursor = cursorArrowStyle.Render("▶ ")
		}

		timeStr := formatEventTime(ev.StartAt, ev.EndAt, loc)
		syncBadge := ""
		if ev.SyncState != cache.SyncStateSynced {
			syncBadge = " [↻]"
		}

		conflictTag := ""
		if conflicts[i] {
			conflictTag = " " + conflictBadgeStyle.Render("⚠ Trùng giờ")
		}

		titleRender := eventTitleStyle.Render(ev.Title)
		if isSelected {
			titleRender = selectedEventStyle.Render(ev.Title)
		}

		sb.WriteString(fmt.Sprintf("%s%s  %s%s%s\n", cursor, eventTimeStyle.Render(timeStr), titleRender, syncBadge, conflictTag))
		if ev.Location != "" {
			sb.WriteString(fmt.Sprintf("         %s\n", eventLocStyle.Render(ev.Location)))
		}
		if ev.Description != "" {
			sb.WriteString(fmt.Sprintf("         %s\n", eventLocStyle.Render(ev.Description)))
		}
		if i < len(events)-1 {
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// 2. Mode GANTT / TIMELINE (Chia cột song song khi trùng lịch)
type GanttItem struct {
	Index    int
	Event    cache.LocalEvent
	StartMin int
	EndMin   int
	Track    int
}

func RenderAgendaGantt(selectedDate time.Time, events []cache.LocalEvent, loc *time.Location, selectedIdx int, isFocused bool, totalWidth int) string {
	var sb strings.Builder

	header := fmt.Sprintf("Timeline Gantt: %s (%d sự kiện)", selectedDate.Format("Mon, 02/01/2006"), len(events))
	if isFocused {
		header += " [TIÊU ĐIỂM]"
	}
	sb.WriteString(agendaHeaderStyle.Render(header))
	sb.WriteString("\n\n")

	if len(events) == 0 {
		sb.WriteString(eventLocStyle.Render("Không có sự kiện nào cho ngày này.\n\nNhấn 'a' để tạo sự kiện mới."))
		return sb.String()
	}

	conflicts := detectConflicts(events, loc)

	// Phân bổ Track (Interval Partitioning)
	var items []GanttItem
	for i, ev := range events {
		t1, _ := time.Parse(time.RFC3339, ev.StartAt)
		t2, _ := time.Parse(time.RFC3339, ev.EndAt)
		loc1 := t1.In(loc)
		loc2 := t2.In(loc)

		startMin := loc1.Hour()*60 + loc1.Minute()
		endMin := loc2.Hour()*60 + loc2.Minute()
		if endMin <= startMin {
			endMin = 24 * 60 // Chạy đến hết ngày nếu qua đêm
		}

		items = append(items, GanttItem{
			Index:    i,
			Event:    ev,
			StartMin: startMin,
			EndMin:   endMin,
			Track:    0,
		})
	}

	// Sắp xếp theo giờ bắt đầu
	sort.Slice(items, func(i, j int) bool {
		if items[i].StartMin == items[j].StartMin {
			return items[i].EndMin < items[j].EndMin
		}
		return items[i].StartMin < items[j].StartMin
	})

	// Gán Track
	var trackEndTimes []int
	numTracks := 1
	for idx := range items {
		assigned := false
		for tr := 0; tr < len(trackEndTimes); tr++ {
			if trackEndTimes[tr] <= items[idx].StartMin {
				items[idx].Track = tr
				trackEndTimes[tr] = items[idx].EndMin
				assigned = true
				break
			}
		}
		if !assigned {
			items[idx].Track = len(trackEndTimes)
			trackEndTimes = append(trackEndTimes, items[idx].EndMin)
		}
		if items[idx].Track+1 > numTracks {
			numTracks = items[idx].Track + 1
		}
	}

	// Tính toán chiều rộng cột
	if totalWidth <= 0 {
		totalWidth = 54
	}
	colWidth := (totalWidth - 10) / numTracks
	if colWidth < 20 {
		colWidth = 20
	}

	// Render từng nhóm sự kiện theo track song song
	// Nhóm các items giao nhau
	type EventRow struct {
		TrackItems map[int]GanttItem
		StartMin   int
		EndMin     int
	}

	for _, item := range items {
		isSelected := isFocused && item.Index == selectedIdx
		timeStr := formatEventTime(item.Event.StartAt, item.Event.EndAt, loc)

		boxStyle := ganttBoxNormal.Width(colWidth)
		if isSelected {
			boxStyle = ganttBoxSelected.Width(colWidth)
		} else if conflicts[item.Index] {
			boxStyle = ganttBoxConflict.Width(colWidth)
		}

		tag := ""
		if isSelected {
			tag = "▶ "
		}
		if conflicts[item.Index] {
			tag += "⚠ "
		}

		cardText := fmt.Sprintf("%s%s\n%s", tag, item.Event.Title, timeStr)
		if item.Event.Location != "" {
			cardText += "\n" + item.Event.Location
		}

		renderedBox := boxStyle.Render(cardText)

		// Indent theo Track
		indent := strings.Repeat(" ", item.Track*(colWidth+2))
		timeAxis := ganttHourLabelStyle.Render(fmt.Sprintf("%02d:%02d", item.StartMin/60, item.StartMin%60))

		sb.WriteString(fmt.Sprintf("%s ── %s%s\n", timeAxis, indent, renderedBox))
	}

	return sb.String()
}

// 3. Mode ASCII (7-bit safe cho môi trường tối giản)
func RenderAgendaASCII(selectedDate time.Time, events []cache.LocalEvent, loc *time.Location, selectedIdx int, isFocused bool) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("AGENDA: %s (%d events) [ASCII MODE]\n", selectedDate.Format("2006-01-02"), len(events)))
	sb.WriteString(strings.Repeat("-", 50) + "\n")

	if len(events) == 0 {
		sb.WriteString("No events scheduled for this day.\n")
		return sb.String()
	}

	conflicts := detectConflicts(events, loc)

	for i, ev := range events {
		cursor := "  "
		if isFocused && i == selectedIdx {
			cursor = "> "
		}

		timeStr := formatEventTime(ev.StartAt, ev.EndAt, loc)
		conflictStr := ""
		if conflicts[i] {
			conflictStr = " [CONFLICT]"
		}

		sb.WriteString(fmt.Sprintf("%s%s | %s%s\n", cursor, timeStr, ev.Title, conflictStr))
		if ev.Location != "" {
			sb.WriteString(fmt.Sprintf("         Loc: %s\n", ev.Location))
		}
		if ev.Description != "" {
			sb.WriteString(fmt.Sprintf("         Desc: %s\n", ev.Description))
		}
	}

	return sb.String()
}
