package media

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// VideoInfo структура для хранения информации о видео
type VideoInfo struct {
	Streams []Stream `json:"streams"`
	Format  Format   `json:"format"`
}

// Stream содержит информацию об одном потоке (видео, аудио, субтитры и т.д.).
type Stream struct {
	Index              int         `json:"index"`
	CodecName          string      `json:"codec_name"`
	CodecLongName      string      `json:"codec_long_name,omitempty"`
	Profile            string      `json:"profile,omitempty"`
	CodecType          string      `json:"codec_type"`
	CodecTagString     string      `json:"codec_tag_string,omitempty"`
	CodecTag           string      `json:"codec_tag,omitempty"`
	Width              int         `json:"width,omitempty"`
	Height             int         `json:"height,omitempty"`
	CodedWidth         int         `json:"coded_width,omitempty"`
	CodedHeight        int         `json:"coded_height,omitempty"`
	ClosedCaptions     int         `json:"closed_captions,omitempty"`
	HasBFrames         int         `json:"has_b_frames,omitempty"`
	SampleAspectRatio  string      `json:"sample_aspect_ratio,omitempty"`
	DisplayAspectRatio string      `json:"display_aspect_ratio,omitempty"`
	PixFmt             string      `json:"pix_fmt,omitempty"`
	Level              int         `json:"level,omitempty"`
	ColorRange         string      `json:"color_range,omitempty"`
	ColorSpace         string      `json:"color_space,omitempty"`
	ColorTransfer      string      `json:"color_transfer,omitempty"`
	ColorPrimaries     string      `json:"color_primaries,omitempty"`
	ChromaLocation     string      `json:"chroma_location,omitempty"`
	Refs               int         `json:"refs,omitempty"`
	RFrameRate         string      `json:"r_frame_rate,omitempty"`
	AvgFrameRate       string      `json:"avg_frame_rate,omitempty"`
	NbFrames           string      `json:"nb_frames,omitempty"` // Количество кадров в потоке
	TimeBase           string      `json:"time_base,omitempty"`
	StartPts           int64       `json:"start_pts,omitempty"`
	StartTime          string      `json:"start_time,omitempty"`
	DurationTs         int64       `json:"duration_ts,omitempty"`
	Duration           string      `json:"duration,omitempty"`
	BitRate            string      `json:"bit_rate,omitempty"`
	Disposition        Disposition `json:"disposition,omitempty"`
	Tags               StreamTags  `json:"tags,omitempty"`
	// Поля для аудио
	SampleFmt     string `json:"sample_fmt,omitempty"`
	SampleRate    string `json:"sample_rate,omitempty"`
	Channels      int    `json:"channels,omitempty"`
	ChannelLayout string `json:"channel_layout,omitempty"`
	BitsPerSample int    `json:"bits_per_sample,omitempty"`
}

// Disposition содержит флаги потока (например, "по умолчанию" или "форсированный").
type Disposition struct {
	Default         int `json:"default"`
	Dub             int `json:"dub"`
	Original        int `json:"original"`
	Comment         int `json:"comment"`
	Lyrics          int `json:"lyrics"`
	Karaoke         int `json:"karaoke"`
	Forced          int `json:"forced"`
	HearingImpaired int `json:"hearing_impaired"`
	VisualImpaired  int `json:"visual_impaired"`
	CleanEffects    int `json:"clean_effects"`
	AttachedPic     int `json:"attached_pic"`
	TimedThumbnails int `json:"timed_thumbnails"`
}

