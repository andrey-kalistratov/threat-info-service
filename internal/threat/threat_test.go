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
		wantErr error
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
			wantErr: threat.ErrInvalidMD5Length,
		},
		{
			name:    "Длиннее 32 символов",
			input:   canonical + "0",
			wantErr: threat.ErrInvalidMD5Length,
		},
		{
			name:    "Недопустимый hex-символ",
			input:   "g" + canonical[1:],
			wantErr: threat.ErrInvalidMD5Characters,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange.
			input := tt.input

			// Act.
			got, err := threat.ParseMD5(input)

			// Assert.
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				return
			}

			require.NoError(t, err)

			require.Equal(t, tt.want, got)
		})
	}
}
