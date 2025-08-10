package media

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/invopop/jsonschema"
	"github.com/openai/openai-go/v2"
)

// FFMpegParams структура для параметров
type FFMpegParams struct {
	VideoMap      string `json:"video_map" jsonschema_description:"Map for the video stream (e.g., 0:v:0)"`
	VideoCodec    string `json:"video_codec" jsonschema_description:"Video codec (e.g., libx264)"`
	PixFmt        string `json:"pix_fmt" jsonschema_description:"Pixel format (e.g., yuv420p)"`
	Preset        string `json:"preset" jsonschema:"enum=ultrafast,enum=superfast,enum=veryfast,enum=faster,enum=fast,enum=medium,enum=slow,enum=slower,enum=veryslow" jsonschema_description:"Encoding preset (e.g., fast, medium)"`
	CRF           string `json:"crf" jsonschema_description:"Constant Rate Factor for video quality (e.g., 23)"`
	AudioMap      string `json:"audio_map" jsonschema_description:"Map for the audio stream (e.g., 0:a:0)"`
	AudioCodec    string `json:"audio_codec" jsonschema_description:"Audio codec (e.g., aac)"`
	AudioChannels string `json:"audio_channels" jsonschema_description:"Number of audio channels (e.g., 2)"`
	AudioBitrate  string `json:"audio_bitrate" jsonschema_description:"Audio bitrate (e.g., 192k)"`
}

// GenerateSchema генерирует JSON Schema для заданной структуры.
func GenerateSchema[T any]() interface{} {
	// Structured Outputs uses a subset of JSON schema
	// These flags are necessary to comply with the subset
	reflector := jsonschema.Reflector{
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}
	var v T
	schema := reflector.Reflect(v)
	return schema
}

// FFMpegParamsResponseSchema Generate the JSON schema at initialization time
var FFMpegParamsResponseSchema = GenerateSchema[FFMpegParams]()

// GenerateFFMpegArgs генерирует аргументы для ffmpeg на основе анализа файла
func GenerateFFMpegArgs(openAIClient *openai.Client, path string) ([]string, error) {
	fileExt := filepath.Ext(path)
	filePathWithoutExt := strings.TrimSuffix(path, fileExt)

	// 1. Получаем информацию о файле
	info, err := GetVideoInfo(path)
	if err != nil {
		return nil, fmt.Errorf("[ffmpegArgs] failed to get video info: %w", err)
	}
	// 2. Генерируем промт на основе данных о видео
	prompt := GenerateAIPrompt(info)

	// 3. Генерируем параметры
	params, err := GenerateOptimalFFMpegParamsWithLLM(openAIClient, info, prompt)
	if err != nil {
		//return nil, fmt.Errorf("[ffmpegArgs] failed to generate ffmpeg params: %w", err)
		fmt.Printf("[ffmpegArgs] failed to generate ffmpeg params: %s\n", err)
		params = GenerateOptimalParams(info)
	}

	// 4. Формируем аргументы ffmpeg
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

// GenerateOptimalFFMpegParamsWithLLM использует LLM для генерации оптимальных параметров FFmpeg.
func GenerateOptimalFFMpegParamsWithLLM(client *openai.Client, info *VideoInfo, prompt string) (*FFMpegParams, error) {
	ctx := context.Background()

	// Параметры для ResponseFormat JSON Schema
	schemaParam := openai.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        "ffmpeg_parameters",
		Description: openai.String("Optimal FFmpeg parameters for HLS conversion"),
		Schema:      FFMpegParamsResponseSchema, // Использование нашей сгенерированной схемы
		Strict:      openai.Bool(true),          // Строгое соблюдение схемы
	}

	messages := []openai.ChatCompletionMessageParamUnion{
		openai.AssistantMessage("You are an expert FFmpeg parameter generator. Your task is to provide optimal FFmpeg command parameters in a structured JSON format based on user requirements and video file analysis. Ensure all output strictly adheres to the provided JSON schema."),
		openai.UserMessage(prompt),
	}

	log.Println("Calling LLM with prompt...")
	chatResp, err := client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:    "openai/gpt-4o", //openai.ChatModelGPT4oMini
		Messages: messages,
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{
				JSONSchema: schemaParam,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get chat completion from LLM: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices returned from LLM")
	}

	content := chatResp.Choices[0].Message.Content
	log.Printf("LLM Raw JSON Response: %s", content)

	var ffmpegParams FFMpegParams
	if err := json.Unmarshal([]byte(content), &ffmpegParams); err != nil {
		return nil, fmt.Errorf("failed to unmarshal LLM response into FFMpegParams: %w", err)
	}

	return &ffmpegParams, nil
}

// GenerateOptimalParams генерирует оптимальные параметры на основе анализа
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
