package env

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"math"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	ENCODER_WORKERS           = EnvNumber("ENCODER_WORKERS", 1)
	ENCODER_USE_HARDWARE      = EnvString("ENCODER_USE_HARDWARE", "true") == "true"
	OUTPUT_FILENAME_VIDEO     = EnvString("ENCODER_OUTPUT_FILENAME_VIDEO", "video.mp4")
	OUTPUT_FILENAME_THUMBNAIL = EnvString("ENCODER_OUTPUT_FILENAME_THUMBNAIL", "preview.webp")
	VIDEO_HEIGHT_LIMIT        = EnvNumber("ENCODER_VIDEO_HEIGHT_LIMIT", 1080)
	VIDEO_FPS_LIMIT           = EnvNumber("ENCODER_VIDEO_FPS_LIMIT", 60)
	VIDEO_STREAMS_LIMIT       = EnvNumber("ENCODER_VIDEO_STREAMS_LIMIT", 1)
	VIDEO_PIXEL_FORMAT        = EnvString("ENCODER_VIDEO_PIXEL_FORMAT", "yuv420p")
	VIDEO_PRESET              = EnvString("ENCODER_VIDEO_PRESET", "fast")
	VIDEO_QUALITY             = EnvString("ENCODER_VIDEO_QUALITY", "27")
	VIDEO_CODEC               = EnvString("ENCODER_VIDEO_CODEC", "libx264")
	VIDEO_HARDWARE_CODEC      = EnvString("ENCODER_VIDEO_HARDWARE_CODEC", "h264_nvenc,h264_qsv,h264_amf")
	AUDIO_STREAMS_LIMIT       = EnvNumber("ENCODER_AUDIO_STREAMS_LIMIT", 6)
	AUDIO_BITRATE             = EnvString("ENCODER_AUDIO_BITRATE", "320K")
	AUDIO_CODEC               = EnvString("ENCODER_AUDIO_CODEC", "aac")
	AUDIO_CHANNELS            = EnvString("ENCODER_AUDIO_CHANNELS", "2")
)

var (
	encoderLostQueue = make(chan string, 100)
	encoderCond      = sync.NewCond(&sync.Mutex{})
	encoderStart     sync.Once
)

// Wake up a sleeping encoder to start working
func WakeEncoder() {
	encoderCond.L.Lock()
	encoderCond.Signal()
	encoderCond.L.Unlock()
}

// Setup for Encoding
func StartEncoders(stop context.Context, await *sync.WaitGroup) {
	encoderStart.Do(func() {

		// Attempt to encode a black frame using an encoder from the list
		// If succesful use that instead
		if ENCODER_USE_HARDWARE {
			for _, someCodec := range strings.Split(VIDEO_HARDWARE_CODEC, ",") {
				if err := exec.Command(
					"ffmpeg", "-an", "-sn",
					"-f", "lavfi",
					"-i", "color=black:s=1080x1080",
					"-vframes", "1",
					"-c:v", someCodec,
					"-f", "null",
					"-",
				).Run(); err == nil {
					VIDEO_CODEC = someCodec
					break
				}
			}
		}
		log.Println("[env/encoder] Using video codec:", VIDEO_CODEC)
		log.Println("[env/encoder] Using audio codec:", AUDIO_CODEC)

		// Fill Lost Video Queue
		// These are videos that were being still being processed by the server when it shutdown
		rows, err := DB.Query("SELECT id from videos WHERE status = 'PROCESS'")
		if err != nil {
			log.Fatalln("[env/encoder]", err)
		}
		for rows.Next() {
			var videoID string
			if err := rows.Scan(&videoID); err != nil {
				log.Fatalln("[env/encoder]", err)
			}
			select {
			case encoderLostQueue <- videoID:
			default:
				// This should NEVER happen, unless you have like over a hundred
				// encoders (on one machine too btw) but at that point ???
				log.Fatalln("[env/encoder] Lost queue is full, resizing...")
				newQueue := make(chan string, cap(encoderLostQueue)+100)
				drainLen := len(encoderLostQueue)
				for i := 0; i < drainLen; i++ {
					newQueue <- <-encoderLostQueue
				}
				encoderLostQueue = newQueue
				encoderLostQueue <- videoID
			}
		}

		// Startup Encoders
		for i := 0; i < ENCODER_WORKERS; i++ {
			go startEncoder(i)
		}

		// Shutdown Logic
		await.Add(1)
		go func() {
			defer await.Done()
			<-stop.Done()
			log.Println("[env/encoder] Closed Encoders")
		}()

	})
}

