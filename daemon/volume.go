package daemon

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jfreymuth/pulse/proto"
)

func ForceSpeakerOutput() error {
	tryConnect := func(url string) (*proto.Client, net.Conn, error) {
		if url == "" {
			return nil, nil, nil
		}
		log.Printf("PulseAudio: Attempting connect to %s", url)
		return proto.Connect(url)
	}

	uid := os.Getuid()
	candidates := []string{
		os.Getenv("PULSE_SERVER"),
		fmt.Sprintf("unix:/run/user/%d/pulse/native", uid),
		fmt.Sprintf("unix:@/run/user/%d/pulse/native", uid),
	}

	var c *proto.Client
	var conn net.Conn
	var err error

	for _, url := range candidates {
		if url == "" {
			continue
		}
		c, conn, err = tryConnect(url)
		if err == nil {
			log.Printf("PulseAudio: Connected successfully to %s", url)
			break
		}
		log.Printf("PulseAudio: Failed to connect to %s: %v", url, err)
	}

	if c == nil {
		log.Printf("PulseAudio: All connection attempts failed. Last error: %v", err)
		return fmt.Errorf("connection failed: %w", err)
	}
	defer conn.Close()

	props := proto.PropList{
		"application.name": proto.PropListString("circadia-daemon"),
	}
	if err := c.Request(&proto.SetClientName{Props: props}, &proto.SetClientNameReply{}); err != nil {
		log.Printf("PulseAudio: SetClientName error: %v", err)
	}

	var sinks proto.GetSinkInfoListReply
	err = c.Request(&proto.GetSinkInfoList{}, &sinks)
	if err != nil {
		log.Printf("PulseAudio: ListSinks error: %v", err)
		return fmt.Errorf("list sinks failed: %w", err)
	}

	var bestSinkIndex uint32 = 0xFFFFFFFF
	var bestSinkName string
	bestScore := -1

	for _, s := range sinks {
		if s == nil {
			continue
		}
		score := 0
		name := strings.ToLower(s.SinkName)
		desc := strings.ToLower(s.Device)

		if strings.Contains(name, "speaker") || strings.Contains(desc, "speaker") || strings.Contains(name, "primary") {
			score += 10
		}
		if strings.Contains(name, "pci") || strings.Contains(name, "platform") {
			score += 2
		}
		if strings.Contains(name, "usb") || strings.Contains(name, "bluez") {
			score -= 5
		}

		log.Printf("PulseAudio: Found Sink: %s (Desc: %s) Score: %d", s.SinkName, s.Device, score)

		if score > bestScore {
			bestScore = score
			bestSinkIndex = s.SinkIndex
			bestSinkName = s.SinkName
		}
	}

	if bestSinkIndex == 0xFFFFFFFF {
		log.Println("PulseAudio: No suitable sink found")
		return fmt.Errorf("no suitable sink found")
	}

	log.Printf("PulseAudio: Selected Sink: %s (Index: %d)", bestSinkName, bestSinkIndex)

	var channels int
	for _, s := range sinks {
		if s.SinkIndex == bestSinkIndex {
			channels = len(s.ChannelMap)
			break
		}
	}
	if channels == 0 {
		channels = 2
	}

	vol := make(proto.ChannelVolumes, channels)
	for i := range vol {
		vol[i] = 0x10000
	}

	if err := c.Request(&proto.SetSinkVolume{
		SinkIndex:      bestSinkIndex,
		ChannelVolumes: vol,
	}, nil); err != nil {
		log.Printf("PulseAudio: SetVolume error: %v", err)
	}

	if err := c.Request(&proto.SetSinkMute{
		SinkIndex: bestSinkIndex,
		Mute:      false,
	}, nil); err != nil {
		log.Printf("PulseAudio: SetMute error: %v", err)
	}

	myPid := strconv.Itoa(os.Getpid())

	for IsRinging() {
		if err := c.Request(&proto.SetDefaultSink{SinkName: bestSinkName}, nil); err != nil {
			log.Printf("PulseAudio: SetDefaultSink error: %v", err)
		}

		var inputs proto.GetSinkInputInfoListReply
		if err := c.Request(&proto.GetSinkInputInfoList{}, &inputs); err != nil {
			log.Printf("PulseAudio: GetSinkInputList error: %v", err)
			time.Sleep(1 * time.Second)
			continue
		}

		for _, input := range inputs {
			if input == nil {
				continue
			}

			isMatch := false
			if pidBytes, ok := input.Properties["application.process.id"]; ok {
				pidVal := strings.TrimRight(string(pidBytes), "\x00")
				if pidVal == myPid {
					isMatch = true
				}
			}

			if !isMatch {
				if nameBytes, ok := input.Properties["application.name"]; ok {
					nameVal := strings.TrimRight(string(nameBytes), "\x00")
					if nameVal == "circadia" || nameVal == "circadia-daemon" {
						isMatch = true
					}
				}
			}

			if isMatch {
				if input.SinkIndex == bestSinkIndex {
					continue
				}

				log.Printf("PulseAudio: Moving our stream (Index %d) to sink %d", input.SinkInputIndex, bestSinkIndex)
				if err := c.Request(&proto.MoveSinkInput{
					SinkInputIndex: input.SinkInputIndex,
					DeviceIndex:    bestSinkIndex,
				}, nil); err != nil {
					log.Printf("PulseAudio: MoveSinkInput error: %v", err)
				} else {
					c.Request(&proto.SetSinkVolume{
						SinkIndex:      bestSinkIndex,
						ChannelVolumes: vol,
					}, nil)
					c.Request(&proto.SetSinkMute{SinkIndex: bestSinkIndex, Mute: false}, nil)
				}

			}
		}

		time.Sleep(500 * time.Millisecond)
	}
	return nil
}
