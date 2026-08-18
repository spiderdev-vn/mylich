package cli

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	// Regex for 12h time: e.g. 10am, 10:30am, 11:30pm, 3:00 am, 3pm
	time12Regex = regexp.MustCompile(`^(?i)(\d{1,2})(?::(\d{2}))?\s*(am|pm)$`)
	// Regex for 24h time: e.g. 10:00, 22:33, 03:00, 23:30:00
	time24Regex = regexp.MustCompile(`^(\d{1,2}):(\d{2})(?::(\d{2}))?$`)
)

func parseFlexibleDate(input string, loc *time.Location) (time.Time, error) {
	now := time.Now().In(loc)
	clean := strings.TrimSpace(strings.ToLower(input))

	switch clean {
	case "", "today", "hom nay", "hôm nay":
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc), nil
	case "tomorrow", "ngay mai", "ngày mai":
		tomorrow := now.AddDate(0, 0, 1)
		return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, loc), nil
	case "yesterday", "hom qua", "hôm qua":
		yesterday := now.AddDate(0, 0, -1)
		return time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, loc), nil
	}

	// 1. Tách theo ký tự phân tách: '/', '-', '.'
	var separator string
	if strings.Contains(input, "/") {
		separator = "/"
	} else if strings.Contains(input, "-") {
		separator = "-"
	} else if strings.Contains(input, ".") {
		separator = "."
	} else {
		return time.Time{}, fmt.Errorf("định dạng ngày không hợp lệ '%s' (hỗ trợ dd/mm, dd-mm, dd/mm/yyyy, dd-mm-yy, yyyy-mm-dd)", input)
	}

	parts := strings.Split(input, separator)
	var day, month, year int

	if len(parts) == 3 {
		p0, err0 := strconv.Atoi(strings.TrimSpace(parts[0]))
		p1, err1 := strconv.Atoi(strings.TrimSpace(parts[1]))
		p2, err2 := strconv.Atoi(strings.TrimSpace(parts[2]))
		if err0 != nil || err1 != nil || err2 != nil {
			return time.Time{}, fmt.Errorf("định dạng ngày không hợp lệ '%s'", input)
		}

		// Trường hợp ISO YYYY-MM-DD hoặc YYYY/MM/DD (p0 >= 1000)
		if p0 >= 1000 {
			year = p0
			month = p1
			day = p2
		} else {
			// Trường hợp DD-MM-YYYY hoặc DD-MM-YY hoặc D-M-YY
			day = p0
			month = p1
			year = p2

			// Guess 2-digit year (ví dụ: 26 -> 2026)
			if year < 100 {
				year += 2000
			}
		}
	} else if len(parts) == 2 {
		// Trường hợp DD/MM, DD-MM, D/M, D-M (Tự động đoán năm hiện tại)
		p0, err0 := strconv.Atoi(strings.TrimSpace(parts[0]))
		p1, err1 := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err0 != nil || err1 != nil {
			return time.Time{}, fmt.Errorf("định dạng ngày không hợp lệ '%s'", input)
		}

		day = p0
		month = p1
		year = now.Year()
	} else {
		return time.Time{}, fmt.Errorf("định dạng ngày không hợp lệ '%s' (hỗ trợ dd/mm, dd-mm, dd-mm-yy, dd-mm-yyyy)", input)
	}

	if month < 1 || month > 12 || day < 1 || day > 31 {
		return time.Time{}, fmt.Errorf("ngày tháng không hợp lệ '%s'", input)
	}

	res := time.Date(year, time.Month(month), day, 0, 0, 0, 0, loc)
	// Kiểm tra xem ngày có bị tràn (overflow) không, ví dụ 31/02 hoặc 31/04
	if res.Day() != day || int(res.Month()) != month || res.Year() != year {
		return time.Time{}, fmt.Errorf("ngày không tồn tại trên lịch '%s'", input)
	}

	return res, nil
}

func parseFlexibleTime(input string) (hour int, minute int, err error) {
	clean := strings.TrimSpace(strings.ToLower(input))
	if clean == "" {
		return 0, 0, fmt.Errorf("chuỗi thời gian trống")
	}

	// 1. Check 12-hour format with AM/PM (e.g. 10am, 11:30pm, 3:00am)
	if match := time12Regex.FindStringSubmatch(clean); match != nil {
		h, _ := strconv.Atoi(match[1])
		m := 0
		if match[2] != "" {
			m, _ = strconv.Atoi(match[2])
		}
		period := strings.ToLower(match[3])

		if h < 1 || h > 12 || m < 0 || m > 59 {
			return 0, 0, fmt.Errorf("thời gian 12h không hợp lệ '%s'", input)
		}

		if period == "am" {
			if h == 12 {
				h = 0
			}
		} else if period == "pm" {
			if h != 12 {
				h += 12
			}
		}

		return h, m, nil
	}

	// 2. Check 24-hour format (e.g. 10:00, 22:33)
	if match := time24Regex.FindStringSubmatch(clean); match != nil {
		h, _ := strconv.Atoi(match[1])
		m, _ := strconv.Atoi(match[2])

		if h < 0 || h > 23 || m < 0 || m > 59 {
			return 0, 0, fmt.Errorf("thời gian 24h không hợp lệ '%s' (00:00 - 23:59)", input)
		}

		return h, m, nil
	}

	return 0, 0, fmt.Errorf("định dạng thời gian không hợp lệ '%s' (ví dụ: 10:00, 22:33, 11:30pm, 3am)", input)
}

