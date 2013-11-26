/*
A binding of SDL and SDL_image.

The binding works in pretty much the same way as it does in C, although
some of the functions have been altered to give them an object-oriented
flavor (eg. Rather than sdl.Flip(surface) it's surface.Flip() )
*/
package sdl

// #cgo CFLAGS: -D_REENTRANT
// #cgo LDFLAGS: -lSDL
// #cgo windows LDFLAGS: -lwinmm -lgdi32 -ldxguid
//
// struct private_hwdata{};
// struct SDL_BlitMap{};
// #define map _map
//
// #include <SDL/SDL.h>
// static void SetError(const char* description){SDL_SetError("%s",description);}
import "C"

import (
	"os"
	"runtime"
	"sync"
	"time"
	"unsafe"
)

// Mutex for serialization of access to certain SDL functions.
//
// There is no need to use this in application code, the mutex is a public variable
// just because it needs to be accessible from other parts of Go-SDL (such as package "sdl/ttf").
//
// Surface-level functions (such as 'Surface.Blit') are not using this mutex,
// so it is possible to modify multiple surfaces concurrently.
// There is no dependency between 'Surface.Lock' and the global mutex.
var GlobalMutex sync.Mutex

type Surface struct {
	cSurface *C.SDL_Surface
	mutex    sync.RWMutex

	Flags  uint32
	Format *PixelFormat
	W      int32
	H      int32
	Pitch  uint16
	Pixels unsafe.Pointer
	Offset int32

	gcPixels interface{} // Prevents garbage collection of pixels passed to func CreateRGBSurfaceFrom
}

func wrap(cSurface *C.SDL_Surface) *Surface {
	var s *Surface

	if cSurface != nil {
		var surface Surface
		surface.setCSurface(unsafe.Pointer(cSurface))
		s = &surface
	} else {
		s = nil
	}

	return s
}

func (s *Surface) setCSurface(cSurface unsafe.Pointer) {
	s.cSurface = (*C.SDL_Surface)(cSurface)
	s.reload()
}

// Pull data from C.SDL_Surface.
// Make sure to use this when the C surface might have been changed.
func (s *Surface) reload() {
	s.Flags = uint32(s.cSurface.flags)
	s.Format = (*PixelFormat)(unsafe.Pointer(s.cSurface.format))
	s.W = int32(s.cSurface.w)
	s.H = int32(s.cSurface.h)
	s.Pitch = uint16(s.cSurface.pitch)
	s.Pixels = s.cSurface.pixels
	s.Offset = int32(s.cSurface.offset)
}

func (s *Surface) destroy() {
	s.cSurface = nil
	s.Format = nil
	s.Pixels = nil
	s.gcPixels = nil
}

// The version of Go-SDL bindings.
// The version descriptor changes into a new unique string
// after a semantically incompatible Go-SDL update.
//
// The returned value can be checked by users of this package
// to make sure they are using a version with the expected semantics.
//
// If Go adds some kind of support for package versioning, this function will go away.
func GoSdlVersion() string {
	return "⚛SDL bindings 1.0"
}

// A Threadbound is a queue of functions bound to execute on one and only
// one OS thread.
type Threadbound chan func()

var (
	startPoll sync.Once
	thread    Threadbound
)

func NewThreadbound() Threadbound {
	return make(chan func())
}

// Drain blocks the calling goroutine until tb is closed, and any
// functions that are queued will be executed on that goroutine's system
// thread.
func (tb Threadbound) Drain() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	for f := range tb {
		f()
	}
}

// Run adds a function to the queue and blocks until it is executed.
// If Run is called on an uninitialised Threadbound, f will be called
// immediately in the calling goroutine.
func (tb Threadbound) Run(f func()) {
	if tb != nil {
		done := make(chan bool, 1)
		tb <- func() {
			f()
			done <- true
		}
		<-done
	} else {
		f()
	}
}

// Close closes tb and allows any blocked Drain calls to return.
func (tb Threadbound) Close() {
	close(tb)
}

func SetThreadbound(tb Threadbound) {
	thread = tb
}

// Initializes SDL.
func Init(flags uint32) int {
	var status int
	
	GlobalMutex.Lock()
	thread.Run(func() {
		status = int(C.SDL_Init(C.Uint32(flags)))
	})
	if (status != 0) && (runtime.GOOS == "darwin") && (flags&INIT_VIDEO != 0) {
		if os.Getenv("SDL_VIDEODRIVER") == "" {
			os.Setenv("SDL_VIDEODRIVER", "x11")
			thread.Run(func() {
				status = int(C.SDL_Init(C.Uint32(flags)))
			})
			if status != 0 {
				os.Setenv("SDL_VIDEODRIVER", "")
			}
		}
	}
	GlobalMutex.Unlock()
	startPoll.Do(func() {
		go pollEvents()
	})
	return status
}

// Shuts down SDL
func Quit() {
	GlobalMutex.Lock()
	defer GlobalMutex.Unlock()
	if currentVideoSurface != nil {
		currentVideoSurface.destroy()
		currentVideoSurface = nil
	}
	C.SDL_Quit()
}

// Initializes subsystems.
func InitSubSystem(flags uint32) int {
	var status int
	
	GlobalMutex.Lock()
	defer GlobalMutex.Unlock()
	thread.Run(func() {
		status = int(C.SDL_InitSubSystem(C.Uint32(flags)))
	})
	if (status != 0) && (runtime.GOOS == "darwin") && (flags&INIT_VIDEO != 0) {
		if os.Getenv("SDL_VIDEODRIVER") == "" {
			os.Setenv("SDL_VIDEODRIVER", "x11")
			thread.Run(func() {
				status = int(C.SDL_InitSubSystem(C.Uint32(flags)))
			})
			if status != 0 {
				os.Setenv("SDL_VIDEODRIVER", "")
			}
		}
	}
	return status
}

// Shuts down a subsystem.
func QuitSubSystem(flags uint32) {
	GlobalMutex.Lock()
	C.SDL_QuitSubSystem(C.Uint32(flags))
	GlobalMutex.Unlock()
}

// Checks which subsystems are initialized.
func WasInit(flags uint32) int {
	GlobalMutex.Lock()
	status := int(C.SDL_WasInit(C.Uint32(flags)))
	GlobalMutex.Unlock()
	return status
}

//
// Error handling
//

// Gets SDL error string
func GetError() string {
	GlobalMutex.Lock()
	s := C.GoString(C.SDL_GetError())
	GlobalMutex.Unlock()
	return s
}

// Set a string describing an error to be submitted to the SDL Error system.
func SetError(description string) {
	GlobalMutex.Lock()
	defer GlobalMutex.Unlock()
	cdescription := C.CString(description)
	C.SetError(cdescription)
	C.free(unsafe.Pointer(cdescription))
}

// Clear the current SDL error
func ClearError() {
	GlobalMutex.Lock()
	C.SDL_ClearError()
	GlobalMutex.Unlock()
}

//
// Time
//

// Gets the number of milliseconds since the SDL library initialization.
func GetTicks() uint32 {
	GlobalMutex.Lock()
	t := uint32(C.SDL_GetTicks())
	GlobalMutex.Unlock()
	return t
}

// Waits a specified number of milliseconds before returning.
func Delay(ms uint32) {
	time.Sleep(time.Duration(ms) * time.Millisecond)
}
