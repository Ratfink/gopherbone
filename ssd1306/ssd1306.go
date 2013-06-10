/* GopherBone - A collection of packages for working with the BeagleBone in Go
 * Copyright (c) 2013 Clayton G. Hobbs
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to
 * deal in the Software without restriction, including without limitation the
 * rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
 * sell copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
 * FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS
 * IN THE SOFTWARE.
 */

package ssd1306

import (
	"github.com/Ratfink/gopherbone/gpio"
	"github.com/Ratfink/gopherbone/i2c"
	"time"
	"image/color"
	"math"
)

// Constants to allow different serial interfaces to be used in communicating
// with the display.  Currently, only IFACE_I2C is supported.
const (
	IFACE_SPI = 0
	IFACE_I2C = 1
)

// Fundamental commands
const (
	CONTRAST = 0x81 // 2 bytes; follow with 8 contrast bits
	DISP_RAM = 0xa4 // Display what's in RAM
	DISP_ALL = 0xa5 // All white
	INVERSE_OFF = 0xa6
	INVERSE_ON = 0xa7
	DISP_OFF = 0xae
	DISP_ON = 0xaf
)

// Scrolling commands
// These should probably be documented and made easier to use, but for now just
// check the datasheet.
const (
	HSCROLL_RIGHT = 0x26 // 7 bytes
	HSCROLL_LEFT = 0x27 // 7 bytes
	VHSCROLL_RIGHT = 0x29 // 6 bytes
	VHSCROLL_LEFT = 0x2a // 6 bytes
	STOP_SCROLL = 0x2e
	START_SCROLL = 0x2f
	VSCROLL = 0xa3 // 3 bytes
)

// Addressing setting commands
const (
	PAGE_START_LOW = 0x00 // OR with low nibble (4 bits)
	PAGE_START_HIGH = 0x10 // OR with high nibble (4 bits)
	ADDRESS_MODE = 0x20 // 2 bytes; follow with one of the ADDRESS_MODE_* bytes
	ADDRESS_MODE_HORI = 0x00
	ADDRESS_MODE_VERT = 0x01
	ADDRESS_MODE_PAGE = 0x02
	COLUMN_ADDRESS = 0x21 // 3 bytes; follow with start and end addresses
	PAGE_ADDRESS = 0x22 // 3 bytes; follow with start and end pages
	PAGE_START = 0xB0 // OR with page address (3 bits)
)

// Hardware configuration commands
const (
	START_LINE = 0x40 // OR with start line for display (6 bits)
	HORI_NORMAL = 0xa0
	HORI_MIRROR = 0xa1
	MUX_RATIO = 0xa8 // 2 bytes; follow with 6 bit mux ratio (14 < r < 64)
	VERT_NORMAL = 0xc0
	VERT_MIRROR = 0xc8
	VERT_SHIFT = 0xd3 // 2 bytes; follow with 6 bit shift
	COM_CONFIG = 0xda // 2 bytes; follow with COM_CONFIG2 | COM_CONFIG2_*
	COM_CONFIG2 = 0x02
	COM_CONFIG2_ALT = 0x10
	COM_CONFIG2_LR_REMAP = 0x20
)

// Timing and driving scheme commands
const (
	CLOCK_FREQ = 0xd5 // 2 bytes; 2nd byte: low nibble is D, high nibble is F[osc]
	PRECHARGE = 0xd9 // 2 bytes; 2nd byte: low nibble is phase 1, high nibble is phase 2
	VCOMH_DESELECT_LEVEL = 0xdb // 2 bytes; 2nd byte: 0x00, 0x20, or 0x30 (voltages; see datasheet)
	NOP = 0xe3
)

// Charge pump commands
const (
	CHARGE_PUMP = 0x8d // 2 bytes; follow with one of the following
	CHARGE_PUMP_OFF = 0x10
	CHARGE_PUMP_ON = 0x14
)

type SSD1306 struct {
	rst *gpio.GPIO
	iface int
	i2cbus *i2c.Bus
	width int
	height int
	buf []byte
}

