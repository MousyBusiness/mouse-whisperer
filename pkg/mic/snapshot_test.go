package mic

import (
	"encoding/binary"
	"os"
	"testing"
)

func writeSamples(t *testing.T, s *snippet, samples []int16) {
	t.Helper()
	for _, v := range samples {
		if err := binary.Write(s.file, binary.LittleEndian, v); err != nil {
			t.Fatal(err)
		}
	}
	s.nSamples += len(samples)
}

func makeSamples(n int, value int16) []int16 {
	out := make([]int16, n)
	for i := range out {
		out[i] = value
	}
	return out
}

func readWav(t *testing.T, path string) (riffSize, dataSize int32, samples []int16) {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) < 44 {
		t.Fatalf("wav too short: %d bytes", len(b))
	}
	riffSize = int32(binary.LittleEndian.Uint32(b[4:8]))
	dataSize = int32(binary.LittleEndian.Uint32(b[40:44]))
	data := b[44:]
	samples = make([]int16, len(data)/2)
	for i := range samples {
		samples[i] = int16(binary.LittleEndian.Uint16(data[i*2 : i*2+2]))
	}
	return riffSize, dataSize, samples
}

// TestSnapshotWAV pins the assumption streaming rests on: a snapshot taken
// while the session file is still being written is a valid padded WAV, the
// source is untouched, and finalizing the source later still works.
func TestSnapshotWAV(t *testing.T) {
	snip := createSnippetWav()
	defer os.Remove(snip.name)

	writeSamples(t, snip, makeSamples(1000, 7))

	snapPath, err := SnapshotWAV(snip.name, snip.nSamples)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(snapPath)

	required := sampleRate * channelCount * minTime
	riffSize, dataSize, samples := readWav(t, snapPath)
	if int(dataSize) != required*bytesPerSample {
		t.Fatalf("data size: got %d, want %d", dataSize, required*bytesPerSample)
	}
	if int(riffSize) != 44+required*bytesPerSample-8 {
		t.Fatalf("riff size: got %d, want %d", riffSize, 44+required*bytesPerSample-8)
	}
	if len(samples) != required {
		t.Fatalf("samples: got %d, want %d", len(samples), required)
	}
	for i := 0; i < 1000; i++ {
		if samples[i] != 7 {
			t.Fatalf("sample %d: got %d, want 7", i, samples[i])
		}
	}
	for i := 1000; i < required; i++ {
		if samples[i] != 0 {
			t.Fatalf("padding sample %d: got %d, want 0", i, samples[i])
		}
	}

	// Source must be untouched: keep writing, snapshot again mid-stream
	writeSamples(t, snip, makeSamples(500, 9))
	snapPath2, err := SnapshotWAV(snip.name, snip.nSamples)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(snapPath2)

	_, _, samples2 := readWav(t, snapPath2)
	if samples2[1000] != 9 || samples2[1499] != 9 {
		t.Fatalf("second snapshot missing later samples: %d %d", samples2[1000], samples2[1499])
	}

	// Finalizing the source afterwards must still produce a correct file
	snip.finalizeWav()
	riffSize, dataSize, samples = readWav(t, snip.name)
	if int(dataSize) != required*bytesPerSample {
		t.Fatalf("final data size: got %d, want %d", dataSize, required*bytesPerSample)
	}
	if samples[999] != 7 || samples[1000] != 9 || samples[1499] != 9 || samples[1500] != 0 {
		t.Fatalf("final samples corrupted: %d %d %d %d", samples[999], samples[1000], samples[1499], samples[1500])
	}
	_ = riffSize
}

// TestSnapshotWAVLongSession covers the no-padding path (utterance > minTime)
func TestSnapshotWAVLongSession(t *testing.T) {
	snip := createSnippetWav()
	defer os.Remove(snip.name)

	n := sampleRate*minTime + 5000
	writeSamples(t, snip, makeSamples(n, 3))

	snapPath, err := SnapshotWAV(snip.name, snip.nSamples)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(snapPath)

	_, dataSize, samples := readWav(t, snapPath)
	if int(dataSize) != n*bytesPerSample {
		t.Fatalf("data size: got %d, want %d", dataSize, n*bytesPerSample)
	}
	if samples[n-1] != 3 {
		t.Fatalf("last sample: got %d, want 3", samples[n-1])
	}
}

// TestJoinWAVs: clips concatenate in order into one valid WAV whose header
// matches the PCM written by both sources
func TestJoinWAVs(t *testing.T) {
	var paths []string
	for _, v := range []int16{3, 9} {
		snip := createSnippetWav()
		defer os.Remove(snip.name)
		writeSamples(t, snip, makeSamples(500, v))
		snip.finalizeWav()
		paths = append(paths, snip.name)
	}

	joined, err := JoinWAVs(paths)
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(joined)

	// finalizeWav pads each clip to the minimum duration
	each := sampleRate * channelCount * minTime
	riffSize, dataSize, samples := readWav(t, joined)
	if int(dataSize) != 2*each*bytesPerSample || int(riffSize) != 44+int(dataSize)-8 {
		t.Fatalf("sizes: riff %d data %d", riffSize, dataSize)
	}
	if len(samples) != 2*each {
		t.Fatalf("samples: got %d, want %d", len(samples), 2*each)
	}
	if samples[0] != 3 || samples[499] != 3 || samples[500] != 0 || samples[each] != 9 || samples[each+499] != 9 {
		t.Fatalf("clips out of order or misaligned")
	}
}
