//go:build windows

package main

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

// A titkos adatok (forrás-tokenek, partnertörzs) Windows DPAPI-val titkosítva kerülnek
// a lemezre: csak ugyanaz a Windows-felhasználó tudja visszafejteni, ugyanazon a gépen.

var dpapiEntropy = []byte("Energofish-Hirlevel-Partnertorzs")

func blob(b []byte) *windows.DataBlob {
	if len(b) == 0 {
		return &windows.DataBlob{}
	}
	return &windows.DataBlob{Size: uint32(len(b)), Data: &b[0]}
}

func dpapi(data []byte, encrypt bool) ([]byte, error) {
	var out windows.DataBlob
	var err error
	if encrypt {
		err = windows.CryptProtectData(blob(data), nil, blob(dpapiEntropy), 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out)
	} else {
		err = windows.CryptUnprotectData(blob(data), nil, blob(dpapiEntropy), 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out)
	}
	if err != nil {
		return nil, err
	}
	defer windows.LocalFree(windows.Handle(unsafe.Pointer(out.Data)))
	return append([]byte(nil), unsafe.Slice(out.Data, out.Size)...), nil
}

var (
	protectSecret   = func(b []byte) ([]byte, error) { return dpapi(b, true) }
	unprotectSecret = func(b []byte) ([]byte, error) { return dpapi(b, false) }
)

const secretsProtected = true