func New(rstpin, iface int, addr, bus byte, width, height int) (ssd1306 *SSD1306, err error) {
	ssd1306 = new(SSD1306)

	ssd1306.rst, err = gpio.Export(rstpin)
	if err != nil {
		return
	}

	ssd1306.iface = iface
	if iface == IFACE_I2C {
		ssd1306.i2cbus, err = i2c.NewBus(addr, bus)
		if err != nil {
			return
		}
	}

	ssd1306.width, ssd1306.height = width, height
	ssd1306.buf = make([]byte, width*height/8)

	return
}

func (ssd1306 *SSD1306) Close() {
	ssd1306.WriteData([]byte{0xae})
	ssd1306.rst.Unexport()
}

func (ssd1306 *SSD1306) Setup() (err error) {
	// Reset the display
	err = ssd1306.rst.SetDirection("out")
	if err != nil {
		return
	}
	err = ssd1306.rst.SetValue(0)
	if err != nil {
		return
	}
	time.Sleep(3*time.Millisecond)
	err = ssd1306.rst.SetValue(1)
	if err != nil {
		return
	}

	// Configure the display.  The whole thing is 24 bytes, so send it in one big write.
	ssd1306.WriteCmd([]byte{
		DISP_OFF,
		START_LINE | 0x00,
		ADDRESS_MODE, ADDRESS_MODE_HORI,
		CONTRAST, 0xcf,
		HORI_MIRROR,
		INVERSE_OFF,
		MUX_RATIO, 0x3f,
		VERT_SHIFT, 0x00,
		VERT_MIRROR,
		CLOCK_FREQ, 0xf0,
		PRECHARGE, 0xf1,
		COM_CONFIG, COM_CONFIG2 | COM_CONFIG2_ALT,
		VCOMH_DESELECT_LEVEL, 0x40,
		CHARGE_PUMP, CHARGE_PUMP_ON,
		DISP_ON})

	return
}

// Draw the display as fast as I can
func (ssd1306 *SSD1306) Draw() (err error) {
	if ssd1306.iface == IFACE_I2C {
		for i := 0; i < len(ssd1306.buf); i += 32 {
			err = ssd1306.WriteData(ssd1306.buf[i:i+32])
			if err != nil {
				return
			}
		}
	}
	return
}

func (ssd1306 *SSD1306) WriteCmd(cmd []byte) (err error) {
	if ssd1306.iface == IFACE_I2C {
		var dc byte
		if len(cmd) == 1 {
			dc = 0x80
		} else {
			dc = 0x00
		}
		err = ssd1306.i2cbus.WriteI2C(dc, cmd)
		if err != nil {
			return
		}
	}
	return
}

func (ssd1306 *SSD1306) WriteData(data []byte) (err error) {
	if ssd1306.iface == IFACE_I2C {
		var dc byte
		if len(data) == 1 {
			dc = 0xc0
		} else {
			dc = 0x40
		}
		err = ssd1306.i2cbus.WriteI2C(dc, data)
		if err != nil {
			return
		}
	}
	return
}

func (ssd1306 *SSD1306) Clear(c color.Gray16) {
	var block byte
	if c == color.White {
		block = 0xff
	} else {
		block = 0x00
	}
	for i := 0; i < len(ssd1306.buf); i++ {
		ssd1306.buf[i] = block
	}
}

func (ssd1306 *SSD1306) Point(x, y int, c color.Gray16) {
	if x >= ssd1306.width || y >= ssd1306.height || x < 0 || y < 0 {
		return
	}

	element := ssd1306.width*(y/8) + x;
	if (c == color.White) {
		ssd1306.buf[element] |= 1 << (uint(y) % 8);
	} else {
		ssd1306.buf[element] &^= byte(1) << (uint(y) % 8);
	}
}

