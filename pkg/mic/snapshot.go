package mic

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// SnapshotWAV copies the first nSamples of the (possibly still-growing)
// session WAV at srcPath into a new temp WAV, pads it to the minimum
// duration whisper needs, patches the RIFF/data sizes and returns the
// snapshot path. srcPath is never modified. Reading a live session file is
// safe because the mic writer is unbuffered and nSamples is only advanced
// after the bytes are on disk.
//
// finalizeWav is deliberately not reused here: it panics via chk() and
// closes the file, either of which would kill the streaming worker.
func SnapshotWAV(srcPath string, nSamples int) (string, error) {
	src, err := os.Open(srcPath)
	if err != nil {
		return "", fmt.Errorf("snapshot: open source: %w", err)
	}
	defer src.Close()

	dst, err := os.CreateTemp(os.TempDir(), "snapshot_*.wav")
	if err != nil {
		return "", fmt.Errorf("snapshot: create: %w", err)
	}
	defer dst.Close()

	fail := func(err error) (string, error) {
		os.Remove(dst.Name())
		return "", err
	}

	if _, err := io.CopyN(dst, src, 44+int64(nSamples)*bytesPerSample); err != nil {
		return fail(fmt.Errorf("snapshot: copy: %w", err))
	}

	total := nSamples
	if required := sampleRate * channelCount * minTime; total < required {
		silence := make([]byte, (required-total)*bytesPerSample)
		if _, err := dst.Write(silence); err != nil {
			return fail(fmt.Errorf("snapshot: pad: %w", err))
		}
		total = required
	}

	dataSize := total * bytesPerSample
	if _, err := dst.Seek(4, io.SeekStart); err != nil {
		return fail(fmt.Errorf("snapshot: seek riff size: %w", err))
	}
	if err := binary.Write(dst, binary.LittleEndian, int32(44+dataSize-8)); err != nil {
		return fail(fmt.Errorf("snapshot: write riff size: %w", err))
	}
	if _, err := dst.Seek(40, io.SeekStart); err != nil {
		return fail(fmt.Errorf("snapshot: seek data size: %w", err))
	}
	if err := binary.Write(dst, binary.LittleEndian, int32(dataSize)); err != nil {
		return fail(fmt.Errorf("snapshot: write data size: %w", err))
	}

	return dst.Name(), nil
}

// JoinWAVs concatenates finalized session WAVs, in order, into one new
// temp WAV and returns its path; the caller removes it. Whisper hears the
// joined recording as continuous speech, restoring the sentence context
// that silence breaks split across sessions.
func JoinWAVs(paths []string) (string, error) {
	dst, err := os.CreateTemp(os.TempDir(), "joined_*.wav")
	if err != nil {
		return "", fmt.Errorf("join: create: %w", err)
	}
	defer dst.Close()

	fail := func(err error) (string, error) {
		os.Remove(dst.Name())
		return "", err
	}

	if _, err := dst.Write(make([]byte, 44)); err != nil {
		return fail(fmt.Errorf("join: header: %w", err))
	}
	dataSize := 0
	for _, p := range paths {
		n, err := appendWAVData(dst, p)
		if err != nil {
			return fail(err)
		}
		dataSize += n
	}
	if dataSize == 0 {
		return fail(fmt.Errorf("join: no audio"))
	}

	if _, err := dst.Seek(0, io.SeekStart); err != nil {
		return fail(fmt.Errorf("join: seek header: %w", err))
	}
	if err := writeWAVHeader(dst, dataSize); err != nil {
		return fail(fmt.Errorf("join: write header: %w", err))
	}
	return dst.Name(), nil
}

// appendWAVData copies the PCM data of the WAV at path (as this package
// writes them: 44-byte header) to w, returning the byte count
func appendWAVData(w io.Writer, path string) (int, error) {
	src, err := os.Open(path)
	if err != nil {
		return 0, fmt.Errorf("join: open %s: %w", path, err)
	}
	defer src.Close()
	info, err := src.Stat()
	if err != nil {
		return 0, fmt.Errorf("join: stat %s: %w", path, err)
	}
	// whole samples only: a stray byte would misalign every later clip
	n := info.Size() - 44
	n -= n % bytesPerSample
	if n <= 0 {
		return 0, nil
	}
	if _, err := src.Seek(44, io.SeekStart); err != nil {
		return 0, fmt.Errorf("join: seek %s: %w", path, err)
	}
	if _, err := io.CopyN(w, src, n); err != nil {
		return 0, fmt.Errorf("join: copy %s: %w", path, err)
	}
	return int(n), nil
}

func writeWAVHeader(w io.Writer, dataSize int) error {
	fields := []any{
		[]byte("RIFF"), int32(44 + dataSize - 8), []byte("WAVE"),
		[]byte("fmt "), int32(16), int16(1), int16(channelCount), int32(sampleRate),
		int32(sampleRate * channelCount * bytesPerSample), int16(channelCount * bytesPerSample), int16(bitsPerSample),
		[]byte("data"), int32(dataSize),
	}
	for _, f := range fields {
		if err := binary.Write(w, binary.LittleEndian, f); err != nil {
			return err
		}
	}
	return nil
}
