package valid

import (
	"testing"
)

func TestValidateStruct(t *testing.T) {
	type TestStruct struct {
		Field1 int    `validate:"required,gte=5,lte=10"`
		Field2 string `validate:"required,min=5,max=10"`
		Field3 string `validate:"required,email"`
		Field4 string `validate:"url"`
	}

	type args struct {
		s interface{}
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "success",
			args: args{TestStruct{
				Field1: 5,
				Field2: "qwerty123",
				Field3: "example@mail.com",
				Field4: "http://example.com",
			}},
			wantErr: false,
		},
		{
			name: "error with less int",
			args: args{TestStruct{
				Field1: 4,
				Field2: "qwerty123",
				Field3: "example@mail.com",
				Field4: "http://example.com",
			}},
			wantErr: true,
		},
		{
			name: "error with much int",
			args: args{TestStruct{
				Field1: 11,
				Field2: "qwerty123",
				Field3: "example@mail.com",
				Field4: "http://example.com",
			}},
			wantErr: true,
		},
		{
			name: "failed with empty field",
			args: args{TestStruct{
				Field1: 10,
				Field2: "",
				Field3: "example@mail.com",
				Field4: "http://example.com",
			}},
			wantErr: true,
		},
		{
			name: "failed with largest string field",
			args: args{TestStruct{
				Field1: 10,
				Field2: "qwerty12345678",
				Field3: "example@mail.com",
				Field4: "http://example.com",
			}},
			wantErr: true,
		},
		{
			name: "failed with less len string field",
			args: args{TestStruct{
				Field1: 10,
				Field2: "qwe",
				Field3: "example@mail.com",
				Field4: "http://example.com",
			}},
			wantErr: true,
		},
		{
			name: "failed with invalid email",
			args: args{TestStruct{
				Field1: 10,
				Field2: "qwerty123",
				Field3: "example.com",
				Field4: "http://example.com",
			}},
			wantErr: true,
		},
		{
			name: "failed with invalid url",
			args: args{TestStruct{
				Field1: 10,
				Field2: "qwerty123",
				Field3: "example@mail.com",
				Field4: "examplecom",
			}},
			wantErr: true,
		},
		{
			name:    "failed with empty",
			args:    args{TestStruct{}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := ValidateStruct(tt.args.s); (err != nil) != tt.wantErr {
				t.Errorf("ValidateStruct() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
