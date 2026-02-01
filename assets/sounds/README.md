# Sound Assets

Place WAV files in this directory to enable game audio.

## Expected Files

| File | Event | Description |
|------|-------|-------------|
| `game_start.wav` | Game starts | Played when the game begins |
| `countdown.wav` | Countdown tick | Played during 3-2-1 countdown |
| `capture.wav` | Territory captured | Played when you capture territory |
| `death.wav` | Player death | Played when you die |
| `game_over.wav` | Game over | Played when the game ends |
| `powerup.wav` | Powerup collected | Played when collecting a powerup |

## Requirements

- Format: WAV (16-bit PCM recommended)
- Duration: Keep sounds short (< 1 second) for responsiveness
- Sample rate: 44100 Hz recommended (all sounds should use the same sample rate)

## Notes

- Missing sound files are silently ignored
- The game will work without any sound files
- Use `--no-sound` flag to disable audio entirely
