// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See LICENSE in the project root for license information.

package accessor

import "context"

// Cache provides storage.
type Cache interface {
	Read(context.Context) ([]byte, error)
	Write(context.Context, []byte) error
}
