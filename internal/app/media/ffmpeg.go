package media

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func CopyToHls(path string) error {
	fileExt := filepath.Ext(path)
	filePathWithoutExt := strings.TrimSuffix(path, fileExt)

	// Создаем директорию для сегментов, если она не существует
	if err := os.MkdirAll(filePathWithoutExt, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", filePathWithoutExt, err)
	}

	// Формируем аргументы для ffmpeg как отдельные строки
	args := []string{
		"-i", path,
		"-c:v", "copy",
		"-c:a", "copy",
		"-map", "0:v",
		"-map", "0:a",
		"-movflags", "+faststart",
		"-f", "hls",
		"-hls_segment_type", "fmp4",
		"-hls_time", "4",
		"-hls_playlist_type", "vod",
		"-hls_flags", "independent_segments",
		"-hls_segment_filename", filepath.Join(filePathWithoutExt, "segment_%03d.m4s"),
		filePathWithoutExt + ".m3u8",
	}

	cmd := exec.Command("ffmpeg", args...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("[ffmpeg] Failed to convert %s to hls: %s\n", filePathWithoutExt, err)
	}
	return nil
}

func ConvertToHls(path string) error {
	fileExt := filepath.Ext(path)
	filePathWithoutExt := strings.TrimSuffix(path, fileExt)

	// Создаем директорию для сегментов, если она не существует
	if err := os.MkdirAll(filePathWithoutExt, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", filePathWithoutExt, err)
	}

	fmt.Printf("filePathWithoutExt: %s\n", filePathWithoutExt)
	fmt.Printf("init.mp4: %s\n", filepath.Join(filePathWithoutExt, "init.mp4"))

	args := []string{
		"-i", path,
		"-c:v", "libx264",
		"-preset", "superfast",
		"-movflags", "+faststart",
		"-crf", "30",
		"-c:a", "aac",
		"-b:a", "128k",
		"-map", "0:v",
		"-map", "0:a",
		"-f", "hls",
		"-hls_time", "4",
		"-hls_playlist_type", "vod",
		"-hls_flags", "independent_segments",
		"-hls_segment_filename", "segment_%03d.ts",
		filepath.Join(filePathWithoutExt, "playlist.m3u8"),
	}

	/*args := []string{
		"-i", path,
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-movflags", "+faststart",
		"-crf", "30",
		"-c:a", "aac",
		"-b:a", "128k",
		"-map", "0:v",
		"-map", "0:a",
		"-f", "hls",
		"-hls_segment_type", "fmp4",
		"-hls_time", "4",
		"-hls_playlist_type", "vod",
		"-hls_flags", "independent_segments",
		"-hls_fmp4_init_filename", "init.mp4",
		"-hls_segment_filename", "segment_%03d.m4s",
		filepath.Join(filePathWithoutExt, "playlist.m3u8"),
	}*/

	cmd := exec.Command("ffmpeg", args...)
	cmd.Dir = filePathWithoutExt // Устанавливаем рабочую директорию для команды
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("[ffmpeg] Failed to convert %s to hls: %s\n", filePathWithoutExt, err)
	}

	return nil
}

func ConvertToHlsWithAdaptiveBitrateSingle(path string) error {
	fileExt := filepath.Ext(path)
	filePathWithoutExt := strings.TrimSuffix(path, fileExt)

	// Создаем директорию для сегментов, если она не существует
	if err := os.MkdirAll(filePathWithoutExt, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", filePathWithoutExt, err)
	}

	// Разные качества
	qualities := []struct {
		bitrate    string
		resolution string
		filename   string
	}{
		{"800k", "640x360", "360p"},
		{"1400k", "842x480", "480p"},
		{"2800k", "1280x720", "720p"},
		{"5000k", "1920x1080", "1080p"},
	}

	args := []string{
		"-i", path,
		"-c:a", "aac",
		"-ar", "48000",
		"-b:a", "128k",
		"-c:v", "libx264",
		"-preset", "fast",
		"-f", "hls",
		"-hls_segment_type", "fmp4",
		"-hls_time", "4",
		"-hls_playlist_type", "vod",
		"-hls_flags", "independent_segments",
	}

	// Добавляем параметры для каждого качества
	for i, q := range qualities {
		args = append(args,
			"-map", "0:v",
			"-map", "0:a",
			fmt.Sprintf("-b:v:%d", i), q.bitrate,
			fmt.Sprintf("-s:v:%d", i), q.resolution,
			fmt.Sprintf("-hls_segment_filename:v:%d", i), filepath.Join(filePathWithoutExt, q.filename+"_%03d.m4s"),
		)
	}

	// Добавляем выходные файлы
	for _, q := range qualities {
		args = append(args, filepath.Join(filePathWithoutExt, q.filename+".m3u8"))
	}

	cmd := exec.Command("ffmpeg", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("[ffmpeg] Failed to convert %s to adaptive HLS: %w", filePathWithoutExt, err)
	}

	// Создаем мастер-плейлист
	qualities_for_master := []struct {
		bitrate    string
		resolution string
		filename   string
	}{
		{"800k", "640x360", "360p"},
		{"1400k", "842x480", "480p"},
		{"2800k", "1280x720", "720p"},
		{"5000k", "1920x1080", "1080p"},
	}

	if err := createMasterPlaylist(filePathWithoutExt, qualities_for_master); err != nil {
		return fmt.Errorf("failed to create master playlist: %w", err)
	}

	return nil
}

func createMasterPlaylist(baseDir string, qualities []struct {
	bitrate    string
	resolution string
	filename   string
}) error {
	masterPlaylist := "#EXTM3U\n#EXT-X-VERSION:3\n"

	for _, q := range qualities {
		// Парсим битрейт (убираем 'k' и конвертируем в число)
		bitrateStr := strings.TrimSuffix(q.bitrate, "k")
		bitrate := bitrateStr + "000" // конвертируем килобиты в биты

		masterPlaylist += fmt.Sprintf("#EXT-X-STREAM-INF:BANDWIDTH=%s,RESOLUTION=%s\n", bitrate, q.resolution)
		masterPlaylist += fmt.Sprintf("%s.m3u8\n", q.filename)
	}

	masterFile := filepath.Join(baseDir, "master.m3u8")
	return os.WriteFile(masterFile, []byte(masterPlaylist), 0644)
}
