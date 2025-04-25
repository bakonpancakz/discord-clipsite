# discord-clipsite
Basic website where you can login with Discord to Upload and Share clips (videos), bypassing the 10MB Upload Limit.

- [discord-clipsite](#discord-clipsite)
  - [Building](#building)
  - [Database](#database)
  - [Configuration](#configuration)
      - [Program Options](#program-options)
      - [Encoder Options](#encoder-options)


## Building
Some dependencies `(notably SQLite)` requires **CGO** to properly compile the program. You can follow this [helpful guide](https://github.com/go101/go101/wiki/CGO-Environment-Setup) to set it up on your machine. 

Additionally, This program requires that both `FFmpeg` and `FFprobe` to be installed and available in your systems PATH.

Afterwards set the appropriate **CGO** environment variable `export CGO_ENABLED=1` and build it as normal.
```
go build -o shareclip main.go
```


## Database
This program stores all files in a single directory for simplicity and portability. 
By default, it creates a `data` folder in the current working directory. 
Below is an example layout after one video has been processed:

```py
.
|__ shareclip.elf                   # Compiled program
|__ /data
    |__ database.db                 # A SQLite Database which stores all user and video metadata.
    |__ /public                     # Note: All Files in the directory are public!
    |   |__ /avQCfm4YEz5            
    |       |__ video.mp4           # Uses value from ENCODER_OUTPUT_FILENAME_VIDEO
    |       |__ thumbnail.webp      # Uses value from ENCODER_OUTPUT_FILENAME_THUMBNAIL
    |
    |__ /video                      # Original uploaded videos
        |__ avQCfm4YEz5             # Stored without a file extension. 
                                    #   These can be safely deleted, although should be kept 
                                    #   for possible future re-encodes.
```

## Configuration
This program can be configured via a `.env` file in the working directory or by exporting them. 

**Required** Variables without a default value (denoted with a `...` in the default column) will throw an error and close the application with exit code 2.

#### Program Options
| Key               | Default          | Description                                         |
| :---------------- | :--------------- | :-------------------------------------------------- |
| DATA              | `data`           | Path to the Data Directory                          |
| HTTP_BIND         | `localhost:8080` | Address to listen to requests on                    |
| TLS_ENABLED       | `false`          | Set this to true to enable TLS v1.3 for your Server |
| TLS_CERT          | `tls_crt.pem`    | The Path to your SSL/TLS Certificate                |
| TLS_KEY           | `tls_key.pem`    | The Path to your SSL/TLS Key                        |
| TLS_CA            | `tls_ca.pem`     | The Path to your SSL/TLS CA Bundle                  |
| DISCORD_REDIRECT  | `...`            | Your Discord Redirect URI                           |
| DISCORD_CLIENT_ID | `...`            | Your Discord Client ID                              |
| DISCORD_SECRET    | `...`            | Your Discord Client Secret                          |

#### Encoder Options
> ⚠ **Warning:** These are advanced options, only modify these if you know how to use FFmpeg.

> 🔎 **Note:** Changing the output filenames will not re-encode or update all the old filenames in the public data directory.

| Key                               | Default                        | Description                                                                                    |
| :-------------------------------- | :----------------------------- | :--------------------------------------------------------------------------------------------- |
| ENCODER_USE_HARDWARE              | `true`                         | Use Hardware Encoding? Set to anything other than `true` to disable.                           |
| ENCODER_WORKERS                   | `1`                            | Amount of simultaneous video encodings                                                         |
| ENCODER_OUTPUT_FILENAME_VIDEO     | `video.mp4`                    | Output Filename for Video                                                                      |
| ENCODER_OUTPUT_FILENAME_THUMBNAIL | `preview.webp`                 | Output Filename for Thumbnail                                                                  |
| ENCODER_VIDEO_HEIGHT_LIMIT        | `1080`                         | Maximum Output Video Height, scales appropriately using `-1` for width.                        |
| ENCODER_VIDEO_FPS_LIMIT           | `60`                           | Maximum Output Video Framerate                                                                 |
| ENCODER_VIDEO_STREAMS_LIMIT       | `1`                            | Maximum Video Streams to Process                                                               |
| ENCODER_VIDEO_PIXEL_FORMAT        | `yuv420p`                      | Output Pixel Format, should not be modifed for compatibility                                   |
| ENCODER_VIDEO_PRESET              | `fast`                         | Video Preset to use, set to `slow` for better compression                                      |
| ENCODER_VIDEO_QUALITY             | `27`                           | Video "Quality" setting for `-qp` argument                                                     |
| ENCODER_VIDEO_CODEC               | `libx264`                      | Fallback Video Encoder, should be software based.                                              |
| ENCODER_VIDEO_HARDWARE_CODEC      | `h264_nvenc,h264_qsv,h264_amf` | Hardware encoders to test for, ordered by highest quality first and delimited with a comma (,) |
| ENCODER_AUDIO_STREAMS_LIMIT       | `6`                            | Maximum amount of audio streams to merge using `amerge`, set to 6 to support OBS               |
| ENCODER_AUDIO_BITRATE             | `320K`                         | Audio Bitrate                                                                                  |
| ENCODER_AUDIO_CODEC               | `aac`                          | Audio Encoder, should be set to something your container supports                              |
| ENCODER_AUDIO_CHANNELS            | `2`                            | Audio Channels, should not be modifed for compatibility                                        |
