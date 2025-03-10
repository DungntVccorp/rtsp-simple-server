package core

import (
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/aler9/gortsplib"
	"github.com/stretchr/testify/require"
)

func TestMetrics(t *testing.T) {
	serverCertFpath, err := writeTempFile(serverCert)
	require.NoError(t, err)
	defer os.Remove(serverCertFpath)

	serverKeyFpath, err := writeTempFile(serverKey)
	require.NoError(t, err)
	defer os.Remove(serverKeyFpath)

	p, ok := newInstance("metrics: yes\n" +
		"encryption: optional\n" +
		"serverCert: " + serverCertFpath + "\n" +
		"serverKey: " + serverKeyFpath + "\n")
	require.Equal(t, true, ok)
	defer p.close()

	track, err := gortsplib.NewTrackH264(96,
		&gortsplib.TrackConfigH264{SPS: []byte{0x01, 0x02, 0x03, 0x04}, PPS: []byte{0x01, 0x02, 0x03, 0x04}})
	require.NoError(t, err)

	source, err := gortsplib.DialPublish("rtsp://localhost:8554/mypath",
		gortsplib.Tracks{track})
	require.NoError(t, err)
	defer source.Close()

	cnt1, err := newContainer("ffmpeg", "source", []string{
		"-re",
		"-stream_loop", "-1",
		"-i", "emptyvideo.mkv",
		"-c", "copy",
		"-f", "flv",
		"rtmp://localhost:1935/test1/test2",
	})
	require.NoError(t, err)
	defer cnt1.close()

	req, err := http.NewRequest(http.MethodGet, "http://localhost:9998/metrics", nil)
	require.NoError(t, err)

	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()
	require.Equal(t, http.StatusOK, res.StatusCode)

	bo, err := ioutil.ReadAll(res.Body)
	require.NoError(t, err)

	vals := make(map[string]string)
	lines := strings.Split(string(bo), "\n")
	for _, l := range lines[:len(lines)-1] {
		fields := strings.Split(l, " ")
		vals[fields[0]] = fields[1]
	}

	require.Equal(t, map[string]string{
		"paths{state=\"notReady\"}":         "0",
		"paths{state=\"ready\"}":            "2",
		"rtmp_conns{state=\"idle\"}":        "0",
		"rtmp_conns{state=\"publish\"}":     "1",
		"rtmp_conns{state=\"read\"}":        "0",
		"rtsp_sessions{state=\"idle\"}":     "0",
		"rtsp_sessions{state=\"publish\"}":  "1",
		"rtsp_sessions{state=\"read\"}":     "0",
		"rtsps_sessions{state=\"idle\"}":    "0",
		"rtsps_sessions{state=\"publish\"}": "0",
		"rtsps_sessions{state=\"read\"}":    "0",
	}, vals)
}
