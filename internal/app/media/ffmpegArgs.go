package media

import (
	"fmt"
	"path/filepath"
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

	// 2. Генерируем оптимальные параметры на основе анализа файла
	params := GenerateOptimalParams(info)

	// 3. Формируем аргументы ffmpeg
	args := []string{
		// Входной файл
		"-i", path,
		// --- Настройки видео ---
		"-map", params.VideoMap,
		"-c:v", params.VideoCodec,
		"-pix_fmt", params.PixFmt,
		"-preset", params.Preset,
		"-crf", params.CRF,
		// --- Настройки аудио ---
		"-map", params.AudioMap,
		"-c:a", params.AudioCodec,
		"-ac", params.AudioChannels,
		"-b:a", params.AudioBitrate,
		// --- Настройки HLS ---
		"-f", "hls",
		"-hls_time", "4",
		"-hls_playlist_type", "vod",
		"-hls_segment_type", "fmp4",
		"-hls_fmp4_init_filename", "init.mp4",
		"-hls_segment_filename", "segment_%04d.m4s",
		// Выходной плейлист
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
				params.CRF = "24"
				params.Preset = "medium"
			} else if stream.Width >= 1920 { // 1080p
				params.CRF = "23"
				params.Preset = "fast"
			} else { // 720p и меньше
				params.CRF = "22"
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
