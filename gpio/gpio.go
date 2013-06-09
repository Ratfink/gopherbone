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

/* This GPIO system uses the sysfs interface to control digital inputs and
 * outputs.  It would probably be better to use the interface built in to the
 * kernel, but sysfs is easy and safe.
 */
package gpio

import (
	"fmt"
	"os"
)

// P8 is an array of pin values made for conveniently referring to pins on the
// BeagleBone's P8 header.
var P8 = [47]int{
    -1,  // P8
    -1,  // GND
    -1,  // GND
    38,  // GPIO1_6
    39,  // GPIO1_7
    34,  // GPIO1_2
    35,  // GPIO1_3
    66,  // TIMER4
    67,  // TIMER7
    69,  // TIMER5
    68,  // TIMER6
    45,  // GPIO1_13
    44,  // GPIO1_12
    23,  // EHRPWM2B
    26,  // GPIO0_26
    47,  // GPIO1_15
    46,  // GPIO1_14
    27,  // GPIO0_27
    65,  // GPIO2_1
    22,  // EHRPWM2A
    63,  // GPIO1_31
    62,  // GPIO1_30
    37,  // GPIO1_5
    36,  // GPIO1_4
    33,  // GPIO1_1
    32,  // GPIO1_0
    61,  // GPIO1_29
    54,  // GPIO1_22
    56,  // GPIO1_24
    55,  // GPIO1_23
    57,  // GPIO1_25
    10,  // UART5_CTSN
    11,  // UART5_RTSN
    9,   // UART4_RTSN
    81,  // UART3_RTSN
    8,   // UART4_CTSN
    80,  // UART3_CTSN
    78,  // UART5_TXD
    79,  // UART5_RXD
    76,  // GPIO2_12
    77,  // GPIO2_13
    74,  // GPIO2_10
    75,  // GPIO2_11
    72,  // GPIO2_8
    73,  // GPIO2_9
    70,  // GPIO2_6
    71,  // GPIO2_7
}

// P9 is an array of pin values made for conveniently referring to pins on the
// BeagleBone's P9 header.
var P9 = [47]int{
    -1,  // P9
    -1,  // GND
    -1,  // GND
    -1,  // DC_3.3V
    -1,  // DC_3.3V
    -1,  // VDD_5V
    -1,  // VDD_5V
    -1,  // SYS_5V
    -1,  // SYS_5V
    -1,  // PWR_BUT
    -1,  // SYS_RESETn
    30,  // UART4_RXD
    60,  // GPIO1_28
    31,  // UART4_TXD
    50,  // EHRPWM1A
    48,  // GPIO1_16
    51,  // EHRPWM1B
    5,   // I2C1_SCL
    4,   // I2C1_SDA
    13,  // I2C2_SCL
    12,  // I2C2_SDA
    3,   // UART2_TXD
    2,   // UART2_RXD
    49,  // GPIO1_17
    15,  // UART1_TXD
    117, // GPIO3_21
    14,  // UART1_RXD
    115, // GPIO3_19
    113, // SPI1_CS0
    111, // SPI1_D0
    112, // SPI1_D1
    110, // SPI1_SCLK
    -1,  // VDD_ADC (1.8V)
    -1,  // AIN4
    -1,  // GNDA_ADC
    -1,  // AIN6
    -1,  // AIN5
    -1,  // AIN2
    -1,  // AIN3
    -1,  // AIN0
    -1,  // AIN1
    20,  // CLKOUT2
    7,   // GPIO0_7
    -1,  // GND
    -1,  // GND
    -1,  // GND
    -1,  // GND
}

// A GPIO structure represents a GPIO pin on the BeagleBone, or for that matter
// any Linux system.  To use a GPIO, create a pointer to a GPIO struct using
// the Create function, passing the number of the pin requested.
type GPIO struct {
	Pin int
	ValueFile *os.File
}

// Export creates a GPIO structure from the specified pin, exports the pin to
// sysfs, and returns the GPIO structure.
func Export(pin int) (gpio *GPIO, err error) {
	gpio = new(GPIO)
	var f *os.File

	_, err = os.Stat(fmt.Sprintf("/sys/class/gpio/gpio%d", pin))
	if err != nil && os.IsNotExist(err) {
		f, err = os.OpenFile("/sys/class/gpio/export", os.O_WRONLY, 0666)
		if err != nil {
			return
		}
		defer f.Close()

		_, err = fmt.Fprintf(f, "%d", pin)
		if err != nil {
			return
		}
	}
	gpio.Pin = pin
	gpio.ValueFile = nil

	return
}

