package aesUtil

import (
	"fmt"
)

func Test() {
	content := "Hello World"
	aes := Ecb{Key: "UTabIUiHgDyh464+"}
	encrypyResult, err := aes.Encrypt(content)
	if err != nil {
		fmt.Printf("\033[31maes encrypt failed: %v\n\033[0m", err)
		return
	}
	fmt.Println("aes encrypt result: ", encrypyResult)

	decrypyResult, err := aes.Decrypt(encrypyResult)
	if err != nil {
		fmt.Printf("\033[31maes decrypt failed: %v\n\033[0m", err)
		return
	}
	if decrypyResult != content {
		fmt.Println("\033[31maes decrypt failed: content is not equal\033[0m")
		return
	}
	fmt.Println("\033[32maes test successed\033[0m")
}
