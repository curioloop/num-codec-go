package numcodec_test

import (
	"bufio"
	"compress/gzip"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/curioloop/num-codec-go"
)

// kline mirrors TestDataSample.KLine (fields[6] carries volume; the Java
// loader accidentally reads fields[1] there — we use the correct index).
type kline struct {
	time, volume           int64
	open, close, high, low float32
	amount                 float64
}

// trade mirrors TestDataSample.Trade.
type trade struct {
	time  int64
	price float32
	size  int32
}

const klineBytes = 8*2 + 4*4 + 8 // 32 bytes/row raw
const tradeBytes = 8 + 4 + 4     // 16 bytes/row raw

func loadKline(t *testing.T, name string) map[string][]kline {
	t.Helper()
	f, err := os.Open(filepath.Join("testdata", "sample", name))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("gzip: %v", err)
	}
	defer gz.Close()

	out := map[string][]kline{}
	sc := bufio.NewScanner(gz)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		fields := strings.Split(sc.Text(), ",")
		if len(fields) < 8 {
			continue
		}
		time, _ := strconv.ParseInt(fields[1], 10, 64)
		open, _ := strconv.ParseFloat(fields[2], 32)
		closev, _ := strconv.ParseFloat(fields[3], 32)
		high, _ := strconv.ParseFloat(fields[4], 32)
		low, _ := strconv.ParseFloat(fields[5], 32)
		volume, _ := strconv.ParseInt(fields[6], 10, 64)
		amount, _ := strconv.ParseFloat(fields[7], 64)
		out[fields[0]] = append(out[fields[0]], kline{
			time: time, volume: volume,
			open: float32(open), close: float32(closev),
			high: float32(high), low: float32(low),
			amount: amount,
		})
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan: %v", err)
	}
	return out
}

func loadTrade(t *testing.T, name string) map[string][]trade {
	t.Helper()
	f, err := os.Open(filepath.Join("testdata", "sample", name))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatalf("gzip: %v", err)
	}
	defer gz.Close()

	out := map[string][]trade{}
	sc := bufio.NewScanner(gz)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for sc.Scan() {
		fields := strings.Split(sc.Text(), ",")
		if len(fields) < 4 {
			continue
		}
		time, _ := strconv.ParseInt(fields[1], 10, 64)
		price, _ := strconv.ParseFloat(fields[2], 32)
		size, _ := strconv.ParseInt(fields[3], 10, 32)
		out[fields[0]] = append(out[fields[0]], trade{
			time: time, price: float32(price), size: int32(size),
		})
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("scan: %v", err)
	}
	return out
}

// compressKline runs each column through the appropriate helper and
// returns totals. Also asserts round-trip correctness.
func compressKline(t *testing.T, series map[string][]kline) (raw, compressed int) {
	t.Helper()
	dst := make([]byte, 0, 1024)
	for _, rows := range series {
		if len(rows) < 2 {
			continue
		}
		times := make([]int64, len(rows))
		volumes := make([]int64, len(rows))
		opens := make([]float32, len(rows))
		closes := make([]float32, len(rows))
		highs := make([]float32, len(rows))
		lows := make([]float32, len(rows))
		amounts := make([]float64, len(rows))
		for i, r := range rows {
			times[i] = r.time
			volumes[i] = r.volume
			opens[i] = r.open
			closes[i] = r.close
			highs[i] = r.high
			lows[i] = r.low
			amounts[i] = r.amount
		}
		raw += klineBytes * len(rows)
		compressed += encAndCheckDelta2(t, times, dst)
		compressed += encAndCheckLongs(t, volumes, dst)
		compressed += encAndCheckFloats(t, opens, dst)
		compressed += encAndCheckFloats(t, closes, dst)
		compressed += encAndCheckFloats(t, highs, dst)
		compressed += encAndCheckFloats(t, lows, dst)
		compressed += encAndCheckDoubles(t, amounts, dst)
	}
	return
}

func compressTrade(t *testing.T, series map[string][]trade) (raw, compressed int) {
	t.Helper()
	dst := make([]byte, 0, 1024)
	for _, rows := range series {
		if len(rows) < 2 {
			continue
		}
		times := make([]int64, len(rows))
		prices := make([]float32, len(rows))
		sizes := make([]int32, len(rows))
		for i, r := range rows {
			times[i] = r.time
			prices[i] = r.price
			sizes[i] = r.size
		}
		raw += tradeBytes * len(rows)
		compressed += encAndCheckDelta2(t, times, dst)
		compressed += encAndCheckFloats(t, prices, dst)
		compressed += encAndCheckInts(t, sizes, dst)
	}
	return
}

