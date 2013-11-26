package main

import (
	"fmt"
	"sdl"
	"log"
	"math"
	"time"
)

type Point struct {
	x int
	y int
}

func (a Point) add(b Point) Point { return Point{a.x + b.x, a.y + b.y} }

func (a Point) sub(b Point) Point { return Point{a.x - b.x, a.y - b.y} }

func (a Point) length() float64 { return math.Sqrt(float64(a.x*a.x + a.y*a.y)) }

func (a Point) mul(b float64) Point {
	return Point{int(float64(a.x) * b), int(float64(a.y) * b)}
}

func worm(in <-chan Point, out chan<- Point, draw chan<- Point) {
	t := Point{0, 0}
	for {
		p := (<-in).sub(t)
		if p.length() > 24 {
			t = t.add(p.mul(0.1))
		}
		draw <- t
		out <- t
	}
}

func main() {
	log.SetFlags(0)
	var joy *sdl.Joystick
	tb := sdl.NewThreadbound()
	sdl.SetThreadbound(tb)
	go tb.Drain()
	if sdl.Init(sdl.INIT_EVERYTHING) != 0 {
		log.Fatal(sdl.GetError())
	}
	println("Init ok")
	if sdl.NumJoysticks() > 0 {
		joy = sdl.JoystickOpen(0)
		if joy != nil {
			println("Opened Joystick 0")
			println("Name: ", sdl.JoystickName(0))
			println("Number of Axes: ", joy.NumAxes())
			println("Number of Buttons: ", joy.NumButtons())
			println("Number of Balls: ", joy.NumBalls())
		} else {
			println("Couldn't open Joystick!")
		}
	}

	screen := sdl.SetVideoMode(640, 480, 32, sdl.RESIZABLE)
	if screen == nil {
		log.Fatal(sdl.GetError())
	}
	println("SetVideoMode ok")

	vidinfo := sdl.GetVideoInfo()
	println("HW_available = ", vidinfo.HW_available)
	println("WM_available = ", vidinfo.WM_available)
	println("Video_mem = ", vidinfo.Video_mem, "kb")

	sdl.EnableUNICODE(1)
	println("EnableUNICODE ok")
	sdl.WM_SetCaption("sdltest", "")
	println("SetCaption ok")

	W := uint16(64)
	H := uint16(64)
	image := sdl.CreateRGBSurface(0, int(W), int(H), 32, 0xff000000, 0x00ff0000, 0x0000ff00, 0x000000ff)
	image.FillRect(&sdl.Rect{0, 0, W, H}, 0xff00ff33)
	image.FillRect(&sdl.Rect{8, 8, W-16, H-16}, 0xff00ff55)
	image.FillRect(&sdl.Rect{16, 16, W-32, H-32}, 0xff00ff77)
	println("created surface")
	sdl.WM_SetIcon(image, nil)
	println("WM_SetIcon ok")

	running := true

	if sdl.GetKeyName(270) != "[+]" {
		log.Fatal("GetKeyName broken")
	}
	println("GetKeyName ok")

	worm_in := make(chan Point)
	draw := make(chan Point, 64)

	in := worm_in
	out := make(chan Point)
	go worm(in, out, draw)

	tick := time.Tick(time.Second / 60) // 60 Hz

	// Note: The following code is highly ineffective.
	//       It is eating too much CPU. If you intend to use sdl,
	//       you should do better than this.

	for running {
		select {
		case <-tick:
			screen.FillRect(nil, 0x00ffff)
		loop: for {
				select {
				case p := <-draw:
					screen.Blit(&sdl.Rect{int16(p.x), int16(p.y), 0, 0}, image, nil)
				case <-out:
				default:
					break loop
				}
			}
			var p Point
			p.x, p.y, _ = sdl.GetMouseState()
			worm_in <- p
			screen.Flip()
		case _event := <-sdl.Events:
			switch e := _event.(type) {
			default:
				println("unknown event")
			case sdl.ActiveEvent:
				println("window made active")
			case sdl.QuitEvent:
				running = false
			case sdl.KeyboardEvent:
				println("")
				println(e.Keysym.Sym, ": ", sdl.GetKeyName(sdl.Key(e.Keysym.Sym)))

				if e.Keysym.Sym == sdl.K_ESCAPE {
					running = false
				}
				fmt.Printf("%04x ", e.Type)
				for i := 0; i < len(e.Pad0); i++ {
					fmt.Printf("%02x ", e.Pad0[i])
				}
				println()
				fmt.Printf("Type: %02x Which: %02x State: %02x Pad: %02x\n", e.Type, e.Which, e.State, e.Pad0[0])
				fmt.Printf("Scancode: %02x Sym: %08x Mod: %04x Unicode: %04x\n", e.Keysym.Scancode, e.Keysym.Sym, e.Keysym.Mod, e.Keysym.Unicode)
			case sdl.MouseMotionEvent:
			case sdl.MouseButtonEvent:
				if e.Type == sdl.MOUSEBUTTONDOWN {
					println("Click:", e.X, e.Y)
					in = out
					out = make(chan Point)
					go worm(in, out, draw)
				}
			case sdl.JoyAxisEvent:
				println("Joystick Axis Event ->", "Type", e.Type, "Axis:", e.Axis, " Value:", e.Value, "Which:", e.Which)
			case sdl.JoyButtonEvent:
				println("Joystick Button Event ->", e.Button)
				println("State of button", e.Button, "->", joy.GetButton(int(e.Button)))
			case sdl.ResizeEvent:
				println("resize screen ", e.W, e.H)
				screen = sdl.SetVideoMode(int(e.W), int(e.H), 32, sdl.RESIZABLE)
				if screen == nil {
					log.Fatal(sdl.GetError())
				}
				println("resize ok")
			}
		}
	}
	if sdl.JoystickOpened(0) > 0 {
		joy.Close()
	}
	tb.Close()
	image.Free()
	sdl.Quit()
}
