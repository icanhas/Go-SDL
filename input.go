package sdl

// #cgo CFLAGS: -D_REENTRANT
// #cgo LDFLAGS: -lSDL
// #cgo windows LDFLAGS: -lwinmm -lgdi32 -ldxguid
//
// #include <SDL/SDL.h>
import "C"
import "unsafe"

// Modifier
type Mod C.int
type Key C.int
type Joystick struct {
	cJoystick *C.SDL_Joystick
}

// Enables UNICODE translation.
func EnableUNICODE(enable int) int {
	return int(C.SDL_EnableUNICODE(C.int(enable)))
}

// Sets keyboard repeat rate.
func EnableKeyRepeat(delay, interval int) int {
	return int(C.SDL_EnableKeyRepeat(C.int(delay), C.int(interval)))
}

// Gets keyboard repeat rate.
func GetKeyRepeat() (int, int) {
	var delay int
	var interval int

	C.SDL_GetKeyRepeat((*C.int)(unsafe.Pointer(&delay)), (*C.int)(unsafe.Pointer(&interval)))
	return delay, interval
}

// Gets a snapshot of the current keyboard state
func GetKeyState() []uint8 {
	var numkeys C.int

	array := C.SDL_GetKeyState(&numkeys)
	ptr := make([]uint8, numkeys)
	*((**C.Uint8)(unsafe.Pointer(&ptr))) = array // TODO
	return ptr
}

// Gets the state of modifier keys
func GetModState() Mod {
	return Mod(C.SDL_GetModState())
}

// Sets the state of modifier keys
func SetModState(modstate Mod) {
	C.SDL_SetModState(C.SDLMod(modstate))
}

// Gets the name of an SDL virtual keysym
func GetKeyName(key Key) string {
	return C.GoString(C.SDL_GetKeyName(C.SDLKey(key)))
}

//
// Mouse
//

// Returns the current mouse coordinates and a bitmask of the current
// button state.
func GetMouseState() (x int, y int, buttons uint32) {
	var xx, yy C.int

	bs := uint32(C.SDL_GetMouseState(&xx, &yy))
	return int(xx), int(yy), uint32(bs)
}

// Returns the mouse coordinates relative to the last time this
// function was called (or relative to event initialisation if this function
// has not been called before), and a bitmask of the current button state.
func GetRelativeMouseState() (x int, y int, buttons uint32) {
	var xx, yy C.int

	bs := uint32(C.SDL_GetRelativeMouseState(&xx, &yy))
	return int(xx), int(yy), uint32(bs)
}

// Toggle whether or not the cursor is shown on the screen.
func ShowCursor(toggle int) int {
	return int(C.SDL_ShowCursor((C.int)(toggle)))
}

//
// Joystick
//

func wrapJoystick(cJoystick *C.SDL_Joystick) *Joystick {
	var j *Joystick
	if cJoystick != nil {
		var joystick Joystick
		joystick.cJoystick = (*C.SDL_Joystick)(unsafe.Pointer(cJoystick))
		j = &joystick
	} else {
		j = nil
	}
	return j
}

// Count the number of joysticks attached to the system
func NumJoysticks() int {
	return int(C.SDL_NumJoysticks())
}

// Get the implementation dependent name of a joystick.
// This can be called before any joysticks are opened.
// If no name can be found, this function returns NULL.
func JoystickName(deviceIndex int) string {
	return C.GoString(C.SDL_JoystickName(C.int(deviceIndex)))
}

// Open a joystick for use The index passed as an argument refers to
// the N'th joystick on the system. This index is the value which will
// identify this joystick in future joystick events.  This function
// returns a joystick identifier, or NULL if an error occurred.
func JoystickOpen(deviceIndex int) *Joystick {
	joystick := C.SDL_JoystickOpen(C.int(deviceIndex))
	return wrapJoystick(joystick)
}

// Returns 1 if the joystick has been opened, or 0 if it has not.
func JoystickOpened(deviceIndex int) int {
	return int(C.SDL_JoystickOpened(C.int(deviceIndex)))
}

// Update the current state of the open joysticks. This is called
// automatically by the event loop if any joystick events are enabled.
func JoystickUpdate() {
	C.SDL_JoystickUpdate()
}

// Enable/disable joystick event polling. If joystick events are
// disabled, you must call SDL_JoystickUpdate() yourself and check the
// state of the joystick when you want joystick information. The state
// can be one of SDL_QUERY, SDL_ENABLE or SDL_IGNORE.
func JoystickEventState(state int) int {
	return int(C.SDL_JoystickEventState(C.int(state)))
}

// Close a joystick previously opened with SDL_JoystickOpen()
func (joystick *Joystick) Close() {
	C.SDL_JoystickClose(joystick.cJoystick)
}

// Get the number of general axis controls on a joystick
func (joystick *Joystick) NumAxes() int {
	return int(C.SDL_JoystickNumAxes(joystick.cJoystick))
}

// Get the device index of an opened joystick.
func (joystick *Joystick) Index() int {
	return int(C.SDL_JoystickIndex(joystick.cJoystick))
}

// Get the number of buttons on a joystick
func (joystick *Joystick) NumButtons() int {
	return int(C.SDL_JoystickNumButtons(joystick.cJoystick))
}

// Get the number of trackballs on a Joystick trackballs have only
// relative motion events associated with them and their state cannot
// be polled.
func (joystick *Joystick) NumBalls() int {
	return int(C.SDL_JoystickNumBalls(joystick.cJoystick))
}

// Get the number of POV hats on a joystick
func (joystick *Joystick) NumHats() int {
	return int(C.SDL_JoystickNumHats(joystick.cJoystick))
}

// Get the current state of a POV hat on a joystick
// The hat indices start at index 0.
func (joystick *Joystick) GetHat(hat int) uint8 {
	return uint8(C.SDL_JoystickGetHat(joystick.cJoystick, C.int(hat)))
}

// Get the current state of a button on a joystick. The button indices
// start at index 0.
func (joystick *Joystick) GetButton(button int) uint8 {
	return uint8(C.SDL_JoystickGetButton(joystick.cJoystick, C.int(button)))
}

// Get the ball axis change since the last poll. The ball indices
// start at index 0. This returns 0, or -1 if you passed it invalid
// parameters.
func (joystick *Joystick) GetBall(ball int, dx, dy *int) int {
	return int(C.SDL_JoystickGetBall(joystick.cJoystick, C.int(ball), (*C.int)(unsafe.Pointer(dx)), (*C.int)(unsafe.Pointer(dy))))
}

// Get the current state of an axis control on a joystick. The axis
// indices start at index 0. The state is a value ranging from -32768
// to 32767.
func (joystick *Joystick) GetAxis(axis int) int16 {
	return int16(C.SDL_JoystickGetAxis(joystick.cJoystick, C.int(axis)))
}
