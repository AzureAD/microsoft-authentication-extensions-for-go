package internal

import (
	"fmt"

	"github.com/billgraziano/dpapi"
)

func encryptWindows() {
	secret := "Hello World!;"

	enc, err := dpapi.Encrypt(secret)
	if err != nil {
		fmt.Println("err from Encrypt: ", err)
	}
	fmt.Println(enc)
	dec, err := dpapi.Decrypt(enc)
	if err != nil {
		fmt.Println("err from Decrypt: ", err)
	}
	fmt.Println(dec)
	if dec != secret {
		fmt.Printf("expected: '%s' got: '%s'", secret, dec)
	}

}

func encryptWindowsusingLibraries() {

}

func main() {
	encryptWindows()
}