func (ssd1306 *SSD1306) Line(x0, y0, x1, y1 int, c color.Gray16) {
	dx := math.Abs(float64(x1) - float64(x0))
	dy := math.Abs(float64(y1) - float64(y0))
	var sx, sy int
	var err, e2 float64

	if x0 < x1 {
		sx = 1
	} else {
		sx = -1
	}
	if y0 < y1 {
		sy = 1
	} else {
		sy = -1
	}
	err = dx - dy

	for {
		ssd1306.Point(x0, y0, c)
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 = 2*err
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if x0 == x1 && y0 == y1 {
			ssd1306.Point(x0, y0, c)
			break
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
}

func (ssd1306 *SSD1306) Circle(x0, y0, radius int, c color.Gray16) {
	f := 1 - radius
	ddF_x := 1
	ddF_y := -2 * radius
	x := 0
	y := radius

	ssd1306.Point(x0, y0 + radius, c)
	ssd1306.Point(x0, y0 - radius, c)
	ssd1306.Point(x0 + radius, y0, c)
	ssd1306.Point(x0 - radius, y0, c)

	for x < y {
		if f >= 0 {
			y--
			ddF_y += 2
			f += ddF_y
		}
		x++
		ddF_x += 2
		f += ddF_x
		ssd1306.Point(x0 + x, y0 + y, c)
		ssd1306.Point(x0 - x, y0 + y, c)
		ssd1306.Point(x0 + x, y0 - y, c)
		ssd1306.Point(x0 - x, y0 - y, c)
		ssd1306.Point(x0 + y, y0 + x, c)
		ssd1306.Point(x0 - y, y0 + x, c)
		ssd1306.Point(x0 + y, y0 - x, c)
		ssd1306.Point(x0 - y, y0 - x, c)
	}
}

func (ssd1306 *SSD1306) Rectangle(x0, y0, x1, y1 int, c color.Gray16) {
	switch {
	// Ignore backwards rectangles
	case x0 > x1 || y0 > y1:
		return
	// If the rectangle is a line, draw it as one
	case x0 == x1 || y0 == y1:
		ssd1306.Line(x0, y0, x1, y1, c)
	// This case can be optimized a lot
	case y0 / 8 < y1 / 8: // Oh man, Vriska's gonna love all these 8's
		var element int
		b := ^byte(0) << uint(y0 % 8 - 1)

		for x := x0; x <= x1; x++ {
			element = ssd1306.width*(y0 / 8) + x;
			if (c == color.White) {
				ssd1306.buf[element] |= b;
			} else {
				ssd1306.buf[element] &^= b;
			}
		}

		for ; y0 / 8 < y1 / 8; y0 = (y0 / 8 * 8 + 8) { // Yeah!!!!!!!!
			b = ^byte(0)
			for x := x0; x <= x1; x++ {
				element = ssd1306.width*(y0 / 8) + x;
				if (c == color.White) {
					ssd1306.buf[element] |= b;
				} else {
					ssd1306.buf[element] &^= b;
				}
			}
		}

		b = ^byte(0) >> uint(7 - y1 % 8)
		for x := x0; x <= x1; x++ {
			element = ssd1306.width*(y1 / 8) + x;
			if (c == color.White) {
				ssd1306.buf[element] |= b;
			} else {
				ssd1306.buf[element] &^= b;
			}
		}
	// Further optimization is possible, but it's easier to just use lines
	default:
		for y := y0; y <= y1; y++ {
			ssd1306.Line(x0, y, x1, y, c)
		}
	}
}

// TODO: make this return an error instead of an int
func (ssd1306 *SSD1306) Char(x, y int, c color.Gray16, r rune) int {
	bufi := (ssd1306.width*(y/8)) + x
	bufiup := bufi - ssd1306.width

    if x >= ssd1306.width || y >= ssd1306.height {
        return -1
	}

    if r < 0 || r > 127 {
        return -1
	}

	if bufi < ssd1306.width * ssd1306.height / 8 && bufi >= 0 {
		for i := 0; i < 5 && x + i < ssd1306.width; i++ {
			if c == color.White {
				ssd1306.buf[bufi+i] |= font[uint((5*int(r))+i)] >> uint(8 - y % 8)
			} else {
				ssd1306.buf[bufi+i] &^= font[uint((5*int(r))+i)] >> uint(8 - y % 8)
			}
		}
	}
    if bufiup < ssd1306.width * ssd1306.height / 8 && bufiup >= 0 {
		for i := 0; i < 5 && x + i < ssd1306.width; i++ {
            if c == color.White {
                ssd1306.buf[bufiup+i] |= font[uint((5*int(r))+i)] << uint(y % 8)
            } else {
                ssd1306.buf[bufiup+i] &^= font[uint((5*int(r))+i)] << uint(y % 8)
			}
        }
    }

    return 0
}
