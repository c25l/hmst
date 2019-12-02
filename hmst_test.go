package main

import (
	"fmt"
	"math/rand"
	"testing"
)

func TestHMST(t *testing.T) {
	t.Run("Prime generation", func(t *testing.T) {
		if 1031 != PrimeSieve(1000, 5)[4] {
			t.Fail()
		}

	})
	hist := NewHMST(10, 1000, []string{"group", "instance", "job", "service"})
	hist.Add(map[string]string{"group": "a", "instance": "a", "job": "c", "service": "d"}, 1, 10.4, 1)
	hist.Add(map[string]string{"group": "a", "instance": "a", "job": "c", "service": "d"}, 1, 45.4, 1)
	hist.Add(map[string]string{"group": "a", "instance": "a", "job": "c", "service": "d"}, 2, 12.4, 2)
	hist.Add(map[string]string{"group": "a", "instance": "b", "job": "d", "service": "q"}, 1, 10.4, 1)
	serialized, _ := hist.Serialize()
	hist2, _ := Deserialize(serialized)
	hist3 := NewHMST(10, 1000, []string{"group", "job", "instance", "service", "foo"})
	t.Run("Projection roundtrip equality", func(t *testing.T) {
		x := hist.project(153)
		y := hist.project(float64(x))
		if y != x {
			t.Log(fmt.Sprintf("first %v, second %v", x, y))
			t.Fail()
		}
	})
	// Todo actual testing!
	t.Run("(De)serialize compatibility", func(t *testing.T) {
		if !Compatible(hist, hist2) {
			t.Fail()
		}
	})
	t.Run("(De)serialize count equality", func(t *testing.T) {
		if hist.TotalCount() != hist2.TotalCount() {
			t.Fail()
		}
	})
	t.Run("Self-compatibility", func(t *testing.T) {
		if !Compatible(hist, hist) {
			t.Fail()
		}
	})
	t.Run("Incompatibility", func(t *testing.T) {
		if Compatible(hist2, hist3) {
			t.Fail()
		}
	})
	t.Run("basic adding", func(t *testing.T) {
		before := hist.TotalCount()
		hist.Add(map[string]string{"group": "a"}, 1, 1.0, 1)
		after := hist.TotalCount()
		if after != before+1 {
			t.Fail()
		}
	})
	t.Run("copying", func(t *testing.T) {
		hist4 := Copy(hist)
		if hist.TotalCount() != hist4.TotalCount() {
			t.Log(hist, hist4)
			t.Fail()
		}
	})
	t.Run("Basic sketch", func(t *testing.T) {
		x := hist2.Sketch(map[string]string{"group": "a", "instance": "a", "job": "c", "service": "d"}, 1)
		if x[10] != 1 {
			t.Log(x[10], x, hist2)
			t.Fail()
		}
	})
	t.Run("Null sketch", func(t *testing.T) {
		x := hist2.Sketch(map[string]string{"group": "a", "instance": "a", "job": "c", "service": "e"}, 1)
		total := 0
		for _, val := range x {
			total += val
		}
		if 0 != total {
			t.Log(total, x)
			t.Fail()
		}
	})
	t.Run("Counting", func(t *testing.T) {
		temp := hist2.Count(map[string]string{"group": "a", "instance": "a", "job": "c", "service": "d"}, 2)
		if 2 != temp {
			t.Log(2, "!=", temp)
			t.Fail()
		}
	})
	t.Run("self-combination total count validation", func(t *testing.T) {
		if val, _ := Combine(hist, hist); 2*hist.TotalCount() != (*val).TotalCount() {
			t.Log(hist.TotalCount(), (*val).TotalCount())
			t.Fail()
		}
	})
	t.Run("Differencing to empty", func(t *testing.T) {
		if val, _ := HistDiff(hist, hist); 0 != (*val).TotalCount() {
			t.Log((*val).TotalCount())
			t.Fail()
		}
	})
	t.Run("Total Count", func(t *testing.T) {
		if 6 != hist.TotalCount() {
			t.Fail()
		}
	})
	t.Run("KS-Test", func(t *testing.T) {
		x := hist2.Sketch(map[string]string{"group": "a", "instance": "a", "job": "c", "service": "d"}, 1)
		if 0.0 != KSTest(x, x) {
			t.Fail()
		}
		y := hist2.Sketch(map[string]string{"group": "a", "instance": "a", "job": "c", "service": "e"}, 1)
		if 0.01 < KSTest(x, y) {
			t.Fail()
		}
	})
	t.Run("Quantiles", func(t *testing.T) {
		x := hist2.Sketch(map[string]string{"group": "a", "instance": "a", "job": "c", "service": "d"}, 1)
		q := Quantile(x, []float64{0.1, 0.5, 0.9})
		t.Log(x, q)
		if q[0] == q[1] {
			t.Fail()
		}
		if q[1] != q[2] {
			t.Fail()
		}
	})
}

var (
	h = NewHMST(10, 1000, []string{"group", "job", "instance", "service", "foo"})
)

func BenchmarkHMST(b *testing.B) {
	for ii := 0; ii < b.N; ii++ {
		gp := fmt.Sprintf("%v", rand.Intn(10))
		in := fmt.Sprintf("%v", rand.Intn(8))
		job := fmt.Sprintf("%v", rand.Intn(6))
		serv := fmt.Sprintf("%v", rand.Intn(4))
		time := rand.Intn(1000)
		value := rand.NormFloat64()*1000 + 100
		count := rand.Intn(5) + 1
		h.Add(map[string]string{"group": gp, "instance": in, "job": job, "service": serv}, time, value, count)
	}

}

func BenchmarkSketch(b *testing.B) {
	for ii := 0; ii < b.N; ii++ {
		gp := fmt.Sprintf("%v", rand.Intn(10))
		in := fmt.Sprintf("%v", rand.Intn(8))
		job := fmt.Sprintf("%v", rand.Intn(6))
		serv := fmt.Sprintf("%v", rand.Intn(4))
		time := rand.Intn(1000)
		h.Sketch(map[string]string{"group": gp, "instance": in, "job": job, "service": serv}, time)
	}
}