func encAndCheckLongs(t *testing.T, vs []int64, dst []byte) int {
	t.Helper()
	data, flags, err := numcodec.AppendEncodeInt64s(dst[:0], vs)
	if err != nil {
		t.Fatalf("encode longs: %v", err)
	}
	out, err := numcodec.AppendDecodeInt64s(nil, data, flags)
	if err != nil {
		t.Fatalf("decode longs: %v", err)
	}
	for i := range vs {
		if out[i] != vs[i] {
			t.Fatalf("long mismatch at %d", i)
		}
	}
	return len(data)
}

func encAndCheckInts(t *testing.T, vs []int32, dst []byte) int {
	t.Helper()
	data, flags, err := numcodec.AppendEncodeInt32s(dst[:0], vs)
	if err != nil {
		t.Fatalf("encode ints: %v", err)
	}
	out, err := numcodec.AppendDecodeInt32s(nil, data, flags)
	if err != nil {
		t.Fatalf("decode ints: %v", err)
	}
	for i := range vs {
		if out[i] != vs[i] {
			t.Fatalf("int mismatch at %d", i)
		}
	}
	return len(data)
}

func encAndCheckDelta2(t *testing.T, vs []int64, dst []byte) int {
	t.Helper()
	data, flags, err := numcodec.AppendEncodeDelta2(dst[:0], vs)
	if err != nil {
		t.Fatalf("encode delta2: %v", err)
	}
	out, err := numcodec.AppendDecodeDelta2(nil, data, flags)
	if err != nil {
		t.Fatalf("decode delta2: %v", err)
	}
	for i := range vs {
		if out[i] != vs[i] {
			t.Fatalf("delta2 mismatch at %d", i)
		}
	}
	return len(data)
}

func encAndCheckFloats(t *testing.T, vs []float32, dst []byte) int {
	t.Helper()
	data, flags, err := numcodec.AppendEncodeFloat32s(dst[:0], vs)
	if err != nil {
		t.Fatalf("encode floats: %v", err)
	}
	out, err := numcodec.AppendDecodeFloat32s(nil, data, flags)
	if err != nil {
		t.Fatalf("decode floats: %v", err)
	}
	for i := range vs {
		if out[i] != vs[i] {
			t.Fatalf("float mismatch at %d", i)
		}
	}
	return len(data)
}

func encAndCheckDoubles(t *testing.T, vs []float64, dst []byte) int {
	t.Helper()
	data, flags, err := numcodec.AppendEncodeFloat64s(dst[:0], vs)
	if err != nil {
		t.Fatalf("encode doubles: %v", err)
	}
	out, err := numcodec.AppendDecodeFloat64s(nil, data, flags)
	if err != nil {
		t.Fatalf("decode doubles: %v", err)
	}
	for i := range vs {
		if out[i] != vs[i] {
			t.Fatalf("double mismatch at %d", i)
		}
	}
	return len(data)
}

// TestCompressRateOnSamples encodes every column of each sample dataset
// and logs the achieved ratio. Not an assertion — smoke check that the
// algorithms deliver real compression on real time-series data.
func TestCompressRateOnSamples(t *testing.T) {
	cases := []struct {
		name  string
		kline string
		trade string
	}{
		{"NBBO", "NBBO_2023-12-15_kline.gz", "NBBO_2023-12-15_trade.gz"},
		{"SEHK", "SEHK_2023-12-20_kline.gz", "SEHK_2023-12-20_trade.gz"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			kraw, kcomp := compressKline(t, loadKline(t, c.kline))
			traw, tcomp := compressTrade(t, loadTrade(t, c.trade))
			total, comp := kraw+traw, kcomp+tcomp
			ratio := 100.0 * float64(comp) / float64(total)
			t.Logf("kline: raw=%d compressed=%d (%.1f%%)", kraw, kcomp, 100.0*float64(kcomp)/float64(kraw))
			t.Logf("trade: raw=%d compressed=%d (%.1f%%)", traw, tcomp, 100.0*float64(tcomp)/float64(traw))
			t.Logf("total: raw=%d compressed=%d (%.1f%%)", total, comp, ratio)
			if ratio >= 100 {
				t.Fatalf("compressed larger than raw (%.1f%%)", ratio)
			}
		})
	}
}
