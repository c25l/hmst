package main

import (
	"bytes"
	"encoding/gob"
	"errors"
	"log"
	"math"
	"sort"
)

var (
	sieve map[int]bool
)

// HMST is the core structure representing the structure
// this version uses maps, it grows larger over time but
// has better gurantees on error rate.xs
type HMST struct {
	Resolution float64
	TimeModuli map[string]int
	Global     map[int]map[int]int
	MaxTime    int
	Registers  map[string]map[int]map[string]map[int]int
}

// PrimeSieve gets the next needed number of primes after minval.
func PrimeSieve(minval int, needed int) []int {
	if sieve == nil {
		sieve = make(map[int]bool)
	}
	for ii := 2; ii < (needed+1)*minval; ii++ {
		if _, ok := sieve[ii]; ok {
			continue
		}
		for kk := ii * ii; kk < (needed+1)*minval; kk += ii {
			sieve[kk] = true
		}
	}
	out := make([]int, needed)
	remaining := needed
	for kk := minval; kk < (needed+1)*minval && remaining > 0; kk++ {
		if _, ok := sieve[kk]; !ok {
			out[needed-remaining] = kk
			remaining--
		}
	}
	return out
}

// NewHMST takes a hist and resets it to the default values.
func NewHMST(resolution float64, maxtime int, mandateVariables []string) *HMST {
	var h HMST
	if resolution == 0 || maxtime < 1 {
		return nil
	}
	h.Resolution = resolution
	h.TimeModuli = make(map[string]int)
	// Using prime factors is stronger than what we need here, encoding the times is
	// accomplished through the chinese remainder theorem. No, really. It works. You
	// can find the exact times from the factor times using Bezout's identity. It isn't
	// necessary for any functionality.
	h.MaxTime = maxtime
	// The number of registers necessary is a function of many things, but ultimately the trade
	// is between space and error rate. Anyhow, you can make them smaller than this.
	h.Registers = make(map[string]map[int]map[string]map[int]int)
	h.Global = make(map[int]map[int]int)
	h.Global[0] = make(map[int]int)
	for _, xx := range mandateVariables {
		(&h).AddVariable(xx)
	}
	return &h
}

// NextModulus finds the next relevant prime for use with the
// timing registers.
func (h *HMST) NextModulus() int {
	current := int(math.Sqrt(float64((*h).MaxTime)))
	for _, val := range (*h).TimeModuli {
		if val > current {
			current = val
		}
	}
	return PrimeSieve(current+1, 1)[0]
}

// AddVariable adds a variable to a histogram with null information or does nothing
// If there is a preferred set of variables, this can be used to guarantee compatibility
// As the incompatibilities due to different modulus are due to orderings.
func (h *HMST) AddVariable(name string) {
	_, ok := (*h).TimeModuli[name]
	if !ok {
		(*h).TimeModuli[name] = (*h).NextModulus()
	}

	_, ok = (*h).Registers[name]
	if !ok {
		(*h).Registers[name] = make(map[int]map[string]map[int]int)
	}
}

// Add adds a value to a histogram
func (h *HMST) Add(locations map[string]string, time int, value float64, count int) {
	t := time % (*h).MaxTime
	bin := (*h).project(value)
	_, ok := (*h).Global[t]
	if !ok {
		(*h).Global[t] = make(map[int]int)
	}
	(*h).Global[t][bin] += count
	for key, val := range locations {
		h.AddVariable(key)
		t = time % (*h).TimeModuli[key]
		_, ok = (*h).Registers[key][t]
		if !ok {
			(*h).Registers[key][t] = make(map[string]map[int]int)
		}
		_, ok = (*h).Registers[key][t][val]
		if !ok {
			(*h).Registers[key][t][val] = make(map[int]int)
		}
		(*h).Registers[key][t][val][bin] += count
	}
}

// Get gets a thing from the sketch. It's nice to have the getter here
// because of the relative unsafety of querying empty maps.
func (h *HMST) Get(key string, time int, val string, bin int) int {
	return 0
}

