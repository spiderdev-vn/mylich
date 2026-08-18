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

	timelineHourLabelStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("#7982A9")).
				Width(6)

	ganttBarNormalStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#A6E3A1")).
				Bold(true)

	ganttBarSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#74C0FC")).
				Bold(true)

	ganttBarConflictStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FCD34D")).
				Bold(true)

	ganttTickStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6C7086"))
)

func formatEventTime(startStr, endStr string, loc *time.Location) string {
	tStart, err1 := time.Parse(time.RFC3339, startStr)
	tEnd, err2 := time.Parse(time.RFC3339, endStr)
	if err1 != nil || err2 != nil {
		return fmt.Sprintf("%s - %s", startStr, endStr)
	}

	startLocal := tStart.In(loc)
	endLocal := tEnd.In(loc)

	startDay := startLocal.Truncate(24 * time.Hour)
	endDay := endLocal.Truncate(24 * time.Hour)

	if endDay.Equal(startDay) {
		// Cùng ngày: "10:00 - 11:30"
		return fmt.Sprintf("%02d:%02d - %02d:%02d",
			startLocal.Hour(), startLocal.Minute(),
			endLocal.Hour(), endLocal.Minute())
	}
	// Qua ngày: "22:00 02/08 - 03:00 03/08"
	return fmt.Sprintf("%02d:%02d %02d/%02d - %02d:%02d %02d/%02d",
		startLocal.Hour(), startLocal.Minute(), startLocal.Day(), int(startLocal.Month()),
		endLocal.Hour(), endLocal.Minute(), endLocal.Day(), int(endLocal.Month()))
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
	case "timeline", "ruler", "slots":
		return RenderAgendaTimeline(selectedDate, events, loc, selectedIdx, isFocused, width)
	case "gantt", "bars":
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

// 2. Mode TIMELINE (Ý tưởng 2: Trục giờ liên tục kéo dài theo tiếng, chia cột song song)
type ParsedEventItem struct {
	Index    int
	Event    cache.LocalEvent
	StartMin int
	EndMin   int
	Track    int
}

func RenderAgendaTimeline(selectedDate time.Time, events []cache.LocalEvent, loc *time.Location, selectedIdx int, isFocused bool, totalWidth int) string {
	var sb strings.Builder

	header := fmt.Sprintf("Timeline: %s (%d sự kiện)", selectedDate.Format("Mon, 02/01/2006"), len(events))
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

	var items []ParsedEventItem
	minHour := 24
	maxHour := 0

	for i, ev := range events {
		t1, _ := time.Parse(time.RFC3339, ev.StartAt)
		t2, _ := time.Parse(time.RFC3339, ev.EndAt)
		loc1 := t1.In(loc)
		loc2 := t2.In(loc)

		startMin := loc1.Hour()*60 + loc1.Minute()
		endMin := loc2.Hour()*60 + loc2.Minute()
		if endMin <= startMin {
			endMin = 24 * 60
		}

		if loc1.Hour() < minHour {
			minHour = loc1.Hour()
		}
		endHour := (endMin + 59) / 60
		if endHour > maxHour {
			maxHour = endHour
		}

		items = append(items, ParsedEventItem{
			Index:    i,
			Event:    ev,
			StartMin: startMin,
			EndMin:   endMin,
			Track:    0,
		})
	}

	if maxHour > 24 {
		maxHour = 24
	}
	if minHour > maxHour {
		minHour = 8
		maxHour = 18
	}

	// Sắp xếp
	sort.Slice(items, func(i, j int) bool {
		if items[i].StartMin == items[j].StartMin {
			return items[i].EndMin < items[j].EndMin
		}
		return items[i].StartMin < items[j].StartMin
	})

	// Phân bổ Track
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

	if totalWidth <= 0 {
		totalWidth = 56
	}
	colWidth := (totalWidth - 10) / numTracks
	if colWidth < 22 {
		colWidth = 22
	}

	// Render từng block sự kiện được nhóm theo hàng
	// Để hiển thị song song tuyệt đẹp, render các card của từng track
	var trackColumns = make([][]string, numTracks)
	for tr := 0; tr < numTracks; tr++ {
		trackColumns[tr] = make([]string, 0)
	}

	// Tạo các box card cho từng item
	for _, it := range items {
		isSelected := isFocused && it.Index == selectedIdx
		isConflict := conflicts[it.Index]

		cardStyle := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#45475A")).
			Width(colWidth).
			Padding(0, 1)

		if isSelected {
			cardStyle = cardStyle.BorderForeground(lipgloss.Color("#74C0FC")).Background(lipgloss.Color("#1E2030"))
		} else if isConflict {
			cardStyle = cardStyle.BorderForeground(lipgloss.Color("#FCD34D"))
		}

		tag := ""
		if isSelected {
			tag = "▶ "
		}
		if isConflict {
			tag += "⚠ "
		}

		timeStr := formatEventTime(it.Event.StartAt, it.Event.EndAt, loc)
		cardContent := fmt.Sprintf("%s%s\n%s", tag, it.Event.Title, timeStr)
		if it.Event.Location != "" {
			cardContent += "\n" + it.Event.Location
		}

		renderedBox := cardStyle.Render(cardContent)
		trackColumns[it.Track] = append(trackColumns[it.Track], renderedBox)
	}

	// Nếu chỉ có 1 track, hiển thị theo dòng
	if numTracks == 1 {
		for _, it := range items {
			timeAxis := timelineHourLabelStyle.Render(fmt.Sprintf("%02d:%02d", it.StartMin/60, it.StartMin%60))
			renderedBox := trackColumns[0][0]
			trackColumns[0] = trackColumns[0][1:]
			sb.WriteString(fmt.Sprintf("%s ── %s\n\n", timeAxis, renderedBox))
		}
	} else {
		// Ghép song song các track cạnh nhau
		var renderedCols []string
		for tr := 0; tr < numTracks; tr++ {
			headerTrack := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7982A9")).Render(fmt.Sprintf("Cột %d", tr+1))
			colContent := headerTrack + "\n" + strings.Join(trackColumns[tr], "\n\n")
			renderedCols = append(renderedCols, colContent)
		}
		joinedTracks := lipgloss.JoinHorizontal(lipgloss.Top, renderedCols...)
		timeAxis := timelineHourLabelStyle.Render(fmt.Sprintf("%02d:%02d", items[0].StartMin/60, items[0].StartMin%60))
		sb.WriteString(fmt.Sprintf("%s ──\n%s\n", timeAxis, joinedTracks))
	}

	return sb.String()
}

// 3. Mode GANTT BARS (Ý tưởng 3: Biểu đồ thanh ngang tỉ lệ thời gian trực quan)
func RenderAgendaGantt(selectedDate time.Time, events []cache.LocalEvent, loc *time.Location, selectedIdx int, isFocused bool, totalWidth int) string {
	var sb strings.Builder

	header := fmt.Sprintf("Gantt Bars: %s (%d sự kiện)", selectedDate.Format("Mon, 02/01/2006"), len(events))
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

	var items []ParsedEventItem
	minMin := 24 * 60
	maxMin := 0

	for i, ev := range events {
		t1, _ := time.Parse(time.RFC3339, ev.StartAt)
		t2, _ := time.Parse(time.RFC3339, ev.EndAt)
		loc1 := t1.In(loc)
		loc2 := t2.In(loc)

		startMin := loc1.Hour()*60 + loc1.Minute()
		endMin := loc2.Hour()*60 + loc2.Minute()
		if endMin <= startMin {
			endMin = 24 * 60
		}

		if startMin < minMin {
			minMin = startMin
		}
		if endMin > maxMin {
			maxMin = endMin
		}

		items = append(items, ParsedEventItem{
			Index:    i,
			Event:    ev,
			StartMin: startMin,
			EndMin:   endMin,
		})
	}

	// Làm tròn span về mốc giờ
	minHour := minMin / 60
	maxHour := (maxMin + 59) / 60
	if maxHour <= minHour {
		maxHour = minHour + 1
	}
	totalSpanMin := (maxHour - minHour) * 60
	if totalSpanMin <= 0 {
		totalSpanMin = 60
	}

	barWidth := totalWidth - 28
	if barWidth < 20 {
		barWidth = 20
	}

	// 1. Vẽ thang đo thời gian trên đầu (Time Ruler Header)
	var rulerLabels []string
	var rulerTicks []string

	numTicks := 4
	stepHour := (maxHour - minHour) / numTicks
	if stepHour < 1 {
		stepHour = 1
	}

	headerPadding := strings.Repeat(" ", 16)
	rulerLine := headerPadding
	tickLine := headerPadding

	for h := minHour; h <= maxHour; h += stepHour {
		lbl := fmt.Sprintf("%02d:00", h)
		rulerLabels = append(rulerLabels, lbl)
		rulerTicks = append(rulerTicks, "│")
	}

	// Định dạng Ruler
	rulerText := fmt.Sprintf("%s%-*s%-*s%-*s%s",
		headerPadding,
		barWidth/3, rulerLabels[0],
		barWidth/3, rulerLabels[len(rulerLabels)/2],
		barWidth/3, rulerLabels[len(rulerLabels)-1],
		"",
	)
	sb.WriteString(ganttTickStyle.Render(rulerText))
	sb.WriteString("\n")
	sb.WriteString(ganttTickStyle.Render(headerPadding + strings.Repeat("─", barWidth)))
	sb.WriteString("\n\n")

	// 2. Vẽ từng thanh Gantt bar cho mỗi sự kiện
	for _, it := range items {
		isSelected := isFocused && it.Index == selectedIdx
		isConflict := conflicts[it.Index]

		cursor := "  "
		if isSelected {
			cursor = cursorArrowStyle.Render("▶ ")
		}

		// Rút gọn title cho vừa 14 ký tự
		title := it.Event.Title
		if len([]rune(title)) > 13 {
			title = string([]rune(title)[:12]) + "…"
		}
		titleCol := fmt.Sprintf("%-14s", title)
		if isSelected {
			titleCol = selectedEventStyle.Render(titleCol)
		} else {
			titleCol = eventTitleStyle.Render(titleCol)
		}

		// Tính toán vị trí bar
		startOffset := int(float64(it.StartMin-minHour*60) / float64(totalSpanMin) * float64(barWidth))
		barLen := int(float64(it.EndMin-it.StartMin) / float64(totalSpanMin) * float64(barWidth))

		if startOffset < 0 {
			startOffset = 0
		}
		if startOffset >= barWidth {
			startOffset = barWidth - 1
		}
		if barLen < 1 {
			barLen = 1
		}
		if startOffset+barLen > barWidth {
			barLen = barWidth - startOffset
		}

		leftSpaces := strings.Repeat(" ", startOffset)
		barBlock := strings.Repeat("█", barLen)
		rightSpaces := strings.Repeat(" ", barWidth-startOffset-barLen)

		var renderedBar string
		if isSelected {
			renderedBar = ganttBarSelectedStyle.Render("[" + barBlock + "]")
		} else if isConflict {
			renderedBar = ganttBarConflictStyle.Render("[" + barBlock + "]")
		} else {
			renderedBar = ganttBarNormalStyle.Render("[" + barBlock + "]")
		}

		timeStr := formatEventTime(it.Event.StartAt, it.Event.EndAt, loc)
		conflictFlag := ""
		if isConflict {
			conflictFlag = " " + conflictBadgeStyle.Render("⚠")
		}
		locInfo := ""
		if it.Event.Location != "" {
			locInfo = fmt.Sprintf(" (%s)", it.Event.Location)
		}

		sb.WriteString(fmt.Sprintf("%s%s %s%s%s  %s%s%s\n",
			cursor,
			titleCol,
			leftSpaces,
			renderedBar,
			rightSpaces,
			eventTimeStyle.Render(timeStr),
			conflictFlag,
			eventLocStyle.Render(locInfo),
		))
	}

	_ = rulerLine
	_ = tickLine
	return sb.String()
}

// 4. Mode ASCII (7-bit safe cho môi trường tối giản)
func RenderAgendaASCII(selectedDate time.Time, events []cache.LocalEvent, loc *time.Location, selectedIdx int, isFocused bool) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "AGENDA: %s (%d events) [ASCII MODE]\n", selectedDate.Format("2006-01-02"), len(events))
	sb.WriteString(strings.Repeat("-", 50))
	sb.WriteString("\n")

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
