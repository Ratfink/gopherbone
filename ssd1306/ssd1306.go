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
	// Use the "refactor" branch - switch back to "bitbucket.org/gmcbay/i2c"
	// eventually, when the pull request is accepted.
//	"bitbucket.org/corburn/i2c"
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

    //--turn off oled panel
    //--set start line address
    //--set memory addressing mode
    //---horizontal addressing mode
    //--set contrast control register
    //--
    //--set segment re-map 95 to 0
    //--set normal display
    //--set multiplex ratio(1 to 64)
    //--1/64 duty
    //-set display offset
    //-not offset
    //--set display clock divide ratio/oscillator frequency
    //--set divide ratio
    //--set pre-charge period
    //--
    //--set com pins hardware configuration
    //--
    //--set vcomh
    //--
    //--set Charge Pump enable/disable
    //--set(0x10) disable
    //--turn on oled panel
	ssd1306.WriteCmd([]byte{0xae, 0x40, 0x20, 0x00, 0x81, 0xcf, 0xa1, 0xa6, 0xa8, 0x3f, 0xd3, 0x00, 0xd5, 0xf0, 0xd9, 0xf1, 0xda, 0x12, 0xdb, 0x40, 0x8d, 0x14, 0xaf});
/*	bone_ssd1306_cmd(disp, 0x40);
	bone_ssd1306_cmd(disp, 0x20);
	bone_ssd1306_cmd(disp, 0x00);
	bone_ssd1306_cmd(disp, 0x81);
	bone_ssd1306_cmd(disp, 0xCF);
	bone_ssd1306_cmd(disp, 0xA1);
	bone_ssd1306_cmd(disp, 0xA6);
	bone_ssd1306_cmd(disp, 0xA8);
	bone_ssd1306_cmd(disp, 0x3F);
	bone_ssd1306_cmd(disp, 0xD3);
	bone_ssd1306_cmd(disp, 0x00);
	bone_ssd1306_cmd(disp, 0xD5);
	bone_ssd1306_cmd(disp, 0xF0);
	bone_ssd1306_cmd(disp, 0xD9);
	bone_ssd1306_cmd(disp, 0xF1);
	bone_ssd1306_cmd(disp, 0xDA);
	bone_ssd1306_cmd(disp, 0x12);
	bone_ssd1306_cmd(disp, 0xDB);
	bone_ssd1306_cmd(disp, 0x40);
	bone_ssd1306_cmd(disp, 0x8D);
	bone_ssd1306_cmd(disp, 0x14);
	bone_ssd1306_cmd(disp, 0xAF);*/
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

	element := ssd1306.width*(y>>3) + x;
    if (c == color.White) {
        ssd1306.buf[element] |= 1 << (y % 8);
    } else {
        ssd1306.buf[element] &^= byte(1) << (y % 8);
    }
}
