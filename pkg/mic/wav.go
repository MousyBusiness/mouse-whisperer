package mic

import (
	"encoding/binary"
	"os"
	"sync/atomic"
	"time"
)

const (
	sampleRate            = 16000
	bytesPerSample        = 2
	bitsPerSample         = bytesPerSample * 8
	channelCount          = 1
	commandBreakThreshold = 500
	inputBreakThreshold   = 3000
	minTime               = 2
	silenceThreshold      = 2.0e+08 // Adjust this threshold as needed
)

var snippetCounter atomic.Uint64

type snippet struct {
	id       uint64
	name     string
	file     *os.File
	nSamples int
	created  time.Time
}

func (s *snippet) session() RecordingSession {
	return RecordingSession{
		ID:               s.id,
		FileName:         s.name,
		RecordingStarted: s.created,
	}
}

func (s *snippet) finalizeWav() {

	// fill to with silence
	requiredSamples := sampleRate * channelCount * minTime
	if s.nSamples < requiredSamples {
		// Calculate the number of silent samples to add
		silenceSamples := requiredSamples - s.nSamples

		// Create a slice of bytes for the silent samples
		silence := make([]byte, silenceSamples*bytesPerSample)

		// Seek to the end of the current data
		_, err := s.file.Seek(0, os.SEEK_END)
		chk(err)

		// Write the silent samples to the file
		_, err = s.file.Write(silence)
		chk(err)

		// Update the number of samples
		s.nSamples += silenceSamples
	}

	// fill in missing sizes
	dataSize := bytesPerSample * s.nSamples // 2 bytes per sample for 16-bit audio
	fileSize := 44 + dataSize - 8           // Adjusted for WAV -8 for RIFF + filesize header

	// Update RIFF chunk size
	_, err := s.file.Seek(4, 0)
	chk(err)
	chk(binary.Write(s.file, binary.LittleEndian, int32(fileSize)))

	// Update data chunk size
	_, err = s.file.Seek(40, 0)
	chk(err)
	chk(binary.Write(s.file, binary.LittleEndian, int32(dataSize)))

	chk(s.file.Close())
}

func createSnippetWav() *snippet {
	f, err := os.CreateTemp(os.TempDir(), "transcription_*.wav")
	chk(err)

	//https://docs.fileformat.com/audio/wav/
	// RIFF header
	_, err = f.WriteString("RIFF")
	chk(err)
	chk(binary.Write(f, binary.LittleEndian, int32(0))) // Placeholder for file size
	_, err = f.WriteString("WAVE")
	chk(err)

	// fmt sub-chunk
	_, err = f.WriteString("fmt ")
	chk(err)
	chk(binary.Write(f, binary.LittleEndian, int32(16)))                                      // Sub-chunk size (16 for PCM)
	chk(binary.Write(f, binary.LittleEndian, int16(1)))                                       // Audio format (1 for PCM)
	chk(binary.Write(f, binary.LittleEndian, int16(channelCount)))                            // Num channels
	chk(binary.Write(f, binary.LittleEndian, int32(sampleRate)))                              // Sample rate
	chk(binary.Write(f, binary.LittleEndian, int32(sampleRate*channelCount*bitsPerSample/8))) // Byte rate (sample rate * num channels * bits per sample / 8)
	chk(binary.Write(f, binary.LittleEndian, int16(channelCount*bitsPerSample/8)))            // Block align (num channels * bits per sample / 8)
	chk(binary.Write(f, binary.LittleEndian, int16(bitsPerSample)))                           // Bits per sample

	// data sub-chunk
	_, err = f.WriteString("data")
	chk(err)
	chk(binary.Write(f, binary.LittleEndian, int32(0))) // Placeholder for data chunk size

	return &snippet{
		id:       snippetCounter.Add(1),
		name:     f.Name(),
		file:     f,
		nSamples: 0,
		created:  time.Now(),
	}
}

func chk(err error) {
	if err != nil {
		panic(err)
	}
}
