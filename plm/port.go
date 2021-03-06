// Copyright 2018 Andrew Bates
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package plm

import (
	"bufio"
	"io"
	"time"

	"github.com/abates/insteon"
)

type Port struct {
	in      *bufio.Reader
	out     io.Writer
	timeout time.Duration
}

func NewPort(readWriter io.ReadWriter, timeout time.Duration) *Port {
	port := &Port{
		in:      bufio.NewReader(readWriter),
		out:     readWriter,
		timeout: timeout,
	}
	return port
}

func (port *Port) Write(buf []byte) {
	insteon.Log.Tracef("TX %s", hexDump("%02x", buf, " "))
	_, err := port.out.Write(buf)
	if err != nil {
		insteon.Log.Infof("Failed to write: %v", err)
	}
}

func (port *Port) Read() (buf []byte, err error) {
	// synchronize
	for err == nil {
		var b byte
		b, err = port.in.ReadByte()
		if err != nil {
			break
		}

		// first byte of PLM packets is always 0x02
		if b != 0x02 {
			insteon.Log.Tracef("Expected Start of Text (0x02) got 0x%02x", b)
			continue
		} else {
			b, err = port.in.ReadByte()
			if packetLen, found := commandLens[Command(b)]; found {
				buf = make([]byte, packetLen+2)
				buf[0] = 0x02
				buf[1] = b
				insteon.Log.Tracef("Attempting to read %d more bytes", packetLen)
				_, err = io.ReadAtLeast(port.in, buf[2:], packetLen)
				insteon.Log.Tracef("Completed read (err %v): %s", err, hexDump("%02x", buf, " "))
				break
			} else {
				err = port.in.UnreadByte()
			}
		}
	}

	if err == nil {
		// read some more if it's an extended message
		if buf[1] == 0x62 && insteon.Flags(buf[5]).Extended() {
			buf = append(buf, make([]byte, 14)...)
			_, err = io.ReadAtLeast(port.in, buf[9:], 14)
		}

		if err == nil {
			insteon.Log.Tracef("RX %s", hexDump("%02x", buf, " "))
		}
	}

	return buf, err
}

func (port *Port) Close() (err error) {
	if closer, ok := port.out.(io.Closer); ok {
		err = closer.Close()
	}
	return err
}
