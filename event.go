package sdl

// #cgo CFLAGS: -D_REENTRANT
// #cgo LDFLAGS: -lSDL
// #cgo windows LDFLAGS: -lwinmm -lgdi32 -ldxguid
//
// #include <SDL/SDL.h>
import "C"

import (
	"time"
	"unsafe"
)

var events chan interface{} = make(chan interface{})

// This channel delivers SDL events. Each object received from this channel
// has one of the following types: sdl.QuitEvent, sdl.KeyboardEvent,
// sdl.MouseButtonEvent, sdl.MouseMotionEvent, sdl.ActiveEvent,
// sdl.ResizeEvent, sdl.JoyAxisEvent, sdl.JoyButtonEvent, sdl.JoyHatEvent,
// sdl.JoyBallEvent
var Events <-chan interface{} = events

// Polling interval, in milliseconds
const poll_interval_ms = 10

// Polls for currently pending events
func (event *Event) poll() bool {
	GlobalMutex.Lock()
	ret := C.SDL_PollEvent((*C.SDL_Event)(unsafe.Pointer(event)))
	if ret != 0 {
		if (event.Type == VIDEORESIZE) && (currentVideoSurface != nil) {
			currentVideoSurface.reload()
		}
	}
	GlobalMutex.Unlock()
	return ret != 0
}

// pollThread does the polling of events in the thread associated with
// the global threadlock.
func (event *Event) pollThread() bool {
	var status bool
	thread.Run(func() {
		status = event.poll()
	})
	return status
}

// Polls SDL events in periodic intervals.
// This function does not return.
func pollEvents() {
	// It is more efficient to create the event-object here once,
	// rather than multiple times within the loop
	event := &Event{}
	for {
		for event.pollThread() {
			switch event.Type {
			case QUIT:
				events <- *(*QuitEvent)(unsafe.Pointer(event))
			case KEYDOWN, KEYUP:
				events <- *(*KeyboardEvent)(unsafe.Pointer(event))
			case MOUSEBUTTONDOWN, MOUSEBUTTONUP:
				events <- *(*MouseButtonEvent)(unsafe.Pointer(event))
			case MOUSEMOTION:
				events <- *(*MouseMotionEvent)(unsafe.Pointer(event))
			case JOYAXISMOTION:
				events <- *(*JoyAxisEvent)(unsafe.Pointer(event))
			case JOYBUTTONDOWN, JOYBUTTONUP:
				events <- *(*JoyButtonEvent)(unsafe.Pointer(event))
			case JOYHATMOTION:
				events <- *(*JoyHatEvent)(unsafe.Pointer(event))
			case JOYBALLMOTION:
				events <- *(*JoyBallEvent)(unsafe.Pointer(event))
			case ACTIVEEVENT:
				events <- *(*ActiveEvent)(unsafe.Pointer(event))
			case VIDEORESIZE:
				events <- *(*ResizeEvent)(unsafe.Pointer(event))
			}
		}
		time.Sleep(poll_interval_ms * 1e6)
	}
}
