package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSampleRequest(t *testing.T) {
	var tests = []struct {
		req                    *NewSampleRequest
		expectedLocalFilename  string
		expectedStreamFilename string
		expectedWAVFilename    string
	}{
		{
			req: &NewSampleRequest{
				Data:     []byte("jaskjhfashdfjhsaflsafjhdsjakfhsajfklsajfkj3kjrqjrfkaskfhsadlfkjsa"),
				Mimetype: "audio/wav",
				Pubkey:   "000005f8bc46b589ace6db0c6f7cf8b1b88dc55595886976e53bbd91423e267e",
				Sum:      "7866659283cb9ba9fa735818a0fa1e61fd1089695d998ae1d575c7163d1c8b1b",
			},
			expectedLocalFilename:  "7866659283cb9ba9fa735818a0fa1e61fd1089695d998ae1d575c7163d1c8b1b.wav",
			expectedStreamFilename: "7866659283cb9ba9fa735818a0fa1e61fd1089695d998ae1d575c7163d1c8b1b",
			expectedWAVFilename:    "7866659283cb9ba9fa735818a0fa1e61fd1089695d998ae1d575c7163d1c8b1b.wav",
		},
		{
			req: &NewSampleRequest{
				Data:     []byte("jaskjhfashdfjhsaflsafjhdsjakfhsajfklsajfkj3kjrqjrfkaskfhsadlfkjsa"),
				Mimetype: "audio/mp3",
				Pubkey:   "000005f8bc46b589ace6db0c6f7cf8b1b88dc55595886976e53bbd91423e267e",
				Sum:      "7866659283cb9ba9fa735818a0fa1e61fd1089695d998ae1d575c7163d1c8b1b",
			},
			expectedLocalFilename:  "7866659283cb9ba9fa735818a0fa1e61fd1089695d998ae1d575c7163d1c8b1b.mp3",
			expectedStreamFilename: "7866659283cb9ba9fa735818a0fa1e61fd1089695d998ae1d575c7163d1c8b1b",
			expectedWAVFilename:    "7866659283cb9ba9fa735818a0fa1e61fd1089695d998ae1d575c7163d1c8b1b.wav",
		},
		{
			req: &NewSampleRequest{
				Data:     []byte("jaskjhfashdfjhsaflsafjhdsjakfhsajfklsajfkj3kjrqjrfkaskfhsadlfkjsa"),
				Mimetype: "audio/m4a",
				Pubkey:   "000005f8bc46b589ace6db0c6f7cf8b1b88dc55595886976e53bbd91423e267e",
				Sum:      "7866659283cb9ba9fa735818a0fa1e61fd1089695d998ae1d575c7163d1c8b1b",
			},
			expectedLocalFilename:  "7866659283cb9ba9fa735818a0fa1e61fd1089695d998ae1d575c7163d1c8b1b.m4a",
			expectedStreamFilename: "7866659283cb9ba9fa735818a0fa1e61fd1089695d998ae1d575c7163d1c8b1b",
			expectedWAVFilename:    "7866659283cb9ba9fa735818a0fa1e61fd1089695d998ae1d575c7163d1c8b1b.wav",
		},
		{
			req: &NewSampleRequest{
				Data:     []byte("jaskjhfashdfjhsaflsafjhdsjakfhsajfklsajfkj3kjrqjrfkaskfhsadlfkjsa"),
				Mimetype: "audio/aiff",
				Pubkey:   "000005f8bc46b589ace6db0c6f7cf8b1b88dc55595886976e53bbd91423e267e",
				Sum:      "7866659283cb9ba9fa735818a0fa1e61fd1089695d998ae1d575c7163d1c8b1b",
			},
			expectedLocalFilename:  "7866659283cb9ba9fa735818a0fa1e61fd1089695d998ae1d575c7163d1c8b1b.aif",
			expectedStreamFilename: "7866659283cb9ba9fa735818a0fa1e61fd1089695d998ae1d575c7163d1c8b1b",
			expectedWAVFilename:    "7866659283cb9ba9fa735818a0fa1e61fd1089695d998ae1d575c7163d1c8b1b.wav",
		},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.expectedLocalFilename, localFilename(tt.req.Sum, tt.req.Mimetype))
		assert.Equal(t, tt.expectedStreamFilename, streamFilename(tt.req.Sum))
		assert.Equal(t, tt.expectedWAVFilename, wavFilename(tt.req.Sum))
	}
}
