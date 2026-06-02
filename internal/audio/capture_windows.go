//go:build windows

package audio

import (
	"encoding/binary"
	"fmt"
	"math"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
	"unsafe"

	ole "github.com/go-ole/go-ole"
	wca "github.com/moutend/go-wca/pkg/wca"
)

// acInitializeFixed calls IAudioClient::Initialize through the COM vtable with
// architecture-correct argument marshaling.
//
// go-wca's own Initialize passes each REFERENCE_TIME (int64) as a single uintptr
// (see IAudioClient_windows.go). That is correct on amd64 (uintptr is 64-bit) but
// BROKEN on 386: the two by-value int64 stack arguments are truncated to 32 bits
// each, shifting the format/GUID pointers and making Initialize return
// E_INVALIDARG. Here we split each int64 into low/high dwords on 32-bit builds so
// the stdcall stack matches what WASAPI expects on every architecture.
func acInitializeFixed(ac *wca.IAudioClient, shareMode, streamFlags uint32, buf, per wca.REFERENCE_TIME, format *wca.WAVEFORMATEX, guid *ole.GUID) error {
	var hr uintptr
	if unsafe.Sizeof(uintptr(0)) == 8 {
		hr, _, _ = syscall.SyscallN(ac.VTable().Initialize,
			uintptr(unsafe.Pointer(ac)),
			uintptr(shareMode),
			uintptr(streamFlags),
			uintptr(buf),
			uintptr(per),
			uintptr(unsafe.Pointer(format)),
			uintptr(unsafe.Pointer(guid)),
		)
	} else {
		hr, _, _ = syscall.SyscallN(ac.VTable().Initialize,
			uintptr(unsafe.Pointer(ac)),
			uintptr(shareMode),
			uintptr(streamFlags),
			uintptr(uint64(buf)&0xFFFFFFFF),
			uintptr(uint64(buf)>>32),
			uintptr(uint64(per)&0xFFFFFFFF),
			uintptr(uint64(per)>>32),
			uintptr(unsafe.Pointer(format)),
			uintptr(unsafe.Pointer(guid)),
		)
	}
	if hr != 0 {
		return ole.NewError(hr)
	}
	return nil
}

// wasapiCapture captures the default render endpoint in loopback mode and keeps
// the latest smoothed band levels available to Levels(). All COM work happens on
// a single OS-thread-locked goroutine (COM is thread-affine).
type wasapiCapture struct {
	mu     sync.Mutex
	levels []float64

	done chan struct{}
	once sync.Once

	// Diagnostics (used by DebugStats): which setup stage we reached and how many
	// audio frames we've pulled. atomic.Int64 is 64-bit-aligned even on 386.
	stage  atomic.Value
	frames atomic.Int64
}

// setStage records the furthest setup/run stage reached, for diagnostics.
func (c *wasapiCapture) setStage(s string) { c.stage.Store(s) }

// DebugStats reports the last stage reached and total frames captured.
// Used only by the throwaway audiodiag command.
func (c *wasapiCapture) DebugStats() (string, int64) {
	s, _ := c.stage.Load().(string)
	return s, c.frames.Load()
}

func newPlatformAnalyzer() Analyzer {
	c := &wasapiCapture{
		levels: make([]float64, internalBands),
		done:   make(chan struct{}),
	}
	go c.run()
	return c
}

// Levels returns the latest band levels resampled to n.
func (c *wasapiCapture) Levels(n int) []float64 {
	c.mu.Lock()
	src := make([]float64, len(c.levels))
	copy(src, c.levels)
	c.mu.Unlock()
	return resample(src, n)
}

// Close stops the capture goroutine.
func (c *wasapiCapture) Close() error {
	c.once.Do(func() { close(c.done) })
	return nil
}