func parseFlexibleTimeRange(
	dateStr string,
	atStr string,
	toStr string,
	durationStr string,
	hasDurationFlag bool,
	loc *time.Location,
) (startTime time.Time, endTime time.Time, isOvernight bool, err error) {
	return parseFlexibleTimeRangeWithEndDate(dateStr, "", atStr, toStr, durationStr, hasDurationFlag, loc)
}

func parseFlexibleTimeRangeWithEndDate(
	startDateStr string,
	endDateStr string,
	atStr string,
	toStr string,
	durationStr string,
	hasDurationFlag bool,
	loc *time.Location,
) (startTime time.Time, endTime time.Time, isOvernight bool, err error) {
	// 1. Kiểm tra mâu thuẫn giữa --to và --duration
	if toStr != "" && hasDurationFlag {
		return time.Time{}, time.Time{}, false, fmt.Errorf("không thể dùng đồng thời cả '--to' và '--duration'. Vui lòng chọn một trong hai")
	}

	// 2. Phân tích ngày bắt đầu
	targetStartDate, err := parseFlexibleDate(startDateStr, loc)
	if err != nil {
		return time.Time{}, time.Time{}, false, err
	}

	// 3. Phân tích ngày kết thúc (Mặc định: CÙNG NGÀY BẮT ĐẦU / DEFAULT SAME DAY)
	targetEndDate := targetStartDate
	if strings.TrimSpace(endDateStr) != "" {
		parsedEndDate, err := parseFlexibleDate(endDateStr, loc)
		if err != nil {
			return time.Time{}, time.Time{}, false, fmt.Errorf("ngày kết thúc không hợp lệ: %w", err)
		}
		targetEndDate = parsedEndDate
	}

	// 4. Phân tích giờ bắt đầu
	now := time.Now().In(loc)
	startHour := (now.Hour() + 1) % 24
	startMin := 0

	if atStr != "" {
		h, m, err := parseFlexibleTime(atStr)
		if err != nil {
			return time.Time{}, time.Time{}, false, fmt.Errorf("lỗi giờ bắt đầu (--at): %w", err)
		}
		startHour = h
		startMin = m
	}

	startTime = time.Date(targetStartDate.Year(), targetStartDate.Month(), targetStartDate.Day(), startHour, startMin, 0, 0, loc)

	// 5. Phân tích giờ kết thúc (--to hoặc --duration)
	if toStr != "" {
		endHour, endMin, err := parseFlexibleTime(toStr)
		if err != nil {
			return time.Time{}, time.Time{}, false, fmt.Errorf("lỗi giờ kết thúc (--to): %w", err)
		}

		endTime = time.Date(targetEndDate.Year(), targetEndDate.Month(), targetEndDate.Day(), endHour, endMin, 0, 0, loc)

		// Nếu targetEndDate cùng ngày với targetStartDate và endTime <= startTime (ví dụ bắt đầu 22:00 kết thúc 03:00) -> qua đêm
		if targetEndDate.Equal(targetStartDate) && !endTime.After(startTime) {
			endTime = endTime.AddDate(0, 0, 1)
			isOvernight = true
		} else if endTime.Before(startTime) {
			return time.Time{}, time.Time{}, false, fmt.Errorf("thời gian kết thúc (%s) phải sau thời gian bắt đầu (%s)", endTime.Format("15:04 02/01/2006"), startTime.Format("15:04 02/01/2006"))
		} else if endTime.Day() != startTime.Day() {
			isOvernight = true
		}
	} else {
		// Mặc định thời lượng là 1h nếu không truyền
		durText := durationStr
		if durText == "" {
			durText = "1h"
		}

		dur, err := time.ParseDuration(durText)
		if err != nil || dur <= 0 {
			return time.Time{}, time.Time{}, false, fmt.Errorf("thời lượng '--duration' không hợp lệ '%s' (ví dụ: 30m, 1h, 2h30m)", durText)
		}

		endTime = startTime.Add(dur)
		if endTime.Day() != startTime.Day() {
			isOvernight = true
		}
	}

	return startTime, endTime, isOvernight, nil
}
