//go:build gnome_keyring
// +build gnome_keyring

package internal

/*
#cgo pkg-config: libsecret-1 glib-2.0
#include <stdlib.h>
#include "libsecret/secret.h"

SecretSchema keyring_schema;

void new_schema(gchar *name, gchar *attrKey1, ghar *attrKey2) {
	keyring_schema =
	{
		name,
		SECRET_SCHEMA_NONE,
		{
		{ attrKey1, SECRET_SCHEMA_ATTRIBUTE_STRING },
		{ attrKey2,  SECRET_SCHEMA_ATTRIBUTE_STRING },
		{  NULL, 0 },
		}
	};
}


// wrap the gnome calls because cgo can't deal with vararg functions
gboolean gkr_set_password(gchar *description, gchar *attrKey1, gchar *attrValue1, gchar *attrKey2, gchar *attrValue2, gchar *data, GError **err) {
	return secret_password_store_sync(
		&keyring_schema,
		NULL,
		description,
		data,
    NULL,
    err,
		attrKey1, attrValue1,
		attrKey2, attrValue2,
		NULL);
}

gchar * gkr_get_password(gchar *attrKey1, gchar *attrValue1, gchar *attrKey2, gchar *attrValue2, GError **err) {
	return secret_password_lookup_sync(
		&keyring_schema,
    NULL,
		err,
		attrKey1, attrValue1,
		attrKey2, attrValue2,
		NULL);
}
*/
import "C"

import (
	"fmt"
	"unsafe"
)

type GnomeKeyRing struct{}

func (p GnomeKeyRing) New(name string, flag int, attrKey1 string, attrKey2 string) {
	C.new_schema(name, attrKey1, attrKey2)
}
func (p GnomeKeyRing) Set(label, data, attrKey1, attrValue1, attrKey2, attrValue2 string) error {
	desc := (*C.gchar)(C.CString(label))
	attrKey1 := (*C.gchar)(C.CString(attrKey1))
	attrValue1 := (*C.gchar)(C.CString(attrValue1))
	attrKey2 := (*C.gchar)(C.CString(attrKey2))
	attrValue2 := (*C.gchar)(C.CString(attrValue2))
	data := (*C.gchar)(C.CString(data))
	defer C.free(unsafe.Pointer(desc))
	defer C.free(unsafe.Pointer(attrKey1))
	defer C.free(unsafe.Pointer(attrValue1))
	defer C.free(unsafe.Pointer(attrKey2))
	defer C.free(unsafe.Pointer(attrValue2))
	defer C.free(unsafe.Pointer(data))

	var gerr *C.GError
	result := C.gkr_set_password(desc, attrKey1, attrValue1, attrKey2, attrValue2, data, &gerr)
	defer C.free(unsafe.Pointer(gerr))

	if result == 0 {
		return fmt.Errorf("Gnome-keyring error: %+v", gerr)
	}
	return nil
}

func (p GnomeKeyRing) Get(attrKey1, attrValue1, attrKey2, attrValue2 string) (string, error) {
	var gerr *C.GError
	var pw *C.gchar

	attrKey1 := (*C.gchar)(C.CString(attrKey1))
	attrValue1 := (*C.gchar)(C.CString(attrValue1))
	attrKey2 := (*C.gchar)(C.CString(attrKey2))
	attrValue2 := (*C.gchar)(C.CString(attrValue2))

	defer C.free(unsafe.Pointer(attrKey1))
	defer C.free(unsafe.Pointer(attrValue1))
	defer C.free(unsafe.Pointer(attrKey2))
	defer C.free(unsafe.Pointer(attrValue2))

	pw = C.gkr_get_password(attrKey1, attrValue1, attrKey2, attrValue2, &gerr)
	defer C.free(unsafe.Pointer(gerr))
	defer C.secret_password_free((*C.gchar)(pw))

	if pw == nil {
		return "", fmt.Errorf("Gnome-keyring error: %+v", gerr)
	}
	return C.GoString((*C.char)(pw)), nil
}

func initializeProvider() (provider, error) {
	return GnomeKeyRing{}, nil
}
