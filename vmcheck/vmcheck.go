// Copyright 2016 VMware, Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package vmcheck

import (
	"encoding/binary"
	"syscall"

	"github.com/vmware/vmw-guestinfo/bdoor"
)

func cpuid_low(arg1, arg2 uint32) (eax, ebx, ecx, edx uint32)

// IsVirtualWorld returns true if running in a VM.
func IsVirtualWorld() (bool, error) {
	// Test the HV bit is set
	if !CPUisVM() {
		return false, nil
	}

	// Test if backdoor port is available.
	if isVM, err := HypervisorPortCheck(); err != nil || !isVM {
		return isVM, err
	}

	return true, nil
}

func HypervisorPortCheck() (bool, error) {
	if err := syscall.Ioperm(int(bdoor.BackdoorPort), int(bdoor.BackdoorPort)+1, 1); err != nil {
		return false, err
	}

	p := &bdoor.BackdoorProto{}
	p.CX.Low.SetWord(bdoor.CommandGetVersion)

	out := p.InOut()

	// if there is no device, we get back all 1s
	return (0xffffffff != out.AX.Low.Word()) && (0 != out.AX.Low.Word()), nil
}

// CPUisVM checks for the HV bit in the ECX register of the CPUID leaf 0x1.
// Intel and AMD CPUs reserve this bit to indicate if the CPU is running
// in a HV. See https://en.wikipedia.org/wiki/CPUID#EAX.3D1:_Processor_Info_and_Feature_Bits
// for details.  If this bit is set, the reserved cpuid levels are used to pass
// information from the HV to the guest.  In ESX, this is the repeating string
// VMwareVMware.
func CPUisVM() bool {
	HV := uint32(1 << 31)
	_, _, c, _ := cpuid_low(0x1, 0)
	if (c & HV) != HV {
		return false
	}

	_, b, c, d := cpuid_low(0x40000000, 0)

	buf := make([]byte, 12)
	binary.LittleEndian.PutUint32(buf, b)
	binary.LittleEndian.PutUint32(buf[4:], c)
	binary.LittleEndian.PutUint32(buf[8:], d)

	if string(buf) != "VMwareVMware" {
		return false
	}

	return true
}
