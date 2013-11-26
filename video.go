package sdl

// #cgo CFLAGS: -D_REENTRANT
// #cgo LDFLAGS: -lSDL
// #cgo windows LDFLAGS: -lwinmm -lgdi32 -ldxguid
//
// #include <SDL/SDL.h>
import "C"
import (
	"reflect"
	"unsafe"
)

var currentVideoSurface *Surface = nil

// Sets up a video mode with the specified width, height, bits-per-pixel and
// returns a corresponding surface.  You don't need to call the Free method
// of the returned surface, as it will be done automatically by sdl.Quit.
func SetVideoMode(w int, h int, bpp int, flags uint32) *Surface {
	var screen *Surface
	thread.Run(func() {
		screen = setVideoMode(w, h, bpp, flags)
	})
	return screen
}

func setVideoMode(w int, h int, bpp int, flags uint32) *Surface {
	screen := C.SDL_SetVideoMode(C.int(w), C.int(h), C.int(bpp), C.Uint32(flags))
	currentVideoSurface = wrap(screen)
	return currentVideoSurface
}

// Returns a pointer to the current display surface.
func GetVideoSurface() *Surface {
	GlobalMutex.Lock()
	surface := currentVideoSurface
	GlobalMutex.Unlock()
	return surface
}

// Checks to see if a particular video mode is supported.  Returns 0 if not
// supported, or the bits-per-pixel of the closest available mode.
func VideoModeOK(width int, height int, bpp int, flags uint32) int {
	GlobalMutex.Lock()
	status := int(C.SDL_VideoModeOK(C.int(width), C.int(height), C.int(bpp), C.Uint32(flags)))
	GlobalMutex.Unlock()
	return status
}

// Returns the list of available screen dimensions for the given format.
//
// NOTE: The result of this function uses a different encoding than the underlying C function.
// It returns an empty array if no modes are available,
// and nil if any dimension is okay for the given format.
func ListModes(format *PixelFormat, flags uint32) []Rect {
	modes := C.SDL_ListModes((*C.SDL_PixelFormat)(unsafe.Pointer(format)), C.Uint32(flags))

	// No modes available
	if modes == nil {
		return make([]Rect, 0)
	}

	// (modes == -1) --> Any dimension is ok
	if uintptr(unsafe.Pointer(modes))+1 == uintptr(0) {
		return nil
	}

	count := 0
	ptr := *modes //first element in the list
	for ptr != nil {
		count++
		ptr = *(**C.SDL_Rect)(unsafe.Pointer(uintptr(unsafe.Pointer(modes)) + uintptr(count*int(unsafe.Sizeof(ptr)))))
	}

	ret := make([]Rect, count)
	for i := 0; i < count; i++ {
		ptr := (**C.SDL_Rect)(unsafe.Pointer(uintptr(unsafe.Pointer(modes)) + uintptr(i*int(unsafe.Sizeof(*modes)))))
		var r *C.SDL_Rect = *ptr
		ret[i].X = int16(r.x)
		ret[i].Y = int16(r.y)
		ret[i].W = uint16(r.w)
		ret[i].H = uint16(r.h)
	}

	return ret
}

type VideoInfo struct {
	HW_available bool         "Flag: Can you create hardware surfaces?"
	WM_available bool         "Flag: Can you talk to a window manager?"
	Blit_hw      bool         "Flag: Accelerated blits HW --> HW"
	Blit_hw_CC   bool         "Flag: Accelerated blits with Colorkey"
	Blit_hw_A    bool         "Flag: Accelerated blits with Alpha"
	Blit_sw      bool         "Flag: Accelerated blits SW --> HW"
	Blit_sw_CC   bool         "Flag: Accelerated blits with Colorkey"
	Blit_sw_A    bool         "Flag: Accelerated blits with Alpha"
	Blit_fill    bool         "Flag: Accelerated color fill"
	Video_mem    uint32       "The total amount of video memory (in K)"
	Vfmt         *PixelFormat "Value: The format of the video surface"
	Current_w    int32        "Value: The current video mode width"
	Current_h    int32        "Value: The current video mode height"
}

func GetVideoInfo() *VideoInfo {
	//GlobalMutex.Lock()
	vinfo := (*internalVideoInfo)(unsafe.Pointer(C.SDL_GetVideoInfo()))
	//GlobalMutex.Unlock()

	flags := vinfo.Flags

	return &VideoInfo{
		HW_available: flags&(1<<0) != 0,
		WM_available: flags&(1<<1) != 0,
		Blit_hw:      flags&(1<<9) != 0,
		Blit_hw_CC:   flags&(1<<10) != 0,
		Blit_hw_A:    flags&(1<<11) != 0,
		Blit_sw:      flags&(1<<12) != 0,
		Blit_sw_CC:   flags&(1<<13) != 0,
		Blit_sw_A:    flags&(1<<14) != 0,
		Blit_fill:    flags&(1<<15) != 0,
		Video_mem:    vinfo.Video_mem,
		Vfmt:         vinfo.Vfmt,
		Current_w:    vinfo.Current_w,
		Current_h:    vinfo.Current_h,
	}
}

