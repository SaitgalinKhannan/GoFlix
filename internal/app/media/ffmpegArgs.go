package media

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// FFMpegParams структура для параметров FFmpeg (локальная версия для media пакета)
type FFMpegParams struct {
	VideoMap      string `json:"video_map"`
	VideoCodec    string `json:"video_codec"`
	PixFmt        string `json:"pix_fmt"`
	Preset        string `json:"preset"`
	CRF           string `json:"crf"`
	AudioMap      string `json:"audio_map"`
	AudioCodec    string `json:"audio_codec"`
	AudioChannels string `json:"audio_channels"`
	AudioBitrate  string `json:"audio_bitrate"`
}

// GenerateFFMpegArgs генерирует аргументы для ffmpeg на основе анализа файла
func GenerateFFMpegArgs(path string) ([]string, error) {
	fileExt := filepath.Ext(path)
	filePathWithoutExt := strings.TrimSuffix(path, fileExt)

	// 1. Получаем информацию о файле
	info, err := GetVideoInfo(path)
	if err != nil {
		return nil, fmt.Errorf("[ffmpegArgs] failed to get video info path%s: %w", path, err)
	}

	// 2. Генерируем оптимальные параметры
	params := GenerateOptimalParams(info)

	// 3. РАССЧИТЫВАЕМ GOP (критически важно!)
	fps := extractFPS(info)                     // Получаем FPS из VideoInfo
	hlsTime := 4.0                              // Должно совпадать с "-hls_time 4"
	gopSize := strconv.Itoa(int(fps * hlsTime)) // GOP в кадрах = FPS * длина сегмента

	args := []string{
		// Входной файл
		"-i", path,

		// Нормализация временных меток
		"-avoid_negative_ts", "make_zero",
		"-fflags", "+genpts",

		// --- Настройки видео ---
		"-map", params.VideoMap,
		"-c:v", params.VideoCodec,
		"-pix_fmt", params.PixFmt,
		"-preset", params.Preset,
		"-crf", params.CRF,
		"-vsync", "cfr",

		// Ключевые кадры для правильной сегментации
		"-force_key_frames", "expr:gte(t,n_forced*4)",
		"-g", gopSize,
		"-keyint_min", gopSize,
		"-sc_threshold", "0",

		// --- Настройки аудио ---
		"-map", params.AudioMap,
		"-c:a", params.AudioCodec,
		"-ac", params.AudioChannels,
		"-b:a", params.AudioBitrate,
		"-async", "1", // Аудио синхронизация

		// --- Настройки HLS ---
		"-f", "hls",
		"-hls_time", strconv.FormatFloat(hlsTime, 'f', 0, 64),
		"-hls_playlist_type", "vod",
		"-hls_segment_type", "fmp4",

		// --- Ключевое ИЗМЕНЕНИЕ ---
		// Гарантирует, что каждый сегмент можно декодировать независимо.
		// ffmpeg сам позаботится о расстановке ключевых кадров на границах сегментов.
		"-hls_flags", "independent_segments",
		"-hls_fmp4_init_filename", "init.mp4",
		"-hls_segment_filename", "segment_%04d.m4s",

		filepath.Join(filePathWithoutExt, "playlist.m3u8"),
	}

	return args, nil
}

// GenerateOptimalParams генерирует оптимальные параметры для ffmpeg на основе анализа
func GenerateOptimalParams(info *VideoInfo) *FFMpegParams {
	params := &FFMpegParams{
		VideoMap:      "0:v:0",
		VideoCodec:    "libx264",
		PixFmt:        "yuv420p",
		Preset:        "fast",
		CRF:           "23",
		AudioMap:      "0:a:0",
		AudioCodec:    "aac",
		AudioChannels: "2",
		AudioBitrate:  "192k",
	}

	// Анализируем видео потоки
	for _, stream := range info.Streams {
		if stream.CodecType == "video" && stream.Index == 0 {
			// Настройка CRF и preset по разрешению
			if stream.Width >= 3840 { // 4K
				params.CRF = "18"
				params.Preset = "medium"
			} else if stream.Width >= 1920 { // 1080p
				params.CRF = "23"
				params.Preset = "fast"
			} else { // 720p и меньше
				params.CRF = "24"
				params.Preset = "faster"
			}

			// Проверка на HDR/10-bit
			if stream.PixFmt == "yuv420p10le" || stream.PixFmt == "yuv422p10le" {
				params.PixFmt = "yuv420p" // Принудительная конвертация в 8-bit
			}
		}

		// Выбираем лучший аудио поток
		if stream.CodecType == "audio" {
			if stream.Channels > 2 {
				params.AudioChannels = "2" // Микшируем в стерео
				params.AudioBitrate = "192k"
			}
		}
	}

	return params
}