// run owns the entire COM lifecycle. Any setup error simply ends the goroutine;
// Levels then keeps returning zeros and the UI falls back to the static visual.
func (c *wasapiCapture) run() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	c.setStage("coinit")
	if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
		c.setStage("coinit-failed: " + err.Error())
		return
	}
	defer ole.CoUninitialize()

	c.setStage("cocreate")
	var de *wca.IMMDeviceEnumerator
	if err := wca.CoCreateInstance(wca.CLSID_MMDeviceEnumerator, 0, wca.CLSCTX_ALL, wca.IID_IMMDeviceEnumerator, &de); err != nil {
		c.setStage("cocreate-failed: " + err.Error())
		return
	}
	defer de.Release()

	// ERender + loopback = capture whatever is being played out.
	c.setStage("endpoint")
	var mmd *wca.IMMDevice
	if err := de.GetDefaultAudioEndpoint(wca.ERender, wca.EConsole, &mmd); err != nil {
		c.setStage("endpoint-failed: " + err.Error())
		return
	}
	defer mmd.Release()

	c.setStage("activate")
	var ac *wca.IAudioClient
	if err := mmd.Activate(wca.IID_IAudioClient, wca.CLSCTX_ALL, nil, &ac); err != nil {
		c.setStage("activate-failed: " + err.Error())
		return
	}
	defer ac.Release()

	c.setStage("mixformat")
	var wfx *wca.WAVEFORMATEX
	if err := ac.GetMixFormat(&wfx); err != nil {
		c.setStage("mixformat-failed: " + err.Error())
		return
	}
	defer ole.CoTaskMemFree(uintptr(unsafe.Pointer(wfx)))

	channels := int(wfx.NChannels)
	bits := int(wfx.WBitsPerSample)
	blockAlign := int(wfx.NBlockAlign)
	if channels < 1 {
		channels = 2
	}
	if bits < 8 {
		bits = 32
	}
	if blockAlign < 1 {
		blockAlign = channels * bits / 8
	}

	c.setStage(fmt.Sprintf("initialize ch=%d bits=%d rate=%d", channels, bits, wfx.NSamplesPerSec))
	// Buffer duration 0 → WASAPI picks the device default. Passing an explicit
	// REFERENCE_TIME here returns E_INVALIDARG on this 386 toolchain (the int64
	// arg marshals differently), and 0 is valid for shared-mode loopback anyway.
	if err := acInitializeFixed(
		ac,
		wca.AUDCLNT_SHAREMODE_SHARED,
		wca.AUDCLNT_STREAMFLAGS_LOOPBACK,
		0, 0, wfx, nil,
	); err != nil {
		c.setStage(fmt.Sprintf("initialize-failed (ch=%d bits=%d rate=%d tag=%#x cbSize=%d): %v",
			channels, bits, wfx.NSamplesPerSec, wfx.WFormatTag, wfx.CbSize, err))
		return
	}

	c.setStage("getservice")
	var acc *wca.IAudioCaptureClient
	if err := ac.GetService(wca.IID_IAudioCaptureClient, &acc); err != nil {
		c.setStage("getservice-failed: " + err.Error())
		return
	}
	defer acc.Release()

	c.setStage("start")
	if err := ac.Start(); err != nil {
		c.setStage("start-failed: " + err.Error())
		return
	}
	defer ac.Stop()
	c.setStage("capturing")

	mono := make([]float64, 0, windowSize*2)
	prev := make([]float64, internalBands)

	for {
		select {
		case <-c.done:
			return
		default:
		}

		// Drain all currently-available packets into the mono buffer.
		for {
			var packetLen uint32
			if err := acc.GetNextPacketSize(&packetLen); err != nil || packetLen == 0 {
				break
			}

			var (
				data   *byte
				frames uint32
				flags  uint32
				devPos uint64
				qpcPos uint64
			)
			if err := acc.GetBuffer(&data, &frames, &flags, &devPos, &qpcPos); err != nil {
				break
			}

			if frames > 0 {
				c.frames.Add(int64(frames))
				silent := flags&wca.AUDCLNT_BUFFERFLAGS_SILENT != 0
				if silent || data == nil {
					for i := 0; i < int(frames); i++ {
						mono = append(mono, 0)
					}
				} else {
					raw := unsafe.Slice(data, int(frames)*blockAlign)
					mono = appendMono(mono, raw, channels, bits)
				}
			}
			acc.ReleaseBuffer(frames)
		}

		// Once we have a full window, analyze the most recent samples.
		if len(mono) >= windowSize {
			seg := make([]float64, windowSize)
			copy(seg, mono[len(mono)-windowSize:])
			mono = mono[len(mono)-windowSize:] // keep only the tail; bound memory

			applyHann(seg)
			mags := magnitudes(seg)
			bnd := bands(mags, internalBands)
			prev = smooth(prev, bnd, 0.6, 0.25)

			c.mu.Lock()
			copy(c.levels, prev)
			c.mu.Unlock()
		}

		time.Sleep(16 * time.Millisecond)
	}
}

// appendMono decodes interleaved PCM frames to mono float samples and appends
// them to dst. Supports 32-bit float and 16-bit int mix formats.
func appendMono(dst []float64, b []byte, channels, bits int) []float64 {
	bytesPerSample := bits / 8
	frameSize := bytesPerSample * channels
	if frameSize == 0 {
		return dst
	}
	for off := 0; off+frameSize <= len(b); off += frameSize {
		var sum float64
		for ch := 0; ch < channels; ch++ {
			so := off + ch*bytesPerSample
			switch bits {
			case 32:
				sum += float64(math.Float32frombits(binary.LittleEndian.Uint32(b[so : so+4])))
			case 16:
				sum += float64(int16(binary.LittleEndian.Uint16(b[so:so+2]))) / 32768.0
			}
		}
		dst = append(dst, sum/float64(channels))
	}
	return dst
}
