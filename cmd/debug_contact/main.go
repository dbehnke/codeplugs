package main

import (
	"codeplugs/database"
	"codeplugs/models"
	"flag"
	"fmt"
	"log"
)

func main() {
	dbPath := flag.String("db", "codeplugs.db", "Path to SQLite database")
	callsign := flag.String("callsign", "KF8S", "Callsign to query")
	flag.Parse()

	database.Connect(*dbPath)

	var contact models.DigitalContact
	err := database.DB.Where("callsign = ?", *callsign).First(&contact).Error
	if err != nil {
		log.Fatalf("Contact not found: %v", err)
	}

	fmt.Printf("ID: %d\n", contact.ID)
	fmt.Printf("DMR ID: %d\n", contact.DMRID)
	fmt.Printf("Callsign: %s\n", contact.Callsign)
	fmt.Printf("Name: '%s'\n", contact.Name)
	fmt.Printf("City: '%s'\n", contact.City)
	fmt.Printf("State: '%s'\n", contact.State)
	fmt.Printf("Country: '%s'\n", contact.Country)
}