// StreamTags содержит метаданные (теги) для конкретного потока.
type StreamTags struct {
	Language                 string `json:"language,omitempty"`
	Title                    string `json:"title,omitempty"`
	BPS                      string `json:"BPS,omitempty"`
	Duration                 string `json:"DURATION,omitempty"`
	NumberOfFrames           string `json:"NUMBER_OF_FRAMES,omitempty"`
	NumberOfBytes            string `json:"NUMBER_OF_BYTES,omitempty"`
	StatisticsWritingApp     string `json:"_STATISTICS_WRITING_APP,omitempty"`
	StatisticsWritingDateUTC string `json:"_STATISTICS_WRITING_DATE_UTC,omitempty"`
	StatisticsTags           string `json:"_STATISTICS_TAGS,omitempty"`
	Filename                 string `json:"filename,omitempty"` // Для вложений
	MimeType                 string `json:"mimetype,omitempty"` // Для вложений
}

// Format содержит информацию о формате медиаконтейнера в целом.
type Format struct {
	Filename       string     `json:"filename"`
	NbStreams      int        `json:"nb_streams"`
	NbPrograms     int        `json:"nb_programs"`
	FormatName     string     `json:"format_name"`
	FormatLongName string     `json:"format_long_name"`
	StartTime      string     `json:"start_time"`
	Duration       string     `json:"duration"`
	Size           string     `json:"size"`
	BitRate        string     `json:"bit_rate"`
	ProbeScore     int        `json:"probe_score"`
	Tags           FormatTags `json:"tags"`
}

// FormatTags содержит метаданные (теги) для всего файла.
type FormatTags struct {
	Encoder      string `json:"encoder,omitempty"`
	CreationTime string `json:"creation_time,omitempty"`
}

// GetVideoInfo получает информацию о видео через ffprobe
func GetVideoInfo(path string) (*VideoInfo, error) {
	// Создаем контекст с таймаутом
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel() // Важно вызвать cancel, чтобы освободить ресурсы

	args := []string{
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		path,
	}

	cmd := exec.CommandContext(ctx, "ffprobe", args...)
	output, err := cmd.Output()

	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("ffprobe command timed out")
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return nil, fmt.Errorf("ffprobe failed with exit code %d: %s", exitErr.ExitCode(), string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("failed to run ffprobe: %w", err)
	}

	var info VideoInfo
	if err := json.Unmarshal(output, &info); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe output: %w", err)
	}

	return &info, nil
}

// extractFPS безопасно извлекает FPS из VideoInfo
func extractFPS(info *VideoInfo) float64 {
	for _, stream := range info.Streams {
		// Ищем первый видеопоток
		if stream.CodecType == "video" && stream.AvgFrameRate != "" {
			rateStr := stream.AvgFrameRate

			// Обрабатываем специальный случай "0/0" (неизвестный FPS)
			if rateStr == "0/0" {
				continue
			}

			// Разбиваем строку на числитель и знаменатель
			parts := strings.Split(rateStr, "/")
			if len(parts) != 2 {
				continue
			}

			// Парсим числитель и знаменатель
			num, errNum := strconv.ParseFloat(parts[0], 64)
			den, errDen := strconv.ParseFloat(parts[1], 64)

			// Проверяем корректность значений
			if errNum != nil || errDen != nil || den == 0 {
				continue
			}

			fps := num / den

			// Защита от аномальных значений (например, 1000000/1)
			if fps > 0 && fps < 120 {
				return fps
			}
		}
	}

	// Дефолтные значения для распространенных стандартов
	if info.Format.Duration != "" {
		duration, _ := strconv.ParseFloat(info.Format.Duration, 64)
		if duration > 0 {
			// Пытаемся рассчитать FPS через количество кадров (если доступно)
			for _, stream := range info.Streams {
				if stream.CodecType == "video" && stream.NbFrames != "" {
					frames, _ := strconv.Atoi(stream.NbFrames)
					if frames > 0 {
						return float64(frames) / duration
					}
				}
			}
		}
	}

	// Последняя инстанция - стандартные значения
	switch {
	case strings.Contains(info.Format.FormatName, "avi"):
		return 25.0 // PAL
	case strings.Contains(info.Format.FormatName, "mov"):
		return 29.97
	default:
		return 30.0 // Самый безопасный дефолт
	}
}
