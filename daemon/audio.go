package daemon

import (
	"circadia/storage"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gopxl/beep/v2"
	"github.com/gopxl/beep/v2/mp3"
	"github.com/gopxl/beep/v2/speaker"
	"github.com/gopxl/beep/v2/vorbis"
	"github.com/gopxl/beep/v2/wav"
)

var (
	ctrl               *beep.Ctrl
	streamer           beep.StreamSeekCloser
	speakerInitialized bool
	speakerMu          sync.Mutex
	speakerSampleRate  = beep.SampleRate(48000)
)

func resolveAudioPath() string {
	customPath, _ := storage.GetAlarmAudioPath()
	log.Printf("[Audio] Resolving path. Custom: %s", customPath)
	if customPath != "" {
		if _, err := os.Stat(customPath); err == nil {
			return customPath
		}
		log.Printf("[Audio] Custom audio file not found: %s", customPath)
	}

	path := "assets/alarm/default.ogg"
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		return path
	}

	flatpakPath := "/app/share/circadia/assets/alarm/default.ogg"
	if info, err := os.Stat(flatpakPath); err == nil && !info.IsDir() {
		return flatpakPath
	}

	return path
}

func initSpeaker() error {
	speakerMu.Lock()
	defer speakerMu.Unlock()

	if speakerInitialized {
		return nil
	}

	log.Println("[Audio] Initializing speaker...")
	err := speaker.Init(speakerSampleRate, speakerSampleRate.N(time.Second/5))
	if err != nil {
		log.Printf("[Audio] Speaker Init failed: %v", err)
		return err
	}

	speakerInitialized = true
	log.Println("[Audio] Speaker initialized successfully.")
	return nil
}

func playSound(path string, loopAudio bool) error {
	log.Printf("[Audio] playSound called for: %s (loop=%v)", path, loopAudio)

	if err := initSpeaker(); err != nil {
		return fmt.Errorf("failed to init speaker: %w", err)
	}

	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open audio file %s: %w", path, err)
	}

	var s beep.StreamSeekCloser
	var format beep.Format
	var errDecode error

	lowerPath := strings.ToLower(path)
	log.Printf("[Audio] Decoding file type: %s", lowerPath)

	if strings.HasSuffix(lowerPath, ".mp3") {
		s, format, errDecode = mp3.Decode(f)
	} else if strings.HasSuffix(lowerPath, ".wav") {
		s, format, errDecode = wav.Decode(f)
	} else {
		s, format, errDecode = vorbis.Decode(f)
	}

	if errDecode != nil {
		f.Close()
		return fmt.Errorf("failed to decode audio: %w", errDecode)
	}

	log.Printf("[Audio] Decoded. Format: %v. Resampling needed: %v", format, format.SampleRate != speakerSampleRate)

	var stream beep.Streamer = s
	if loopAudio {
		var errLoop error
		stream, errLoop = beep.Loop2(s)
		if errLoop != nil {
			f.Close()
			return fmt.Errorf("failed to create loop: %w", errLoop)
		}
	}

	if format.SampleRate != speakerSampleRate {
		stream = beep.Resample(4, format.SampleRate, speakerSampleRate, stream)
	}

	newCtrl := &beep.Ctrl{Streamer: stream, Paused: false}

	log.Println("[Audio] Acquiring speaker lock...")
	speaker.Lock()

	if ctrl != nil {
		ctrl.Paused = true
		ctrl = nil
	}
	if streamer != nil {
		streamer.Close()
	}

	streamer = s
	ctrl = newCtrl

	speaker.Unlock()

	log.Println("[Audio] Playing started (calling speaker.Play)...")
	speaker.Play(newCtrl)

	return nil
}

func StartAlarmSound() {
	log.Println("[Audio] StartAlarmSound requested")
	path := resolveAudioPath()
	if err := playSound(path, true); err != nil {
		log.Printf("[Audio] StartAlarmSound Error: %v", err)
		return
	}

	go func() {
		time.Sleep(500 * time.Millisecond)
		ForceSpeakerOutput()
	}()
}

func PreviewAudio(path string) error {
	log.Printf("[Audio] PreviewAudio requested for: %s", path)
	StopAlarmSound()

	if path == "" {
		path = resolveAudioPath()
	}

	err := playSound(path, false)
	if err != nil {
		log.Printf("[Audio] PreviewAudio failed: %v", err)
	}
	return err
}

func StopAlarmSound() {
	log.Println("[Audio] StopAlarmSound called")
	speaker.Lock()
	defer speaker.Unlock()

	if ctrl != nil {
		ctrl.Paused = true
		ctrl = nil
	}
	if streamer != nil {
		streamer.Close()
		streamer = nil
	}
	log.Println("[Audio] Stopped.")
}
