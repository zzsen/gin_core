package encrypt

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAesEcbEncrypt(t *testing.T) {
	type args struct {
		src2Encrypt string
		key         string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "aes ecn encrypt",
			args: args{
				src2Encrypt: "Hello World",
				key:         "UTabIUiHgDyh464+",
			},
			want:    "/t8wxJyz5nLKYDa7w8W3oQ==",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AesEcbEncrypt(tt.args.src2Encrypt, tt.args.key)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestAesEcbDecrypt(t *testing.T) {
	type args struct {
		src2Decrypt string
		key         string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "aes ecb decrypt",
			args: args{
				src2Decrypt: "/t8wxJyz5nLKYDa7w8W3oQ==",
				key:         "UTabIUiHgDyh464+",
			},
			want:    "Hello World",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AesEcbDecrypt(tt.args.src2Decrypt, tt.args.key)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestAesEcbCrypt(t *testing.T) {
	type args struct {
		src2Encrypt   string
		encryptResult string
		key           string
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "aes ecb crypt",
			args: args{
				src2Encrypt:   "Hello World",
				encryptResult: "/t8wxJyz5nLKYDa7w8W3oQ==",
				key:           "UTabIUiHgDyh464+",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			encrypyResult, err := AesEcbEncrypt(tt.args.src2Encrypt, tt.args.key)
			assert.Nil(t, err)

			decrypyResult, err := AesEcbDecrypt(encrypyResult, tt.args.key)
			assert.Nil(t, err)

			assert.Equal(t, decrypyResult, tt.args.src2Encrypt)
		})
	}
}
