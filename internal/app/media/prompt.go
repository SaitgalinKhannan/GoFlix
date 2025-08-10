package media

import (
	"fmt"
)

func GenerateAIPrompt(info *VideoInfo) string {
	// Форматируем информацию о потоках
	streamsInfo := ""
	for _, stream := range info.Streams {
		if stream.CodecType == "video" {
			streamsInfo += fmt.Sprintf("Видео поток %d: codec=%s, resolution=%dx%d, pix_fmt=%s\n",
				stream.Index, stream.CodecName, stream.Width, stream.Height, stream.PixFmt)
		} else if stream.CodecType == "audio" {
			streamsInfo += fmt.Sprintf("Аудио поток %d: codec=%s, channels=%d, layout=%s, sample_rate=%s, language=%s\n",
				stream.Index, stream.CodecName, stream.Channels, stream.ChannelLayout, stream.SampleRate, stream.Language)
		}
	}

	prompt := fmt.Sprintf(`Анализ медиафайла для конвертации в HLS:
		ИНФОРМАЦИЯ О ФАЙЛЕ:
		%s
		
		ЗАДАЧА: Оптимизировать параметры ffmpeg для конвертации в HLS формат.
		
		ТРЕБОВАНИЯ:
		1. Видео: выбрать лучший видеопоток, использовать libx264, yuv420p для совместимости
		2. Аудио: выбрать лучший аудиопоток (предпочтительно русский язык), конвертировать в AAC стерео
		3. Для 4K видео использовать crf 24-28, для 1080p - crf 23, для меньших разрешений - crf 22
		4. Preset выбирать в зависимости от разрешения: 4K - medium, 1080p - fast, меньше - faster
		5. Если есть HDR или 10-bit видео - обязательно конвертировать в 8-bit yuv420p
		6. Для многоканального аудио (5.1, 7.1) - микшировать в стерео с битрейтом 192k
		7. Выбирай оптимальные значения из предложенного списка.

		Верни ТОЛЬКО JSON объект, точно соответствующий следующей схеме:
       %s`, streamsInfo, "") // Схема будет передана через ResponseFormat, поэтому здесь пустая строка

	return prompt
}
