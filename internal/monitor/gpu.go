package monitor

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
)

const GPUCommand = "LC_ALL=C nvidia-smi --query-gpu=uuid,index,name,utilization.gpu,memory.used,memory.total,temperature.gpu --format=csv,noheader,nounits"

func metric(s string) (*float64, error) {
	s = strings.TrimSpace(s)
	switch strings.ToLower(s) {
	case "n/a", "[n/a]", "[not supported]", "not supported", "":
		return nil, nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(v) || math.IsInf(v, 0) || v < 0 {
		return nil, fmt.Errorf("无效数值 %q", s)
	}
	return &v, nil
}
func ParseGPUs(raw string) ([]GPU, error) {
	result := []GPU{}
	r := csv.NewReader(strings.NewReader(raw))
	r.TrimLeadingSpace = true
	r.FieldsPerRecord = 7
	seen := map[string]bool{}
	for {
		fields, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("GPU CSV 解析失败：%w", err)
		}
		for i := range fields {
			fields[i] = strings.TrimSpace(fields[i])
		}
		index, err := strconv.Atoi(fields[1])
		if err != nil || index < 0 || !strings.HasPrefix(fields[0], "GPU-") || fields[2] == "" || seen[fields[0]] {
			return nil, fmt.Errorf("GPU 标识或编号无效")
		}
		seen[fields[0]] = true
		vals := make([]*float64, 4)
		for i := range vals {
			vals[i], err = metric(fields[i+3])
			if err != nil {
				return nil, err
			}
		}
		if vals[0] != nil && *vals[0] > 100 {
			return nil, fmt.Errorf("GPU 利用率超出范围")
		}
		result = append(result, GPU{UUID: fields[0], Index: index, Name: fields[2], Util: vals[0], MemUsed: vals[1], MemTotal: vals[2], Temp: vals[3]})
	}
	return result, nil
}