// Sketch gets a point histogram for a given time and key-value map
// TODO this function is flaky. wtf? Tests irreglar at best.
func (h *HMST) Sketch(kvs map[string]string, time int) map[int]int {
	t := time % (*h).MaxTime
	output := make(map[int]int, len((*h).Global[0]))
	for bin, count := range (*h).Global[t] {
		output[bin] = count
	}
	for key, val := range kvs {
		t = time % (*h).TimeModuli[key]

		_, ok := (*h).Registers[key]
		if ok {
			_, ok = (*h).Registers[key][t]
			if ok {
				use, ok := (*h).Registers[key][t][val]
				if ok {
					visited := make(map[int]bool)
					for bin, count := range use {
						visited[bin] = true
						if output[bin] > count {
							output[bin] = count
						}
					}
					for bin, _ := range output {
						if !visited[bin] {
							output[bin] = 0
						}
					}
				} else {
					return make(map[int]int)
				}
			} else {
				return make(map[int]int)
			}
		} else {
			return make(map[int]int)
		}
	}
	return output
}

// Count returns the count of all items at a location and time
func (h *HMST) Count(kvs map[string]string, time int) int {
	hist := h.Sketch(kvs, time)
	total := 0
	for _, val := range hist {
		total += val
	}
	return total
}

// TotalCount returns the count of all locations at all times
func (h *HMST) TotalCount() int {
	counts := 0
	for _, raw := range (*h).Global {
		for _, val := range raw {
			counts += val
		}
	}
	return counts
}

// Compatible checks to see if two hmsts have identical parameters.
// It's possible for them not to if they should, in particular if the times
// don't match for various variables because of a difference of insert order.
// Kind of a bug...
func Compatible(h1 *HMST, h2 *HMST) bool {
	compat := (h1.Resolution == h2.Resolution)
	for key, val := range h1.TimeModuli {
		test, ok := h2.TimeModuli[key]
		if ok {
			compat = compat && (val == test)
		}
	}
	for key, val := range h2.TimeModuli {
		test, ok := h1.TimeModuli[key]
		if ok {
			compat = compat && (val == test)
		}
	}
	return compat
}

// Copy duplicates the other hmst into this one,
// useful because I've blanked on how to do this more quickly
// and everything being refrence-duplicated sucks.
func Copy(o *HMST) *HMST {
	h := NewHMST((*o).Resolution, 1, nil)
	(*h).MaxTime = (*o).MaxTime
	for key, val := range (*o).TimeModuli {
		(*h).TimeModuli[key] = val
	}
	for vbl, val := range (*o).Registers {
		for time, val2 := range val {
			for key, val3 := range val2 {
				for bin, ct := range val3 {
					(*h).Add(map[string]string{vbl: key}, time, float64(bin), ct)
				}
			}
		}
	}
	for time, val := range (*o).Global {
		(*h).Global[time] = make(map[int]int)
		for bin, ct := range val {
			(*h).Global[time][bin] = ct
		}
	}

	return h
}

// Combine takes two compatible hmsts and sums them pointwise
func Combine(h1 *HMST, h2 *HMST) (*HMST, error) {
	if !Compatible(h1, h2) {
		return nil, errors.New("Histograms not compatible")
	}
	out := Copy(h1)
	for vbl, val := range (*h2).Registers {
		for time, val2 := range val {
			for key, val3 := range val2 {
				for bin, ct := range val3 {
					out.Add(map[string]string{vbl: key}, time, float64(bin), ct)
				}
			}
		}
	}
	for time, val := range (*h1).Global {
		out.Global[time] = make(map[int]int)
		for bin, ct := range val {
			out.Global[time][bin] += ct
		}
	}
	for time, val := range (*h2).Global {
		for bin, ct := range val {
			out.Global[time][bin] += ct

		}
	}

	return out, nil
}