// Makes sure the given area is updated on the given screen.  If x, y, w, and
// h are all 0, the whole screen will be updated.
func (screen *Surface) UpdateRect(x int32, y int32, w uint32, h uint32) {
	//GlobalMutex.Lock()
	screen.mutex.Lock()

	C.SDL_UpdateRect(screen.cSurface, C.Sint32(x), C.Sint32(y), C.Uint32(w), C.Uint32(h))

	screen.mutex.Unlock()
	//GlobalMutex.Unlock()
}

func (screen *Surface) UpdateRects(rects []Rect) {
	if len(rects) > 0 {
		//GlobalMutex.Lock()
		screen.mutex.Lock()

		C.SDL_UpdateRects(screen.cSurface, C.int(len(rects)), (*C.SDL_Rect)(unsafe.Pointer(&rects[0])))

		screen.mutex.Unlock()
		//GlobalMutex.Unlock()
	}
}

// Gets the window title and icon name.
func WM_GetCaption() (title, icon string) {
	//GlobalMutex.Lock()

	// SDL seems to free these strings.  TODO: Check to see if that's the case
	var ctitle, cicon *C.char
	C.SDL_WM_GetCaption(&ctitle, &cicon)
	title = C.GoString(ctitle)
	icon = C.GoString(cicon)

	//GlobalMutex.Unlock()

	return
}

// Sets the window title and icon name.
func WM_SetCaption(title, icon string) {
	ctitle := C.CString(title)
	cicon := C.CString(icon)

	thread.Run(func() {
		C.SDL_WM_SetCaption(ctitle, cicon)
	})

	C.free(unsafe.Pointer(ctitle))
	C.free(unsafe.Pointer(cicon))
}

// Sets the icon for the display window.
func WM_SetIcon(icon *Surface, mask *uint8) {
	thread.Run(func() {
		C.SDL_WM_SetIcon(icon.cSurface, (*C.Uint8)(mask))
	})
}

// Minimizes the window
func WM_IconifyWindow() int {
	var status int
	thread.Run(func() {
		status = int(C.SDL_WM_IconifyWindow())
	})
	return status
}

// Toggles fullscreen mode
func WM_ToggleFullScreen(surface *Surface) int {
	var status int
	thread.Run(func() {
		status = int(C.SDL_WM_ToggleFullScreen(surface.cSurface))
	})
	return status
}

// Swaps OpenGL framebuffers/Update Display.
func GL_SwapBuffers() {
	thread.Run(func() {
		C.SDL_GL_SwapBuffers()
	})
}

func GL_SetAttribute(attr int, value int) int {
	var status int
	thread.Run(func() {
		status = int(C.SDL_GL_SetAttribute(C.SDL_GLattr(attr), C.int(value)))
	})
	return status
}

// Swaps screen buffers.
func (screen *Surface) Flip() int {
	//GlobalMutex.Lock()
	screen.mutex.Lock()

	status := int(C.SDL_Flip(screen.cSurface))

	screen.mutex.Unlock()
	//GlobalMutex.Unlock()

	return status
}

// Frees (deletes) a Surface
func (screen *Surface) Free() {
	//GlobalMutex.Lock()
	screen.mutex.Lock()

	C.SDL_FreeSurface(screen.cSurface)

	screen.destroy()
	if screen == currentVideoSurface {
		currentVideoSurface = nil
	}

	screen.mutex.Unlock()
	//GlobalMutex.Unlock()
}

// Locks a surface for direct access.
func (screen *Surface) Lock() int {
	screen.mutex.Lock()
	status := int(C.SDL_LockSurface(screen.cSurface))
	screen.mutex.Unlock()
	return status
}

// Unlocks a previously locked surface.
func (screen *Surface) Unlock() {
	screen.mutex.Lock()
	C.SDL_UnlockSurface(screen.cSurface)
	screen.mutex.Unlock()
}

// Performs a fast blit from the source surface to the destination surface.
// This is the same as func BlitSurface, but the order of arguments is reversed.
func (dst *Surface) Blit(dstrect *Rect, src *Surface, srcrect *Rect) int {
	//GlobalMutex.Lock()
	global := true
	if (src != currentVideoSurface) && (dst != currentVideoSurface) {
		//GlobalMutex.Unlock()
		global = false
	}

	// At this point: GlobalMutex is locked only if at least one of 'src' or 'dst'
	//                was identical to 'currentVideoSurface'

	var ret C.int
	{
		src.mutex.RLock()
		dst.mutex.Lock()

		ret = C.SDL_UpperBlit(
			src.cSurface,
			(*C.SDL_Rect)(unsafe.Pointer(srcrect)),
			dst.cSurface,
			(*C.SDL_Rect)(unsafe.Pointer(dstrect)))

		dst.mutex.Unlock()
		src.mutex.RUnlock()
	}

	if global {
		//GlobalMutex.Unlock()
	}

	return int(ret)
}

