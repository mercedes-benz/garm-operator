package secret

import (
	"os"
	"testing"
)

func TestReadFromFile(t *testing.T) {
	tests := []struct {
		name                string
		secretValue         string
		expected            string
		wantErr             bool
		notExistingFilePath string
	}{
		{
			name:        "read secret from path and strip new line char",
			secretValue: "mysecretvalue\n",
			expected:    "mysecretvalue",
			wantErr:     false,
		},
		{
			name:        "read secret from path and strip new line chars",
			secretValue: "\nmysecretvalue\n",
			expected:    "mysecretvalue",
			wantErr:     false,
		},
		{
			name:        "read secret from path and strip new line and tab char",
			secretValue: "\n\tmysecretvalue\n",
			expected:    "mysecretvalue",
			wantErr:     false,
		},
		{
			name:        "read secret from path and strip new line, tabs and whitespaces",
			secretValue: "\n\t mysecretvalue\t \n",
			expected:    "mysecretvalue",
			wantErr:     false,
		},
		{
			name:        "read secret from path and strip whitespaces",
			secretValue: "    mysecretvalue   ",
			expected:    "mysecretvalue",
			wantErr:     false,
		},
		{
			name:                "return empty string and err: wrong file path",
			secretValue:         "mysecretvalue",
			expected:            "",
			wantErr:             true,
			notExistingFilePath: "/my/not/existing/file",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			tempFile, err := os.CreateTemp(os.TempDir(), "garmpassword")
			if err != nil {
				t.Fatalf("Could not create temp file for garmpassword: %v", err)
			}
			defer os.Remove(tempFile.Name())

			_, err = tempFile.Write([]byte(tt.secretValue))
			if err != nil {
				t.Fatalf("Failed to write to temp file: %v", err)
			}
			tempFile.Close()

			filePathToCheck := tempFile.Name()
			if tt.notExistingFilePath != "" {
				filePathToCheck = tt.notExistingFilePath
			}

			secretValue, err := ReadFromFile(filePathToCheck)
			if (err != nil) && tt.wantErr {
				t.Logf("secret.ReadFromFile() \n got error = %v \n wantErr = %v", err, tt.wantErr)
				return
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("secret.ReadFromFile() \n got error = %v \n wantErr = %v", err, tt.wantErr)
				return
			}

			if secretValue != tt.expected {
				t.Errorf("secret.ReadFromFile() \n got =  %q \n want = %q", secretValue, tt.expected)
			}
		})
	}
}
