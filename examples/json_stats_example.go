package main

import (
	"encoding/json"
	"fmt"
	"mi0772/podcache/cache"
	"mi0772/podcache/logging"
)

// Esempio di test per il logging JSON delle statistiche
func testJSONStats() {
	logger := logging.NewDebugLogger()
	
	// Crea una cache di test
	testCache, err := cache.NewPodCache(2, 1024*1024, logger)
	if err != nil {
		fmt.Printf("Error creating cache: %v\n", err)
		return
	}
	
	// Aggiungi alcuni dati per avere statistiche interessanti
	testCache.Put("key1", []byte("value1"))
	testCache.Put("key2", []byte("value2"))
	testCache.Get("key1") // hit
	testCache.Get("key3") // miss
	
	// Ottieni statistiche
	stats := testCache.Stats()
	
	// Mostra il JSON risultante
	statsJSON, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		fmt.Printf("Error marshaling stats: %v\n", err)
		return
	}
	
	fmt.Println("Esempio di output JSON delle statistiche:")
	fmt.Println("==========================================")
	fmt.Println(string(statsJSON))
	fmt.Println("==========================================")
	
	// Test del logging effettivo
	fmt.Println("\nOutput del logging JSON:")
	fmt.Println("========================")
	logging.LogStatsAsJSON(logger, stats)
}