// Performs a fast blit from the source surface to the destination surface.
func BlitSurface(src *Surface, srcrect *Rect, dst *Surface, dstrect *Rect) int {
	return dst.Blit(dstrect, src, srcrect)
}

// This function performs a fast fill of the given rectangle with some color.
func (dst *Surface) FillRect(dstrect *Rect, color uint32) int {
	dst.mutex.Lock()

	var ret = C.SDL_FillRect(
		dst.cSurface,
		(*C.SDL_Rect)(unsafe.Pointer(dstrect)),
		C.Uint32(color))

	dst.mutex.Unlock()

	return int(ret)
}

// Adjusts the alpha properties of a Surface.
func (s *Surface) SetAlpha(flags uint32, alpha uint8) int {
	s.mutex.Lock()
	status := int(C.SDL_SetAlpha(s.cSurface, C.Uint32(flags), C.Uint8(alpha)))
	s.mutex.Unlock()
	return status
}

// Sets the color key (transparent pixel)  in  a  blittable  surface  and
// enables or disables RLE blit acceleration.
func (s *Surface) SetColorKey(flags uint32, ColorKey uint32) int {
	s.mutex.Lock()
	status := int(C.SDL_SetColorKey(s.cSurface, C.Uint32(flags), C.Uint32(ColorKey)))
	s.mutex.Unlock()
	return status
}

// Gets the clipping rectangle for a surface.
func (s *Surface) GetClipRect(r *Rect) {
	s.mutex.RLock()
	C.SDL_GetClipRect(s.cSurface, (*C.SDL_Rect)(unsafe.Pointer(r)))
	s.mutex.RUnlock()
}

// Sets the clipping rectangle for a surface.
func (s *Surface) SetClipRect(r *Rect) {
	s.mutex.Lock()
	C.SDL_SetClipRect(s.cSurface, (*C.SDL_Rect)(unsafe.Pointer(r)))
	s.mutex.Unlock()
}

// Map a RGBA color value to a pixel format.
func MapRGBA(format *PixelFormat, r, g, b, a uint8) uint32 {
	return (uint32)(C.SDL_MapRGBA((*C.SDL_PixelFormat)(unsafe.Pointer(format)), (C.Uint8)(r), (C.Uint8)(g), (C.Uint8)(b), (C.Uint8)(a)))
}

// Gets RGBA values from a pixel in the specified pixel format.
func GetRGBA(color uint32, format *PixelFormat, r, g, b, a *uint8) {
	C.SDL_GetRGBA(C.Uint32(color), (*C.SDL_PixelFormat)(unsafe.Pointer(format)), (*C.Uint8)(r), (*C.Uint8)(g), (*C.Uint8)(b), (*C.Uint8)(a))
}

// Creates an empty Surface.
func CreateRGBSurface(flags uint32, width int, height int, bpp int, Rmask uint32, Gmask uint32, Bmask uint32, Amask uint32) *Surface {
	var p *C.SDL_Surface
	//GlobalMutex.Lock()

	thread.Run(func() {
		p = C.SDL_CreateRGBSurface(C.Uint32(flags), C.int(width), C.int(height), C.int(bpp),
			C.Uint32(Rmask), C.Uint32(Gmask), C.Uint32(Bmask), C.Uint32(Amask))
	})
	//GlobalMutex.Unlock()
	return wrap(p)
}

// Creates a Surface from existing pixel data. It expects pixels to be a slice, pointer or unsafe.Pointer.
func CreateRGBSurfaceFrom(pixels interface{}, width, height, bpp, pitch int, Rmask, Gmask, Bmask, Amask uint32) *Surface {
	var ptr unsafe.Pointer
	switch v := reflect.ValueOf(pixels); v.Kind() {
	case reflect.Ptr, reflect.UnsafePointer, reflect.Slice:
		ptr = unsafe.Pointer(v.Pointer())
	default:
		panic("Don't know how to handle type: " + v.Kind().String())
	}

	//GlobalMutex.Lock()
	p := C.SDL_CreateRGBSurfaceFrom(ptr, C.int(width), C.int(height), C.int(bpp), C.int(pitch),
		C.Uint32(Rmask), C.Uint32(Gmask), C.Uint32(Bmask), C.Uint32(Amask))
	//GlobalMutex.Unlock()

	s := wrap(p)
	s.gcPixels = pixels
	return s
}

// Converts a surface to the display format
func (s *Surface) DisplayFormat() *Surface {
	s.mutex.RLock()
	p := C.SDL_DisplayFormat(s.cSurface)
	s.mutex.RUnlock()
	return wrap(p)
}

// Converts a surface to the display format with alpha
func (s *Surface) DisplayFormatAlpha() *Surface {
	s.mutex.RLock()
	p := C.SDL_DisplayFormatAlpha(s.cSurface)
	s.mutex.RUnlock()
	return wrap(p)
}