// Unexport removes the sysfs entry of a GPIO.
func (gpio *GPIO) Unexport() (err error) {
	if gpio.ValueFile != nil {
		err = gpio.CloseValue()
		if err != nil {
			return
		}
	}
	f, err := os.OpenFile("/sys/class/gpio/unexport", os.O_WRONLY, 0666)
	if err != nil {
		return
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, "%d", gpio.Pin)
	// Don't bother checking for errors here because we're returning anyway

	return
}

// Value returns the current value of a GPIO.  If the pin is an output, this
// value is the one set by SetValue; if the pin is an input, the value comes
// from the outside world.
func (gpio *GPIO) Value() (value int, err error) {
	var f *os.File
	if gpio.ValueFile == nil {
		f, err = os.OpenFile(fmt.Sprintf("/sys/class/gpio/gpio%d/value", gpio.Pin), os.O_RDONLY, 0666)
		if err != nil {
			return
		}
		defer f.Close()
	} else {
		f = gpio.ValueFile
		f.Seek(0, 0)
	}

	n, err := fmt.Fscanf(f, "%d", &value)
	if n != 1 {
		err = fmt.Errorf("Bad number of values read from /sys/class/gpio/gpio%d/value: %d", gpio.Pin, n)
	}

	return
}

// SetValue sets the value of an output pin.
func (gpio *GPIO) SetValue(value int) (err error) {
	var f *os.File
	if value != 0 && value != 1 {
		err = fmt.Errorf("Invalid value: %d", value)
		return
	}
	if gpio.ValueFile == nil {
		f, err = os.OpenFile(fmt.Sprintf("/sys/class/gpio/gpio%d/value", gpio.Pin), os.O_WRONLY, 0666)
		if err != nil {
			return
		}
		defer f.Close()
	} else {
		f = gpio.ValueFile
		f.Seek(0, 0)
	}

	_, err = fmt.Fprintf(f, "%d", value)

	return
}

// OpenValue opens the GPIO's value file for reading and writing.  The open
// file is kept in the GPIO struct's ValueFile member.
func (gpio *GPIO) OpenValue() (err error) {
	gpio.ValueFile, err = os.OpenFile(fmt.Sprintf("/sys/class/gpio/gpio%d/value", gpio.Pin), os.O_RDWR, 0666)

	return
}

func (gpio *GPIO) CloseValue() (err error) {
	gpio.ValueFile.Close()
	if err != nil {
		return
	}

	gpio.ValueFile = nil

	return
}

// Direction sets returns the current direction of a pin.  This may be either
// "in" or "out".
func (gpio *GPIO) Direction() (dir string, err error) {
	f, err := os.OpenFile(fmt.Sprintf("/sys/class/gpio/gpio%d/direction", gpio.Pin), os.O_RDONLY, 0666)
	if err != nil {
		return
	}
	defer f.Close()

	n, err := fmt.Fscanf(f, "%s", &dir)
	if n != 1 {
		err = fmt.Errorf("Bad number of values read from /sys/class/gpio/gpio%d/direction: %d", gpio.Pin, n)
	}

	return
}

// SetDirection sets a pin's direction, input or output.  The argument must be
// either "in" or "out".
func (gpio *GPIO) SetDirection(dir string) (err error) {
	if dir != "in" && dir != "out" {
		err = fmt.Errorf("Invalid direction: %s", dir)
		return
	}
	f, err := os.OpenFile(fmt.Sprintf("/sys/class/gpio/gpio%d/direction", gpio.Pin), os.O_WRONLY, 0666)
	if err != nil {
		return
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, "%s", dir)

	return
}

// Edge returns the current edge(s) for which polling this pin's value file
// will return.
func (gpio *GPIO) Edge() (edge string, err error) {
	f, err := os.OpenFile(fmt.Sprintf("/sys/class/gpio/gpio%d/edge", gpio.Pin), os.O_RDONLY, 0666)
	if err != nil {
		return
	}
	defer f.Close()

	n, err := fmt.Fscanf(f, "%s", &edge)
	if n != 1 {
		err = fmt.Errorf("Bad number of values read from /sys/class/gpio/gpio%d/edge: %d", gpio.Pin, n)
	}

	return
}

// SetEdge sets the edge(s) for which polling this pin's value file will
// return.
func (gpio *GPIO) SetEdge(edge string) (err error) {
	if edge != "none" && edge != "rising" && edge != "falling" && edge != "both" {
		err = fmt.Errorf("Invalid edge: %s", edge)
		return
	}
	f, err := os.OpenFile(fmt.Sprintf("/sys/class/gpio/gpio%d/edge", gpio.Pin), os.O_WRONLY, 0666)
	if err != nil {
		return
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, "%s", edge)

	return
}
