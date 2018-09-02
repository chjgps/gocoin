package zk

/*
#cgo CFLAGS:-DMCLBN_FP_UNIT_SIZE=6 -I${SRCDIR}/include
#cgo LDFLAGS:-L${SRCDIR}/lib  -lppzksnark -lff -lsnark -lzm -lgmpxx -lm -lgmp -lprocps -lstdc++
#include "ppzksnark.h"
*/
import "C"
import "fmt"
import "unsafe"

func R1cs_ppzkadsnark_Generator(filepath string, num_constraints, num_inputs int) error {

	buf := []byte(filepath)

	err := C.r1cs_ppzkadsnark_Generator((*C.char)(unsafe.Pointer(&buf[0])), C.int(num_constraints), C.int(num_inputs))
	if err != 0 {
		return fmt.Errorf("err r1cs_ppzkadsnark_Generator %x", err)
	}
	return nil
}

func R1cs_ppzkSNARK_Prover(filepath string) error {
	buf := []byte(filepath)
	err := C.r1cs_ppzkSNARK_Prover((*C.char)(unsafe.Pointer(&buf[0])))
	if err != 0 {
		return fmt.Errorf("err r1cs_ppzkSNARK_Prover %x", err)
	}
	return nil
}

func R1cs_ppzkSNARK_Verifier(vk, input, proof string) error {

	vkbuf := []byte(vk)
	vkinput := []byte(input)
	vkproof := []byte(proof)

	err := C.r1cs_ppzkSNARK_Verifier((*C.char)(unsafe.Pointer(&vkbuf[0])), (*C.char)(unsafe.Pointer(&vkinput[0])), (*C.char)(unsafe.Pointer(&vkproof[0])))
	if err != 0 {
		return fmt.Errorf("err r1cs_ppzkSNARK_Verifier %x", err)
	}
	return nil
}
