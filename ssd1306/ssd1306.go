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
    width uint
    height uint
    buf []byte
}

func New(rstpin, iface int, addr, bus byte, width, height uint) (ssd1306 *SSD1306, err error) {
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

func (ssd1306 *SSD1306) Point(x, y uint, c color.Gray16) {
	if x >= ssd1306.width || y >= ssd1306.height {
		return
	}

	element := ssd1306.width*(y/8) + x;
    if (c == color.White) {
        ssd1306.buf[element] |= 1 << (y % 8);
    } else {
        ssd1306.buf[element] &^= byte(1) << (y % 8);
    }
}
