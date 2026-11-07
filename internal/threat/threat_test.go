package threat_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/andrey-kalistratov/threat-info-service/internal/threat"
)

func TestParseMD5(t *testing.T) {
	const canonical = "d41d8cd98f00b204e9800998ecf8427e"

	tests := []struct {
		name    string
		input   string
		want    threat.MD5
		wantErr bool
	}{
		{
			name:  "Каноническое значение сохраняется",
			input: canonical,
			want:  threat.MD5(canonical),
		},
		{
			name:  "Регистр нормализуется",
			input: "D41d8Cd98f00B204e9800998eCf8427E",
			want:  threat.MD5(canonical),
		},
		{
			name:    "Короче 32 символов",
			input:   canonical[:31],
			wantErr: true,
		},
		{
			name:    "Длиннее 32 символов",
			input:   canonical + "0",
			wantErr: true,
		},
		{
			name:    "Недопустимый hex-символ",
			input:   "g" + canonical[1:],
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := threat.ParseMD5(tt.input)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, tt.want, got)
		})
	}
}