// Startup an Encoder
func startEncoder(workerId int) {
	for {

		// Step 0. Look for work
		var videoID, videoCreated, userID string
	searchLoop:
		for {
			select {

			// Fetch an Item from the Lost Video Backlog
			case videoId := <-encoderLostQueue:
				err := DB.
					QueryRow("SELECT id, created, user_id FROM videos WHERE id = $1", videoId).
					Scan(&videoID, &videoCreated, &userID)
				if err == sql.ErrNoRows {
					continue
				}
				if err != nil {
					log.Printf("[encoders][%d] %s\n", workerId, err)
					time.Sleep(time.Second)
					continue
				}
				break searchLoop

			// Search the Database for a Queued Video
			default:
				err := DB.
					QueryRow("SELECT id, created, user_id FROM videos WHERE status = 'QUEUE'").
					Scan(&videoID, &videoCreated, &userID)
				if err == sql.ErrNoRows {
					log.Printf("[encoders][%d] Sleeping...\n", workerId)
					encoderCond.L.Lock()
					encoderCond.Wait()
					encoderCond.L.Unlock()
					continue
				}
				if err != nil {
					log.Printf("[encoders][%d] %s\n", workerId, err)
					time.Sleep(time.Second)
					continue
				}
				break searchLoop
			}
		}

		// Step 1. Preparations
		var (
			inputFilepath        = path.Join(DATA_DIR, "video", videoID)
			outputDirectory      = path.Join(DATA_DIR, "public", videoID)
			errorMessage         string
			errorOutput          string
			encodeVideoHeight    int
			encodeVideoFramerate int
			encodeAudioStreams   int
			encodeVideoStreams   int
		)
		defer func() {
			if errorMessage != "" {
				log.Printf(
					"[encoders][%d] Encoding Error (ID: %s): %s\nOutput: %s\n---\n",
					workerId, videoID, errorMessage, errorOutput,
				)
				SendEvent(userID, "VIDEO_PROCESSING_ERROR", videoID, errorMessage)
				DB.Exec("UPDATE videos SET status = 'ERROR' WHERE id = $1", videoID)
				os.RemoveAll(outputDirectory)
			}
		}()
		if err := os.MkdirAll(outputDirectory, FILE_MODE); err != nil {
			errorMessage = "Cannot Create Output Directory"
			errorOutput = err.Error()
			return
		}
		if _, err := DB.Exec("UPDATE videos SET status = 'PROCESS' WHERE id = $1", videoID); err != nil {
			errorMessage = "Cannot Mark Video as Processing"
			errorOutput = err.Error()
			return
		}
		SendEvent(userID, "VIDEO_PROCESSING_BEGIN", videoID, "")

		// Step 2. Probe Video File
		var Probe struct {
			Streams []struct {
				Index            int       `json:"index"`
				CodecName        string    `json:"codec_name"`
				Profile          string    `json:"profile"`
				CodecType        string    `json:"codec_type"`
				Width            int       `json:"width"`
				Height           int       `json:"height"`
				SampleRate       string    `json:"sample_rate"`
				Channels         int       `json:"channels"`
				ChannelLayout    string    `json:"channel_layout"`
				AverageFrameRate framerate `json:"avg_frame_rate"`
				TimeBase         string    `json:"time_base"`
				Duration         string    `json:"duration"`
				BitRate          string    `json:"bit_rate"`
			} `json:"streams"`
			Format struct {
				Filename        string            `json:"filename"`
				NumberOfStreams integer           `json:"nb_streams"`
				Duration        float             `json:"duration"`
				Size            integer           `json:"size"`
				Bitrate         integer           `json:"bit_rate"`
				Tags            map[string]string `json:"tags"`
			} `json:"format"`
		}
		{
			proc := exec.Command(
				"ffprobe",
				"-v", "error",
				"-i", inputFilepath,
				"-print_format", "json",
				"-show_format",
				"-show_streams",
			)
			output := bytes.Buffer{}
			proc.Stderr = &output
			proc.Stdout = &output
			if err := proc.Run(); err != nil {
				errorMessage = "Probe Error"
				errorOutput = output.String()
				return
			}

			// Parse JSON Output
			if err := json.Unmarshal(output.Bytes(), &Probe); err != nil {
				errorMessage = "Invalid or Malformed Probe Output"
				errorOutput = err.Error()
				return
			}

			// Sanity Checks
			for _, s := range Probe.Streams {
				switch s.CodecType {
				case "video":
					encodeVideoStreams++
					encodeVideoFramerate = min(int(s.AverageFrameRate), VIDEO_FPS_LIMIT)
					encodeVideoHeight = min(s.Height, VIDEO_HEIGHT_LIMIT)
				case "audio":
					if encodeAudioStreams < AUDIO_STREAMS_LIMIT {
						encodeAudioStreams++
					}
				}
			}
			if encodeVideoStreams == 0 {
				errorMessage = "No Video Streams Present"
				errorOutput = "N/A"
				return
			}
		}

		// Step 3. Encode Video
		{
			proc := exec.Command(
				"ffmpeg",
				"-y",
				"-v", "error",
				"-progress", "pipe:1",
				"-i", inputFilepath,
				"-c:v", VIDEO_CODEC,
				"-pix_fmt", VIDEO_PIXEL_FORMAT,
				"-preset", VIDEO_PRESET,
				"-qp", VIDEO_QUALITY,
				"-vf", "scale=-1:"+strconv.Itoa(encodeVideoHeight),
				"-r", strconv.Itoa(encodeVideoFramerate),
				"-c:a", AUDIO_CODEC,
				"-b:a", AUDIO_BITRATE,
				"-ac", AUDIO_CHANNELS,
				"-filter_complex", "amerge=inputs="+strconv.Itoa(encodeAudioStreams),
				path.Join(outputDirectory, OUTPUT_FILENAME_VIDEO),
			)
			output := bytes.Buffer{}
			proc.Stderr = &output

			// Stream Progress
			p, _ := proc.StdoutPipe()
			go func() {
				for {
					// Parse Progress
					b := make([]byte, 256)
					n, err := p.Read(b)
					if err != nil {
						return
					}
					m := map[string]string{}
					for _, line := range strings.Split(string(b[:n]), "\n") {
						s := strings.SplitN(line, "=", 2)
						if len(s) == 2 {
							m[s[0]] = strings.TrimSpace(s[1])
						}
					}
					switch m["progress"] {
					case "continue":
						o, err := strconv.ParseFloat(m["out_time_us"], 64)
						if err != nil {
							return
						}
						// Convert to Milliseconds
						duration := math.Floor(float64(Probe.Format.Duration) * 1000)
						progress := o / 1000
						// Format as Percentage
						percentd := strconv.FormatFloat((progress/duration)*100, 'f', 0, 64)
						SendEvent(userID, "VIDEO_PROCESSING_PROGRESS", videoID, percentd)
					case "end":
						return
					}
				}
			}()
			if err := proc.Run(); err != nil {
				errorMessage = "Encoding Error"
				errorOutput = output.String()
				return
			}
		}

		// Step 4. Generate Thumbnail
		{
			proc := exec.Command(
				"ffmpeg",
				"-y",
				"-v", "error",
				"-i", inputFilepath,
				"-vf", "scale=-1:"+strconv.Itoa(encodeVideoHeight),
				"-frames:v", "1",
				path.Join(outputDirectory, OUTPUT_FILENAME_THUMBNAIL),
			)
			if b, err := proc.CombinedOutput(); err != nil {
				errorMessage = "Thumbnail Error"
				errorOutput = string(bytes.TrimSpace(b))
				return
			}
		}

		// Step 5. Mark Video as Finished
		if _, err := DB.Exec("UPDATE videos SET status = 'FINISH' WHERE id = $1", videoID); err != nil {
			errorMessage = "Database Error"
			errorOutput = err.Error()
			return
		}
		SendEvent(userID, "VIDEO_PROCESSING_COMPLETE", videoID, videoCreated)
		log.Printf("[encoders][%d] Video Processed: %s\n", workerId, videoID)
	}
}

// Some custom types since some values are wrapped in quotes and it trips up the json unmarshaller

// Parses the 1/60000 or whatever as a rounded integer
type framerate int

func (f *framerate) UnmarshalJSON(b []byte) error {
	s := strings.SplitN(strings.Trim(string(b), "\""), "/", 2)
	if len(s) != 2 {
		return errors.New("incorrect amount of segments")
	}
	x, err := strconv.ParseFloat(s[0], 32)
	if err != nil {
		return err
	}
	y, err := strconv.ParseFloat(s[1], 32)
	if err != nil {
		return err
	}
	*f = framerate(math.Round(x / y))
	return nil
}

// Parses a number string as an integer (e.g. "123" => 123)
type integer int

func (f *integer) UnmarshalJSON(b []byte) error {
	v, err := strconv.Atoi(strings.Trim(string(b), "\""))
	if err != nil {
		return err
	}
	*f = integer(v)
	return nil
}

// Parses a float string as a float64 (e.g. "123.123" => 123.123)
type float float64

func (f *float) UnmarshalJSON(b []byte) error {
	v, err := strconv.ParseFloat(strings.Trim(string(b), "\""), 64)
	if err != nil {
		return err
	}
	*f = float(v)
	return nil
}