// HistDiff subtracts a hist from another if possible
// This is really more of a right-cancellation than a subtraction.
func HistDiff(h1 *HMST, h2 *HMST) (*HMST, error) {
	out := Copy(h1)
	for vbl, val := range (*h2).Registers {
		for time, val2 := range val {
			for key, val3 := range val2 {
				for bin, cnt := range val3 {
					_, ok := (*out).Registers[vbl][time][key][bin]
					if !ok {
						log.Println("value not found")
					}
					out.Add(map[string]string{vbl: key}, time, float64(bin), 0)
					temp := (*out).Registers[vbl][time][key][bin] - cnt

					if temp < 0 {
						return nil, errors.New("Negative values not supported")
					}
					out.Registers[vbl][time][key][bin] = temp

				}
			}

		}
	}
	out.Global = make(map[int]map[int]int)
	for time, hist := range (*h2).Global {
		out.Global[time] = make(map[int]int)
		for bin, ct := range hist {
			out.Global[time][bin] = ct - (*h1).Global[time][bin]
		}
	}
	return out, nil
}

// CDF turns a sketch into a cumulogram
func CDF(sketch map[int]int) map[int]float64 {
	known := make([]int, len(sketch))
	loc := 0
	for bin, _ := range sketch {
		known[loc] = bin
		loc++
	}
	sort.Ints(known)
	out := make(map[int]float64)
	running := 0.0
	for _, bin := range known {
		if sketch[bin] > 0 {
			running += float64(sketch[bin])
			out[bin] = running

		}
	}
	for bin, cnt := range out {
		out[bin] = cnt / running
	}
	return out
}

// ICDF turns a sketch into an inverse cumulogram.
// As a convention, each bucket corresponds to quantile*1000
func ICDF(sketch map[int]int) []int {
	out := make([]int, 1000)
	proto := CDF(sketch)
	reverse := make(map[float64]int)
	known := make([]float64, len(proto))
	loc := 0
	for bin, quant := range proto {
		reverse[quant] = bin
		known[loc] = quant
		loc++
	}
	sort.Float64s(known)
	previous := 0
	prevQuant := 0
	for _, quantile := range known {
		for i := previous; i < int(quantile*1000); i++ {
			out[i] = reverse[quantile]
		}
		previous = int(math.Max(float64(previous), quantile*1000))
		prevQuant = reverse[quantile]
	}
	if previous < len(out) {
		for i := previous + 1; i < len(out); i++ {
			out[i] = prevQuant
		}
	}
	return out
}

// Quantile reports on quantiles from a cdf
func Quantile(sketch map[int]int, quantiles []float64) []int {
	icdf := ICDF(sketch)
	out := make([]int, len(quantiles))
	for ii, quant := range quantiles {
		out[ii] = icdf[int(quant*1000)]
	}
	return out
}

// KSTest Performs a ks-test between two cdfs
func KSTest(sketch1 map[int]int, sketch2 map[int]int) float64 {
	cdf1 := CDF(sketch1)
	cdf2 := CDF(sketch2)
	ks := 0.0
	for ii, val := range cdf1 {
		ks += math.Abs(val - cdf2[ii])
	}
	return 0.0
}

// Serialize will render the hmst as bytes.
func (h *HMST) Serialize() ([]byte, error) {
	var outbytes bytes.Buffer
	enc := gob.NewEncoder(&outbytes)
	err := enc.Encode(*h)
	if err != nil {
		return nil, err
	}
	return outbytes.Bytes(), nil
}

// Deserialize is the inverse of serialize.
func Deserialize(input []byte) (*HMST, error) {
	var inbytes bytes.Buffer
	var h HMST
	inbytes.Write(input)
	dec := gob.NewDecoder(&inbytes)
	err := dec.Decode(&h)
	if err != nil {
		return nil, err
	}
	return &h, nil
}

// project figures out which bin a value belongs in.
// these bins are all linear for now. Could do exponential in the future.
func (h *HMST) project(inputval float64) int {
	return int(inputval - math.Mod(inputval, (*h).Resolution))
}
