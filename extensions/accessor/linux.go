// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See LICENSE in the project root for license information.

//go:build linux
// +build linux

package accessor

/*
#cgo LDFLAGS: -ldl
#include <dlfcn.h>
#include <stdlib.h>

// This C API wraps libsecret so that Go code can call into it without requiring libsecret
// at build time. The API is simpler than libsecret's because some values (schema attributes
// and label) are the same in all cases. (Caches are distinguished by the schema name.)

typedef struct
{
    int domain;
    int code;
    char *message;
} gError;

typedef struct
{
    const char *name;
    int type;
} schemaAttribute;

typedef struct
{
    char *name;
    int flags;
    schemaAttribute attributes[32];
    // private fields
    int r;
    char *r2;
    char *r3;
    char *r4;
    char *r5;
    char *r6;
    char *r7;
    char *r8;
} schema;

schema *new_schema(char *name)
{
    schema *s;
    s = malloc(sizeof(schema));
    s->name = name;
    s->attributes[0] = (schemaAttribute){"MsalClientID", 0};
    s->attributes[1] = (schemaAttribute){NULL, 0};
    return s;
}

// lookup a password. f must point to secret_password_lookup_sync
char *lookup(void *f, schema *sch, gError **err)
{
    char *(*fn)(schema * s, void *cancellable, gError **err, char *attrKey1, char *attrValue1, ...);
    fn = (char *(*)(schema * s, void *cancellable, gError **err, char *attrKey1, char *attrValue1, ...)) f;
    char *pw = fn(sch, NULL, err, "MsalClientID", "Microsoft.Developer.IdentityService", NULL);
    return pw;
}

// store a password. f must point to secret_password_store_sync
int store(void *f, schema *sch, char *password, gError **err)
{
    int (*fn)(schema * s, char *collection, char *label, char *data, void *cancellable, gError **err, ...);
    fn = (int (*)(schema * s, char *collection, char *label, char *data, void *cancellable, gError **err, ...)) f;
    int b = fn(sch, NULL, "MSALCache", password, NULL, err, "MsalClientID", "Microsoft.Developer.IdentityService", NULL);
    return b;
}
*/
import "C"
import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"unsafe"
)

// Accessor stores data on the GNOME keyring using libsecret
type Accessor struct {
	// handle is an opaque handle for libsecret returned by dlopen(). It should be
	// released via dlclose() when no longer needed so the loader knows whether it's
	// safe to unload libsecret.
	handle unsafe.Pointer
	// lookup and store are the addresses of libsecret functions
	lookup, store unsafe.Pointer
	schema        *C.schema
}

// New is the constructor for LibsecretAccessor. "name" distinguishes one cache from another.
func New(name string) (*Accessor, error) {
	n := C.CString("libsecret-1.so")
	defer C.free(unsafe.Pointer(n))
	// set the handle and finalizer first so any handle will be
	// released even when this constructor goes on to return an error
	ls := &Accessor{handle: C.dlopen(n, C.RTLD_LAZY)}
	if ls.handle == nil {
		return nil, fmt.Errorf("couldn't load %s", C.GoString(n))
	}
	runtime.SetFinalizer(ls, func(a *Accessor) {
		if a.handle != nil {
			C.dlclose(a.handle)
		}
		if a.schema != nil {
			C.free(unsafe.Pointer(a.schema.name))
			C.free(unsafe.Pointer(a.schema))
		}
	})
	lookup, err := ls.symbol("secret_password_lookup_sync")
	if err != nil {
		return nil, err
	}
	store, err := ls.symbol("secret_password_store_sync")
	if err != nil {
		return nil, err
	}
	ls.lookup = lookup
	ls.store = store
	ls.schema = C.new_schema(C.CString(name))
	return ls, nil
}

// Read returns cache data stored by libsecret
func (s *Accessor) Read(context.Context) ([]byte, error) {
	var e *C.gError
	defer C.free(unsafe.Pointer(e))

	data := C.lookup(s.lookup, s.schema, &e)
	if e != nil {
		return nil, fmt.Errorf("failed to read cache from keyring: %q", C.GoString(e.message))
	}
	defer C.free(unsafe.Pointer(data))
	return []byte(C.GoString(data)), nil
}

// Write stores cache data via libsecret
func (s *Accessor) Write(ctx context.Context, data []byte) error {
	var e *C.gError
	defer C.free(unsafe.Pointer(e))

	pw := C.CString(string(data))
	defer C.free(unsafe.Pointer(pw))
	if r := C.store(s.store, s.schema, pw, &e); r == 0 {
		return errors.New("failed to write cache to keyring")
	} else if e != nil {
		return errors.New(C.GoString(e.message))
	}
	return nil
}

func (s *Accessor) symbol(name string) (unsafe.Pointer, error) {
	n := C.CString(name)
	defer C.free(unsafe.Pointer(n))
	C.dlerror()
	fp := C.dlsym(s.handle, n)
	if er := C.dlerror(); er != nil {
		return nil, fmt.Errorf("couldn't load %q: %s", name, C.GoString(er))
	}
	return fp, nil
}

var _ Cache = (*Accessor)(nil)
