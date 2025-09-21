package main

import (
	"testing"
)

func TestFrequencyToBand(t *testing.T) {
	// Test cases: frequency in Hz -> expected band
	testCases := map[float64]string{
		// HF bands
		28074000:  "10m",     // 28.074 MHz
		14200000:  "20m",     // 14.2 MHz
		7100000:   "40m",     // 7.1 MHz
		3700000:   "80m",     // 3.7 MHz
		10136000:  "30m",     // 10.136 MHz
		21200000:  "15m",     // 21.2 MHz
		18100000:  "17m",     // 18.1 MHz
		1800000:   "160m",    // 1.8 MHz
		5400000:   "60m",     // 5.4 MHz
		24900000:  "12m",     // 24.9 MHz

		// VHF/UHF bands
		144100000: "2m",      // 144.1 MHz
		430000000: "70cm",    // 430 MHz
		52000000:  "6m",      // 52 MHz
		223000000: "1.25m",   // 223 MHz

		// Microwave bands
		1250000000: "23cm",   // 1250 MHz
		920000000:  "33cm",   // 920 MHz

		// LF bands
		136000:    "2200m",   // 136 kHz
		475000:    "630m",    // 475 kHz

		// Edge cases
		999999:    "unknown", // Not in any band
		100000000: "unknown", // Between bands
	}

	for freq, expected := range testCases {
		result := frequencyToBand(freq)
		if result != expected {
			t.Errorf("frequencyToBand(%.0f) = %s; want %s", freq, result, expected)
		}
	}
}

func TestBandPlanLoaded(t *testing.T) {
	if len(bandPlan) == 0 {
		t.Error("bandPlan is empty - band plan failed to load")
	}

	// Should have at least the major amateur radio bands
	expectedMinBands := 15
	if len(bandPlan) < expectedMinBands {
		t.Errorf("bandPlan has %d bands; want at least %d", len(bandPlan), expectedMinBands)
	}

	// Check that some key bands exist
	keyBands := map[string]bool{
		"10m": false,
		"20m": false,
		"40m": false,
		"80m": false,
	}

	for _, band := range bandPlan {
		if _, exists := keyBands[band.Name]; exists {
			keyBands[band.Name] = true
		}
	}

	for bandName, found := range keyBands {
		if !found {
			t.Errorf("Key band %s not found in band plan", bandName)
		}
	}
}

func TestBandPlanOrder(t *testing.T) {
	// Test that all bands have valid frequency ranges
	for _, band := range bandPlan {
		if band.StartMHz >= band.EndMHz {
			t.Errorf("Band %s has invalid frequency range: %.4f >= %.4f", band.Name, band.StartMHz, band.EndMHz)
		}
		if band.StartMHz <= 0 {
			t.Errorf("Band %s has invalid start frequency: %.4f", band.Name, band.StartMHz)
		}
		if band.Name == "" {
			t.Error("Band has empty name")
		}
	}
}