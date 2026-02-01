package audio

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/speaker"
	"github.com/faiface/beep/wav"
)

// Sound names for game events
const (
	SoundGameStart = "game_start"
	SoundCountdown = "countdown"
	SoundCapture   = "capture"
	SoundDeath     = "death"
	SoundGameOver  = "game_over"
	SoundPowerup   = "powerup"
)

var (
	sounds      map[string]*beep.Buffer
	soundsMu    sync.RWMutex
	initialized bool
	enabled     = true
	sampleRate  beep.SampleRate
)

// Init initializes the audio system and loads sounds from the assets directory.
// If initialization fails, audio will be silently disabled.
func Init(assetsPath string) {
	soundsMu.Lock()
	defer soundsMu.Unlock()

	if initialized {
		return
	}

	sounds = make(map[string]*beep.Buffer)
	soundsDir := filepath.Join(assetsPath, "sounds")

	// Try to load all sound files
	soundFiles := []string{
		SoundGameStart,
		SoundCountdown,
		SoundCapture,
		SoundDeath,
		SoundGameOver,
		SoundPowerup,
	}

	var firstFormat *beep.Format
	for _, name := range soundFiles {
		path := filepath.Join(soundsDir, name+".wav")
		buffer, format := loadWAV(path)
		if buffer != nil {
			sounds[name] = buffer
			if firstFormat == nil {
				firstFormat = format
			}
		}
	}

	// If no sounds were loaded, disable audio
	if len(sounds) == 0 {
		enabled = false
		return
	}

	// Initialize speaker with the sample rate of the first loaded sound
	if firstFormat != nil {
		sampleRate = firstFormat.SampleRate
		err := speaker.Init(sampleRate, sampleRate.N(time.Second/10))
		if err != nil {
			enabled = false
			return
		}
	}

	initialized = true
}

// loadWAV loads a WAV file and returns a buffer and format
func loadWAV(path string) (*beep.Buffer, *beep.Format) {
	f, err := os.Open(path)
	if err != nil {
		return nil, nil
	}
	defer f.Close()

	streamer, format, err := wav.Decode(f)
	if err != nil {
		return nil, nil
	}
	defer streamer.Close()

	buffer := beep.NewBuffer(format)
	buffer.Append(streamer)
	return buffer, &format
}

// Play plays the specified sound asynchronously.
// If the sound is not loaded or audio is disabled, this is a no-op.
func Play(name string) {
	if !enabled {
		return
	}

	soundsMu.RLock()
	buffer, ok := sounds[name]
	soundsMu.RUnlock()

	if !ok || buffer == nil {
		return
	}

	// Play sound asynchronously
	streamer := buffer.Streamer(0, buffer.Len())
	speaker.Play(streamer)
}

// SetEnabled enables or disables audio playback
func SetEnabled(e bool) {
	soundsMu.Lock()
	enabled = e
	soundsMu.Unlock()
}

// IsEnabled returns whether audio is enabled
func IsEnabled() bool {
	soundsMu.RLock()
	defer soundsMu.RUnlock()
	return enabled
}
