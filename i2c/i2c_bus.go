package i2c

import (
	"fmt"
	"os"
	"sync"
	"syscall"
	"unsafe"
)

// as defined in /usr/include/linux/i2c-dev.h
const (
	I2C_SLAVE = 0x0703
	I2C_SMBUS = 0x0720
)

// as defined in /usr/include/linux/i2c.h
const (
	I2C_SMBUS_WRITE            = 0
	I2C_SMBUS_READ             = 1
	I2C_SMBUS_I2C_BLOCK_BROKEN = 6
	I2C_SMBUS_I2C_BLOCK_DATA   = 8
	I2C_SMBUS_BLOCK_MAX        = 32
)

var busMap map[byte]*Bus
var busMapLock sync.Mutex

func init() {
	busMap = make(map[byte]*Bus)
}

// as defined in /usr/include/linux/i2c-dev.h
type i2c_smbus_ioctl_data struct {
	readWrite byte
	command   byte
	size      uint32
	data      uintptr
}

type Bus struct {
	// i2c-dev file pointer
	file *os.File
	// last transmitted address, track here
	// so we don't have to redo the ioctl
	// call if the address hasn't changed since the
	// last access
	addr byte
	// simple bus access lock to ensure address
	// set and data writes occur atomically
	lock sync.Mutex
}

// Returns an instance to an I2CBus.  If we already have an I2CBus
// created for the requested bus number, just return that, otherwise
// set up a new one and open up its associated i2c-dev file
func NewBus(addr, bus byte) (i2cbus *Bus, err error) {
	busMapLock.Lock()
	defer busMapLock.Unlock()

	if i2cbus = busMap[bus]; i2cbus == nil {
		i2cbus = new(Bus)
		if i2cbus.file, err = os.OpenFile(fmt.Sprintf("/dev/i2c-%d", bus), os.O_RDWR, os.ModeExclusive); err == nil {
			busMap[bus] = i2cbus
			err = i2cbus.SetAddress(addr)
		}
	}

	return
}

func (i2cbus *Bus) SetAddress(addr byte) (err error) {
	if addr != i2cbus.addr {
		if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL, i2cbus.file.Fd(), I2C_SLAVE, uintptr(addr)); errno != 0 {
			err = syscall.Errno(errno)
			return
		}

		i2cbus.addr = addr
	}

	return
}

func (i2cbus *Bus) Read(reg byte, readLength byte) (list []byte, err error) {
	i2cbus.lock.Lock()
	defer i2cbus.lock.Unlock()

	blockData := make([]byte, readLength+1)
	blockData[0] = readLength

	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		i2cbus.file.Fd(), I2C_SMBUS, uintptr(unsafe.Pointer(&i2c_smbus_ioctl_data{
			readWrite: I2C_SMBUS_READ,
			command:   reg,
			size:      I2C_SMBUS_I2C_BLOCK_DATA,
			data:      uintptr(unsafe.Pointer(&blockData[0]))}))); errno != 0 {
		err = syscall.Errno(errno)
	}

	list = make([]byte, readLength)
	copy(list, blockData[1:])

	return
}

func (i2cbus *Bus) Write(reg byte, list []byte) (err error) {
	i2cbus.lock.Lock()
	defer i2cbus.lock.Unlock()

	blockData := make([]byte, len(list)+1)
	blockData[0] = byte(len(list))
	copy(blockData[1:], list)

	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		i2cbus.file.Fd(), I2C_SMBUS, uintptr(unsafe.Pointer(&i2c_smbus_ioctl_data{
			readWrite: I2C_SMBUS_WRITE,
			command:   reg,
			size:      I2C_SMBUS_I2C_BLOCK_DATA,
			data:      uintptr(unsafe.Pointer(&blockData[0]))}))); errno != 0 {
		err = syscall.Errno(errno)
	}

	return
}

func (i2cbus *Bus) WriteI2C(reg byte, list []byte) (err error) {
	i2cbus.lock.Lock()
	defer i2cbus.lock.Unlock()

	blockData := make([]byte, len(list)+1)
	blockData[0] = byte(len(list))
	copy(blockData[1:], list)

	if _, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		i2cbus.file.Fd(), I2C_SMBUS, uintptr(unsafe.Pointer(&i2c_smbus_ioctl_data{
			readWrite: I2C_SMBUS_WRITE,
			command:   reg,
			size:      I2C_SMBUS_I2C_BLOCK_BROKEN,
			data:      uintptr(unsafe.Pointer(&blockData[0]))}))); errno != 0 {
		err = syscall.Errno(errno)
	}

	return
}
